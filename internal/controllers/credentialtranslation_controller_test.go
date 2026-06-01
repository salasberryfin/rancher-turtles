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

package controllers_test

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/rancher/turtles/internal/controllers"
	"github.com/rancher/turtles/internal/sync"
	turtlesannotations "github.com/rancher/turtles/util/annotations"
)

var awsStaticIdentityGVK = controllers.AWSClusterStaticIdentityGVK

func newTestScheme() *runtime.Scheme {
	s := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(s)
	_ = apiextensionsv1.AddToScheme(s)

	// Register AWSClusterStaticIdentity as an unstructured kind so the fake client can handle it.
	s.AddKnownTypeWithName(awsStaticIdentityGVK, &unstructured.Unstructured{})
	s.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   awsStaticIdentityGVK.Group,
		Version: awsStaticIdentityGVK.Version,
		Kind:    awsStaticIdentityGVK.Kind + "List",
	}, &unstructured.UnstructuredList{})

	return s
}

// awsCRD returns a minimal CustomResourceDefinition object for AWSClusterStaticIdentity.
// Add this to the fake client to simulate the CRD being installed in the cluster.
func awsCRD() *apiextensionsv1.CustomResourceDefinition {
	return &apiextensionsv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "awsclusterstaticidentities.infrastructure.cluster.x-k8s.io",
		},
	}
}

func newAWSCredentialSecret(name string, accessKey, secretKey string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: sync.RancherCredentialsNamespace,
			Annotations: map[string]string{
				sync.DriverNameAnnotation: sync.AWSDriverName,
			},
		},
		Data: map[string][]byte{
			sync.AWSAccessKeyField: []byte(accessKey),
			sync.AWSSecretKeyField: []byte(secretKey),
		},
	}
}

func newReconciler(cl client.Client) *controllers.RancherCredentialReconciler {
	return &controllers.RancherCredentialReconciler{
		Client:              cl,
		CAPISystemNamespace: "capa-system",
	}
}

func reconcileCredential(g *WithT, r *controllers.RancherCredentialReconciler, credential *corev1.Secret) ctrl.Result {
	result, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      credential.Name,
			Namespace: credential.Namespace,
		},
	})
	g.Expect(err).ToNot(HaveOccurred())

	return result
}

func TestRancherCredentialReconciler_CreatesAWSIdentity(t *testing.T) {
	g := NewWithT(t)
	scheme := newTestScheme()

	credential := newAWSCredentialSecret("cc-test123", "AKIAIOSFODNN7EXAMPLE", "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY")

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(credential, awsCRD()).
		Build()

	r := newReconciler(cl)
	reconcileCredential(g, r, credential)

	// Verify the credentials Secret was created in capa-system.
	credSecret := &corev1.Secret{}
	g.Expect(cl.Get(context.Background(), types.NamespacedName{
		Name:      "cc-test123",
		Namespace: "capa-system",
	}, credSecret)).To(Succeed())

	g.Expect(credSecret.Data).To(HaveKeyWithValue("AccessKeyID", []byte("AKIAIOSFODNN7EXAMPLE")))
	g.Expect(credSecret.Data).To(HaveKeyWithValue("SecretAccessKey", []byte("wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY")))

	// Verify the AWSClusterStaticIdentity was created.
	awsIdentity := &unstructured.Unstructured{}
	awsIdentity.SetGroupVersionKind(awsStaticIdentityGVK)
	g.Expect(cl.Get(context.Background(), types.NamespacedName{Name: "cc-test123"}, awsIdentity)).To(Succeed())

	spec := awsIdentity.Object["spec"].(map[string]interface{})
	g.Expect(spec["secretRef"]).To(Equal("cc-test123"))
	g.Expect(spec["allowedNamespaces"]).To(Equal(map[string]interface{}{}))

	// Verify the Rancher credential was annotated.
	updated := &corev1.Secret{}
	g.Expect(cl.Get(context.Background(), client.ObjectKeyFromObject(credential), updated)).To(Succeed())
	g.Expect(updated.GetAnnotations()).To(HaveKeyWithValue(
		turtlesannotations.AWSClusterStaticIdentityRefAnnotation, "cc-test123"))

	// Verify finalizer was added.
	g.Expect(updated.Finalizers).To(ContainElement(controllers.AWSCredentialFinalizer))
}

