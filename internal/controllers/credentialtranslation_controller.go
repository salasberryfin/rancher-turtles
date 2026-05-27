/*
Copyright © 2023 - 2024 SUSE LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/rancher/turtles/internal/sync"
	turtlesannotations "github.com/rancher/turtles/util/annotations"
)

const (
	// awsCredentialFinalizer is the finalizer added to Rancher Cloud Credentials to ensure
	// cleanup of the derived AWSClusterStaticIdentity when the credential is deleted.
	awsCredentialFinalizer = "cloudcredential.cattle.io/aws-identity-finalizer"

	// awsDriverAnnotationValue is the value of the driver annotation that identifies AWS credentials.
	awsDriverAnnotationValue = "amazonec2"

	// awsStaticIdentityGroup is the API group for CAPA infrastructure resources.
	awsStaticIdentityGroup = "infrastructure.cluster.x-k8s.io"

	// awsStaticIdentityVersion is the API version for CAPA infrastructure resources.
	awsStaticIdentityVersion = "v1beta2"

	// awsStaticIdentityKind is the kind for the CAPA static identity resource.
	awsStaticIdentityKind = "AWSClusterStaticIdentity"

	// awsCredentialSecretKeyAccessKeyID is the key in the CAPA credentials secret for the AWS access key ID.
	awsCredentialSecretKeyAccessKeyID = "AccessKeyID"

	// awsCredentialSecretKeySecretAccessKey is the key in the CAPA credentials secret for the AWS secret access key.
	awsCredentialSecretKeySecretAccessKey = "SecretAccessKey"

	// rancherAWSAccessKeyField is the field name in Rancher Cloud Credential secrets for the AWS access key ID.
	rancherAWSAccessKeyField = "amazonec2credentialConfig-accessKey"

	// rancherAWSSecretKeyField is the field name in Rancher Cloud Credential secrets for the AWS secret key.
	rancherAWSSecretKeyField = "amazonec2credentialConfig-secretKey"
)

// RancherCredentialReconciler reconciles Rancher Cloud Credentials in the cattle-global-data
// namespace into CAPA-specific AWSClusterStaticIdentity resources. This enables users to reuse
// Rancher Cloud Credentials when provisioning AWS clusters with CAPI/CAPA.
type RancherCredentialReconciler struct {
	Client client.Client

	// CAPISystemNamespace is the namespace where the CAPA controller is installed and where
	// the credentials secret will be created. Defaults to "capa-system".
	CAPISystemNamespace string
}

// SetupWithManager sets up the controller with the Manager.
func (r *RancherCredentialReconciler) SetupWithManager(_ context.Context, mgr ctrl.Manager, options controller.Options) error {
	if r.CAPISystemNamespace == "" {
		r.CAPISystemNamespace = "capa-system"
	}

	if err := ctrl.NewControllerManagedBy(mgr).
		Named("rancher-credential-translation").
		For(&corev1.Secret{}).
		WithOptions(options).
		WithEventFilter(predicate.NewPredicateFuncs(func(obj client.Object) bool {
			return obj.GetNamespace() == sync.RancherCredentialsNamespace &&
				obj.GetAnnotations()[sync.DriverNameAnnotation] == awsDriverAnnotationValue
		})).
		Complete(r); err != nil {
		return fmt.Errorf("creating RancherCredential translation controller: %w", err)
	}

	return nil
}

//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=awsclusterstaticidentities,verbs=get;list;watch;create;update;patch;delete

// Reconcile watches Rancher Cloud Credentials for AWS and translates them into
// AWSClusterStaticIdentity resources for use with CAPA.
func (r *RancherCredentialReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	credential := &corev1.Secret{}
	if err := r.Client.Get(ctx, req.NamespacedName, credential); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Only process AWS (amazonec2) cloud credentials.
	if credential.GetAnnotations()[sync.DriverNameAnnotation] != awsDriverAnnotationValue {
		return ctrl.Result{}, nil
	}

	if !credential.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, credential)
	}

	return r.reconcileNormal(ctx, credential)
}

// reconcileNormal handles creation and updates of the AWSClusterStaticIdentity for a given
// Rancher Cloud Credential.
func (r *RancherCredentialReconciler) reconcileNormal(ctx context.Context, credential *corev1.Secret) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Add finalizer to the credential so we can clean up derived resources on deletion.
	if !controllerutil.ContainsFinalizer(credential, awsCredentialFinalizer) {
		patch := client.MergeFrom(credential.DeepCopy())
		controllerutil.AddFinalizer(credential, awsCredentialFinalizer)

		if err := r.Client.Patch(ctx, credential, patch); err != nil {
			return ctrl.Result{}, fmt.Errorf("adding finalizer to credential %s: %w", client.ObjectKeyFromObject(credential), err)
		}

		log.Info("Added finalizer to AWS credential", "credential", client.ObjectKeyFromObject(credential))
	}

	// Extract the AWS credentials from the Rancher secret.
	accessKeyID := string(credential.Data[rancherAWSAccessKeyField])
	secretAccessKey := string(credential.Data[rancherAWSSecretKeyField])

	if accessKeyID == "" || secretAccessKey == "" {
		log.Info("AWS credential secret is missing required keys, skipping",
			"credential", client.ObjectKeyFromObject(credential),
			"missingKeys", fmt.Sprintf("%s or %s", rancherAWSAccessKeyField, rancherAWSSecretKeyField))

		return ctrl.Result{}, nil
	}

	identityName := credential.Name

	// Ensure the capa-system namespace exists; create it if necessary.
	if err := r.ensureNamespace(ctx, r.CAPISystemNamespace); err != nil {
		return ctrl.Result{}, err
	}

	// Create or update the credentials Secret in the CAPI system namespace.
	if err := r.reconcileCredentialSecret(ctx, identityName, accessKeyID, secretAccessKey); err != nil {
		return ctrl.Result{}, err
	}

	// Create or update the AWSClusterStaticIdentity referencing the credentials secret.
	if err := r.reconcileAWSClusterStaticIdentity(ctx, identityName); err != nil {
		return ctrl.Result{}, err
	}

	// Annotate the original Rancher credential with a reference to the AWSClusterStaticIdentity.
	if credential.GetAnnotations()[turtlesannotations.AWSClusterStaticIdentityRefAnnotation] != identityName {
		patch := client.MergeFrom(credential.DeepCopy())
		annotations := credential.GetAnnotations()
		if annotations == nil {
			annotations = map[string]string{}
		}

		annotations[turtlesannotations.AWSClusterStaticIdentityRefAnnotation] = identityName
		credential.SetAnnotations(annotations)

		if err := r.Client.Patch(ctx, credential, patch); err != nil {
			return ctrl.Result{}, fmt.Errorf("annotating credential %s with identity reference: %w", client.ObjectKeyFromObject(credential), err)
		}

		log.Info("Annotated AWS credential with AWSClusterStaticIdentity reference",
			"credential", client.ObjectKeyFromObject(credential),
			"identity", identityName)
	}

	log.Info("Successfully reconciled AWS credential translation",
		"credential", client.ObjectKeyFromObject(credential),
		"identity", identityName)

	return ctrl.Result{}, nil
}

// reconcileDelete cleans up the AWSClusterStaticIdentity and its credentials secret when
// the Rancher Cloud Credential is deleted.
func (r *RancherCredentialReconciler) reconcileDelete(ctx context.Context, credential *corev1.Secret) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	if !controllerutil.ContainsFinalizer(credential, awsCredentialFinalizer) {
		return ctrl.Result{}, nil
	}

	identityName := credential.Name
	// Delete the AWSClusterStaticIdentity.
	awsIdentity := r.awsClusterStaticIdentity(identityName)
	if err := r.Client.Delete(ctx, awsIdentity); err != nil && !apierrors.IsNotFound(err) {
		return ctrl.Result{}, fmt.Errorf("deleting AWSClusterStaticIdentity %s: %w", identityName, err)
	}

	log.Info("Deleted AWSClusterStaticIdentity", "name", identityName)

	// Delete the credentials Secret from the CAPI system namespace.
	credSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      identityName,
			Namespace: r.CAPISystemNamespace,
		},
	}

	if err := r.Client.Delete(ctx, credSecret); err != nil && !apierrors.IsNotFound(err) {
		return ctrl.Result{}, fmt.Errorf("deleting credentials secret %s/%s: %w", r.CAPISystemNamespace, identityName, err)
	}

	log.Info("Deleted credentials secret", "namespace", r.CAPISystemNamespace, "name", identityName)

	// Remove the finalizer so the credential can be garbage-collected.
	patch := client.MergeFrom(credential.DeepCopy())
	controllerutil.RemoveFinalizer(credential, awsCredentialFinalizer)

	if err := r.Client.Patch(ctx, credential, patch); err != nil {
		return ctrl.Result{}, fmt.Errorf("removing finalizer from credential %s: %w", client.ObjectKeyFromObject(credential), err)
	}

	return ctrl.Result{}, nil
}

// reconcileCredentialSecret creates or updates the credentials Secret in the CAPI system namespace.
func (r *RancherCredentialReconciler) reconcileCredentialSecret(ctx context.Context, name, accessKeyID, secretAccessKey string) error {
	log := log.FromContext(ctx)

	credSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: r.CAPISystemNamespace,
		},
	}

	result, err := controllerutil.CreateOrUpdate(ctx, r.Client, credSecret, func() error {
		credSecret.Data = map[string][]byte{
			awsCredentialSecretKeyAccessKeyID:     []byte(accessKeyID),
			awsCredentialSecretKeySecretAccessKey: []byte(secretAccessKey),
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("creating/updating credentials secret %s/%s: %w", r.CAPISystemNamespace, name, err)
	}

	log.V(4).Info("Reconciled credentials secret", "namespace", r.CAPISystemNamespace, "name", name, "result", result)

	return nil
}

// reconcileAWSClusterStaticIdentity creates or updates the AWSClusterStaticIdentity referencing
// the credentials secret.
func (r *RancherCredentialReconciler) reconcileAWSClusterStaticIdentity(ctx context.Context, name string) error {
	log := log.FromContext(ctx)

	awsIdentity := r.awsClusterStaticIdentity(name)

	result, err := controllerutil.CreateOrUpdate(ctx, r.Client, awsIdentity, func() error {
		awsIdentity.Object["spec"] = map[string]interface{}{
			"secretRef": name,
			// allowedNamespaces with an empty list permits all namespaces to use this identity.
			"allowedNamespaces": map[string]interface{}{
				"list": []interface{}{},
			},
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("creating/updating AWSClusterStaticIdentity %s: %w", name, err)
	}

	log.V(4).Info("Reconciled AWSClusterStaticIdentity", "name", name, "result", result)

	return nil
}

// ensureNamespace creates the given namespace if it does not exist.
func (r *RancherCredentialReconciler) ensureNamespace(ctx context.Context, name string) error {
	ns := &corev1.Namespace{}
	if err := r.Client.Get(ctx, client.ObjectKey{Name: name}, ns); err != nil {
		if !apierrors.IsNotFound(err) {
			return fmt.Errorf("getting namespace %s: %w", name, err)
		}

		ns = &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name}}
		if err := r.Client.Create(ctx, ns); err != nil && !apierrors.IsAlreadyExists(err) {
			return fmt.Errorf("creating namespace %s: %w", name, err)
		}
	}

	return nil
}

// awsClusterStaticIdentity returns an unstructured AWSClusterStaticIdentity with the given name.
func (r *RancherCredentialReconciler) awsClusterStaticIdentity(name string) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   awsStaticIdentityGroup,
		Version: awsStaticIdentityVersion,
		Kind:    awsStaticIdentityKind,
	})
	obj.SetName(name)

	return obj
}