func TestRancherCredentialReconciler_SkipsNonAWSCredentials(t *testing.T) {
	g := NewWithT(t)
	scheme := newTestScheme()

	// A secret with no driver annotation (or non-AWS driver).
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "non-aws-cred",
			Namespace: sync.RancherCredentialsNamespace,
			Annotations: map[string]string{
				sync.DriverNameAnnotation: "azure",
			},
		},
	}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(secret).
		Build()

	r := newReconciler(cl)

	_, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      secret.Name,
			Namespace: secret.Namespace,
		},
	})
	g.Expect(err).ToNot(HaveOccurred())

	// No identity should have been created.
	awsIdentity := &unstructured.Unstructured{}
	awsIdentity.SetGroupVersionKind(awsStaticIdentityGVK)
	err = cl.Get(context.Background(), types.NamespacedName{Name: "non-aws-cred"}, awsIdentity)
	g.Expect(err).To(HaveOccurred())
}

func TestRancherCredentialReconciler_SkipsMissingCredentialKeys(t *testing.T) {
	g := NewWithT(t)
	scheme := newTestScheme()

	// A secret with the amazonec2 driver annotation but missing credential keys.
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "incomplete-aws-cred",
			Namespace: sync.RancherCredentialsNamespace,
			Annotations: map[string]string{
				sync.DriverNameAnnotation: sync.AWSDriverName,
			},
		},
		Data: map[string][]byte{
			// Missing AWSSecretKeyField.
			sync.AWSAccessKeyField: []byte("AKIAIOSFODNN7EXAMPLE"),
		},
	}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(secret, awsCRD()).
		Build()

	r := newReconciler(cl)
	reconcileCredential(g, r, secret)

	// No identity should have been created.
	awsIdentity := &unstructured.Unstructured{}
	awsIdentity.SetGroupVersionKind(awsStaticIdentityGVK)
	err := cl.Get(context.Background(), types.NamespacedName{Name: "incomplete-aws-cred"}, awsIdentity)
	g.Expect(err).To(HaveOccurred())
}

func TestRancherCredentialReconciler_DeleteCleansUpResources(t *testing.T) {
	g := NewWithT(t)
	scheme := newTestScheme()

	now := metav1.Now()

	// A credential with a deletion timestamp and the finalizer already set.
	credential := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "cc-delete-me",
			Namespace:         sync.RancherCredentialsNamespace,
			DeletionTimestamp: &now,
			Finalizers:        []string{controllers.AWSCredentialFinalizer},
			Annotations: map[string]string{
				sync.DriverNameAnnotation: sync.AWSDriverName,
			},
		},
	}

	// Pre-existing credentials secret and AWSClusterStaticIdentity.
	credSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cc-delete-me",
			Namespace: "capa-system",
		},
	}

	awsIdentity := &unstructured.Unstructured{}
	awsIdentity.SetGroupVersionKind(awsStaticIdentityGVK)
	awsIdentity.SetName("cc-delete-me")

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(credential, credSecret, awsIdentity).
		Build()

	r := newReconciler(cl)
	reconcileCredential(g, r, credential)

	// Credentials secret should be deleted.
	deletedSecret := &corev1.Secret{}
	err := cl.Get(context.Background(), types.NamespacedName{
		Name:      "cc-delete-me",
		Namespace: "capa-system",
	}, deletedSecret)
	g.Expect(err).To(HaveOccurred())

	// AWSClusterStaticIdentity should be deleted.
	deletedIdentity := &unstructured.Unstructured{}
	deletedIdentity.SetGroupVersionKind(awsStaticIdentityGVK)
	err = cl.Get(context.Background(), types.NamespacedName{Name: "cc-delete-me"}, deletedIdentity)
	g.Expect(err).To(HaveOccurred())

	// The credential itself should also be gone (fake client removes objects once
	// the deletion timestamp is set and all finalizers are removed).
	updatedCredential := &corev1.Secret{}
	err = cl.Get(context.Background(), client.ObjectKeyFromObject(credential), updatedCredential)
	g.Expect(err).To(HaveOccurred())
}

func TestRancherCredentialReconciler_UpdatesCredentials(t *testing.T) {
	g := NewWithT(t)
	scheme := newTestScheme()

	credential := newAWSCredentialSecret("cc-update-test", "OLD_ACCESS_KEY", "OLD_SECRET_KEY")

	existingCredSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cc-update-test",
			Namespace: "capa-system",
		},
		Data: map[string][]byte{
			"AccessKeyID":     []byte("OLD_ACCESS_KEY"),
			"SecretAccessKey": []byte("OLD_SECRET_KEY"),
		},
	}

	existingIdentity := &unstructured.Unstructured{}
	existingIdentity.SetGroupVersionKind(awsStaticIdentityGVK)
	existingIdentity.SetName("cc-update-test")

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(credential, existingCredSecret, existingIdentity, awsCRD()).
		Build()

	// Simulate an update to the credential keys.
	updatedCredential := credential.DeepCopy()
	updatedCredential.Data[sync.AWSAccessKeyField] = []byte("NEW_ACCESS_KEY")
	updatedCredential.Data[sync.AWSSecretKeyField] = []byte("NEW_SECRET_KEY")
	g.Expect(cl.Update(context.Background(), updatedCredential)).To(Succeed())

	r := newReconciler(cl)
	reconcileCredential(g, r, updatedCredential)

	// Verify credentials secret was updated.
	credSecret := &corev1.Secret{}
	g.Expect(cl.Get(context.Background(), types.NamespacedName{
		Name:      "cc-update-test",
		Namespace: "capa-system",
	}, credSecret)).To(Succeed())

	g.Expect(credSecret.Data).To(HaveKeyWithValue("AccessKeyID", []byte("NEW_ACCESS_KEY")))
	g.Expect(credSecret.Data).To(HaveKeyWithValue("SecretAccessKey", []byte("NEW_SECRET_KEY")))
}

func TestRancherCredentialReconciler_SkipsWhenCRDNotAvailable(t *testing.T) {
	g := NewWithT(t)

	// Use the standard scheme but do NOT add the CRD object to the fake store,
	// simulating a cluster where the CRD is not installed.
	scheme := newTestScheme()

	credential := newAWSCredentialSecret("cc-no-crd", "AKIAIOSFODNN7EXAMPLE", "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY")

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(credential).
		Build()

	r := newReconciler(cl)
	result := reconcileCredential(g, r, credential)

	// Should return an empty result (no error, no requeue).
	g.Expect(result).To(Equal(ctrl.Result{}))

	// No finalizer should have been added.
	updated := &corev1.Secret{}
	g.Expect(cl.Get(context.Background(), client.ObjectKeyFromObject(credential), updated)).To(Succeed())
	g.Expect(updated.Finalizers).NotTo(ContainElement(controllers.AWSCredentialFinalizer))

	// No identity reference annotation should have been set.
	g.Expect(updated.GetAnnotations()).NotTo(HaveKey(turtlesannotations.AWSClusterStaticIdentityRefAnnotation))
}

func TestRancherCredentialReconciler_DeleteHandlesMissingCRD(t *testing.T) {
	g := NewWithT(t)

	// Use the standard scheme but do NOT add the CRD object to the fake store,
	// simulating a credential that has a finalizer from a previous translation but
	// the CRD is no longer installed.
	scheme := newTestScheme()

	now := metav1.Now()
	// A credential with a finalizer set (e.g. was previously translated), now being deleted.
	credential := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "cc-delete-no-crd",
			Namespace:         sync.RancherCredentialsNamespace,
			DeletionTimestamp: &now,
			Finalizers:        []string{controllers.AWSCredentialFinalizer},
			Annotations: map[string]string{
				sync.DriverNameAnnotation: sync.AWSDriverName,
			},
		},
	}

	// Pre-existing credentials secret in the CAPI namespace.
	credSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cc-delete-no-crd",
			Namespace: "capa-system",
		},
	}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(credential, credSecret).
		Build()

	r := newReconciler(cl)
	// Must not return an error even though the CRD is unavailable.
	reconcileCredential(g, r, credential)

	// The credentials secret in the CAPI namespace should have been deleted.
	deletedSecret := &corev1.Secret{}
	err := cl.Get(context.Background(), types.NamespacedName{
		Name:      "cc-delete-no-crd",
		Namespace: "capa-system",
	}, deletedSecret)
	g.Expect(err).To(HaveOccurred())

	// The finalizer should have been removed (or the object fully gone).
	updatedCredential := &corev1.Secret{}
	if err := cl.Get(context.Background(), client.ObjectKeyFromObject(credential), updatedCredential); err == nil {
		g.Expect(updatedCredential.Finalizers).NotTo(ContainElement(controllers.AWSCredentialFinalizer))
	}
}
