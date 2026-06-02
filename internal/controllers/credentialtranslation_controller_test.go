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
	"encoding/base64"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	turtlesv1 "github.com/rancher/turtles/api/v1alpha1"
	"github.com/rancher/turtles/internal/sync"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/cluster-api-operator/controller"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/controller-runtime/pkg/client"
	. "sigs.k8s.io/controller-runtime/pkg/envtest/komega"
)

// newCredentialReconciler returns a CAPIProviderReconciler wired to the shared test
// environment client.  The caller must update r.GenericProviderReconciler.Provider
// before each call to syncSecrets so that it holds the most recent version of the
// provider object (including its ResourceVersion).
func newCredentialReconciler(provider *turtlesv1.CAPIProvider) *CAPIProviderReconciler {
	return &CAPIProviderReconciler{
		Client: cl,
		GenericProviderReconciler: controller.GenericProviderReconciler{
			Provider:     provider,
			ProviderList: &turtlesv1.CAPIProviderList{},
			Client:       cl,
			Config:       testEnv.GetConfig(),
		},
	}
}

// runSyncSecrets fetches the latest version of provider, sets it on r and calls
// r.syncSecrets.  It is designed to be called repeatedly inside an Eventually
// block so that transient conflicts are retried automatically.
func runSyncSecrets(g Gomega, r *CAPIProviderReconciler, provider *turtlesv1.CAPIProvider) error {
	g.Expect(cl.Get(ctx, client.ObjectKeyFromObject(provider), provider)).ToNot(HaveOccurred())
	r.GenericProviderReconciler.Provider = provider
	_, err := r.syncSecrets(ctx)

	return err
}

var _ = Describe("Credential Translation", func() {
	var (
		ns            *corev1.Namespace
		rancherSecret *corev1.Secret
	)

	BeforeEach(func() {
		SetClient(testEnv)
		SetContext(ctx)

		var err error
		ns, err = testEnv.CreateNamespace(ctx, "credtranslation")
		Expect(err).ToNot(HaveOccurred())

		// Ensure the shared credentials namespace exists; it may already have been
		// created by another test so we treat AlreadyExists as success.
		globalDataNs := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: sync.RancherCredentialsNamespace}}
		if createErr := cl.Create(ctx, globalDataNs); createErr != nil && !apierrors.IsAlreadyExists(createErr) {
			Expect(createErr).ToNot(HaveOccurred())
		}

		rancherSecret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "cc-",
				Namespace:    sync.RancherCredentialsNamespace,
			},
		}
	})

	AfterEach(func() {
		// Only delete the Rancher secret if it was actually created.
		if rancherSecret != nil && rancherSecret.UID != "" {
			Expect(testEnv.Cleanup(ctx, rancherSecret)).ToNot(HaveOccurred())
		}
		Expect(testEnv.Cleanup(ctx, ns)).ToNot(HaveOccurred())
	})

	It("Should map DigitalOcean credentials to the provider secret", func() {
		provider := &turtlesv1.CAPIProvider{
			ObjectMeta: metav1.ObjectMeta{Name: "digitalocean", Namespace: ns.Name},
			Spec: turtlesv1.CAPIProviderSpec{
				Type: turtlesv1.Infrastructure,
				Credentials: &turtlesv1.Credentials{
					RancherCloudCredential: "do-cred",
				},
			},
		}
		Expect(cl.Create(ctx, provider)).ToNot(HaveOccurred())

		rancherSecret.Annotations = map[string]string{
			sync.NameAnnotation:       "do-cred",
			sync.DriverNameAnnotation: "digitalocean",
		}
		rancherSecret.Data = map[string][]byte{
			"digitaloceancredentialConfig-accessToken": []byte("my-do-token"),
		}
		Expect(cl.Create(ctx, rancherSecret)).ToNot(HaveOccurred())

		r := newCredentialReconciler(provider)
		providerSecret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: provider.Name, Namespace: ns.Name}}

		Eventually(func(g Gomega) {
			g.Expect(runSyncSecrets(g, r, provider)).ToNot(HaveOccurred())
			g.Expect(cl.Get(ctx, client.ObjectKeyFromObject(providerSecret), providerSecret)).ToNot(HaveOccurred())
			g.Expect(providerSecret.Data).To(HaveKeyWithValue("DIGITALOCEAN_ACCESS_TOKEN", []byte("my-do-token")))
			g.Expect(providerSecret.Data).To(HaveKeyWithValue(
				"DO_B64ENCODED_CREDENTIALS",
				[]byte(base64.StdEncoding.EncodeToString([]byte("my-do-token"))),
			))
		}).WithTimeout(10 * time.Second).Should(Succeed())

		Expect(conditions.IsTrue(provider, string(turtlesv1.RancherCredentialsSecretCondition))).To(BeTrue())
	})

	It("Should map AWS credentials to the provider secret", func() {
		provider := &turtlesv1.CAPIProvider{
			ObjectMeta: metav1.ObjectMeta{Name: "aws", Namespace: ns.Name},
			Spec: turtlesv1.CAPIProviderSpec{
				Type: turtlesv1.Infrastructure,
				Credentials: &turtlesv1.Credentials{
					RancherCloudCredential: "aws-cred",
				},
			},
		}
		Expect(cl.Create(ctx, provider)).ToNot(HaveOccurred())

		rancherSecret.Annotations = map[string]string{
			sync.NameAnnotation:       "aws-cred",
			sync.DriverNameAnnotation: "aws",
		}
		rancherSecret.StringData = map[string]string{
			"amazonec2credentialConfig-accessKey":     "access-key",
			"amazonec2credentialConfig-secretKey":     "secret-key",
			"amazonec2credentialConfig-defaultRegion": "us-east-1",
		}
		Expect(cl.Create(ctx, rancherSecret)).ToNot(HaveOccurred())

		r := newCredentialReconciler(provider)
		providerSecret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: provider.Name, Namespace: ns.Name}}

		Eventually(func(g Gomega) {
			g.Expect(runSyncSecrets(g, r, provider)).ToNot(HaveOccurred())
			g.Expect(cl.Get(ctx, client.ObjectKeyFromObject(providerSecret), providerSecret)).ToNot(HaveOccurred())
			g.Expect(providerSecret.Data).To(HaveKeyWithValue("AWS_ACCESS_KEY_ID", []byte("access-key")))
			g.Expect(providerSecret.Data).To(HaveKeyWithValue("AWS_SECRET_ACCESS_KEY", []byte("secret-key")))
			g.Expect(providerSecret.Data).To(HaveKeyWithValue("AWS_REGION", []byte("us-east-1")))
			g.Expect(providerSecret.Data).To(HaveKey("AWS_B64ENCODED_CREDENTIALS"))
			g.Expect(providerSecret.Data["AWS_B64ENCODED_CREDENTIALS"]).ToNot(BeEmpty())
		}).WithTimeout(10 * time.Second).Should(Succeed())

		Expect(conditions.IsTrue(provider, string(turtlesv1.RancherCredentialsSecretCondition))).To(BeTrue())
	})

	It("Should map Azure credentials to the provider secret", func() {
		provider := &turtlesv1.CAPIProvider{
			ObjectMeta: metav1.ObjectMeta{Name: "azure", Namespace: ns.Name},
			Spec: turtlesv1.CAPIProviderSpec{
				Type: turtlesv1.Infrastructure,
				Credentials: &turtlesv1.Credentials{
					RancherCloudCredential: "azure-cred",
				},
			},
		}
		Expect(cl.Create(ctx, provider)).ToNot(HaveOccurred())

		rancherSecret.Annotations = map[string]string{
			sync.NameAnnotation:       "azure-cred",
			sync.DriverNameAnnotation: "azure",
		}
		rancherSecret.Data = map[string][]byte{
			"azurecredentialConfig-subscriptionId": []byte("sub-123"),
			"azurecredentialConfig-clientId":       []byte("client-456"),
			"azurecredentialConfig-clientSecret":   []byte("secret-789"),
			"azurecredentialConfig-tenantId":       []byte("tenant-abc"),
		}
		Expect(cl.Create(ctx, rancherSecret)).ToNot(HaveOccurred())

		r := newCredentialReconciler(provider)
		providerSecret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: provider.Name, Namespace: ns.Name}}

		Eventually(func(g Gomega) {
			g.Expect(runSyncSecrets(g, r, provider)).ToNot(HaveOccurred())
			g.Expect(cl.Get(ctx, client.ObjectKeyFromObject(providerSecret), providerSecret)).ToNot(HaveOccurred())
			g.Expect(providerSecret.Data).To(HaveKeyWithValue("AZURE_SUBSCRIPTION_ID", []byte("sub-123")))
			g.Expect(providerSecret.Data).To(HaveKeyWithValue("AZURE_CLIENT_ID", []byte("client-456")))
			g.Expect(providerSecret.Data).To(HaveKeyWithValue("AZURE_CLIENT_SECRET", []byte("secret-789")))
			g.Expect(providerSecret.Data).To(HaveKeyWithValue("AZURE_TENANT_ID", []byte("tenant-abc")))
			g.Expect(providerSecret.Data).To(HaveKeyWithValue(
				"AZURE_SUBSCRIPTION_ID_B64",
				[]byte(base64.StdEncoding.EncodeToString([]byte("sub-123"))),
			))
			g.Expect(providerSecret.Data).To(HaveKeyWithValue(
				"AZURE_CLIENT_ID_B64",
				[]byte(base64.StdEncoding.EncodeToString([]byte("client-456"))),
			))
			g.Expect(providerSecret.Data).To(HaveKeyWithValue(
				"AZURE_CLIENT_SECRET_B64",
				[]byte(base64.StdEncoding.EncodeToString([]byte("secret-789"))),
			))
			g.Expect(providerSecret.Data).To(HaveKeyWithValue(
				"AZURE_TENANT_ID_B64",
				[]byte(base64.StdEncoding.EncodeToString([]byte("tenant-abc"))),
			))
		}).WithTimeout(10 * time.Second).Should(Succeed())

		Expect(conditions.IsTrue(provider, string(turtlesv1.RancherCredentialsSecretCondition))).To(BeTrue())
	})

	It("Should map GCP credentials to the provider secret with double base64 encoding", func() {
		provider := &turtlesv1.CAPIProvider{
			ObjectMeta: metav1.ObjectMeta{Name: "gcp", Namespace: ns.Name},
			Spec: turtlesv1.CAPIProviderSpec{
				Type: turtlesv1.Infrastructure,
				Credentials: &turtlesv1.Credentials{
					RancherCloudCredential: "gcp-cred",
				},
			},
		}
		Expect(cl.Create(ctx, provider)).ToNot(HaveOccurred())

		rancherSecret.Annotations = map[string]string{
			sync.NameAnnotation:       "gcp-cred",
			sync.DriverNameAnnotation: "gcp",
		}
		// Rancher stores GCP credentials as a base64-encoded JSON blob; the sync
		// layer encodes the raw bytes once more, producing double base64.
		gcpJSON := `{"type":"service_account"}`
		rancherSecret.Data = map[string][]byte{
			"googlecredentialConfig-authEncodedJson": []byte(gcpJSON),
		}
		Expect(cl.Create(ctx, rancherSecret)).ToNot(HaveOccurred())

		r := newCredentialReconciler(provider)
		providerSecret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: provider.Name, Namespace: ns.Name}}

		Eventually(func(g Gomega) {
			g.Expect(runSyncSecrets(g, r, provider)).ToNot(HaveOccurred())
			g.Expect(cl.Get(ctx, client.ObjectKeyFromObject(providerSecret), providerSecret)).ToNot(HaveOccurred())
			g.Expect(providerSecret.Data).To(HaveKeyWithValue(
				"GCP_B64ENCODED_CREDENTIALS",
				[]byte(base64.StdEncoding.EncodeToString([]byte(gcpJSON))),
			))
		}).WithTimeout(10 * time.Second).Should(Succeed())

		Expect(conditions.IsTrue(provider, string(turtlesv1.RancherCredentialsSecretCondition))).To(BeTrue())
	})

	It("Should map vSphere credentials to the provider secret", func() {
		// The Rancher driver name for vSphere is "vmwarevsphere", while the CAPI
		// provider name is "vsphere".  The sync layer handles the translation.
		provider := &turtlesv1.CAPIProvider{
			ObjectMeta: metav1.ObjectMeta{Name: "vsphere", Namespace: ns.Name},
			Spec: turtlesv1.CAPIProviderSpec{
				Type: turtlesv1.Infrastructure,
				Credentials: &turtlesv1.Credentials{
					RancherCloudCredential: "vsphere-cred",
				},
			},
		}
		Expect(cl.Create(ctx, provider)).ToNot(HaveOccurred())

		rancherSecret.Annotations = map[string]string{
			sync.NameAnnotation:       "vsphere-cred",
			sync.DriverNameAnnotation: "vmwarevsphere",
		}
		rancherSecret.Data = map[string][]byte{
			"vmwarevspherecredentialConfig-password": []byte("secret-pass"),
			"vmwarevspherecredentialConfig-username": []byte("admin-user"),
		}
		Expect(cl.Create(ctx, rancherSecret)).ToNot(HaveOccurred())

		r := newCredentialReconciler(provider)
		providerSecret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: provider.Name, Namespace: ns.Name}}

		Eventually(func(g Gomega) {
			g.Expect(runSyncSecrets(g, r, provider)).ToNot(HaveOccurred())
			g.Expect(cl.Get(ctx, client.ObjectKeyFromObject(providerSecret), providerSecret)).ToNot(HaveOccurred())
			g.Expect(providerSecret.Data).To(HaveKeyWithValue("VSPHERE_PASSWORD", []byte("secret-pass")))
			g.Expect(providerSecret.Data).To(HaveKeyWithValue("VSPHERE_USERNAME", []byte("admin-user")))
		}).WithTimeout(10 * time.Second).Should(Succeed())

		Expect(conditions.IsTrue(provider, string(turtlesv1.RancherCredentialsSecretCondition))).To(BeTrue())
	})

	It("Should set a failure condition when the Rancher credential secret does not exist", func() {
		provider := &turtlesv1.CAPIProvider{
			ObjectMeta: metav1.ObjectMeta{Name: "digitalocean", Namespace: ns.Name},
			Spec: turtlesv1.CAPIProviderSpec{
				Type: turtlesv1.Infrastructure,
				Credentials: &turtlesv1.Credentials{
					RancherCloudCredential: "non-existent-cred",
				},
			},
		}
		Expect(cl.Create(ctx, provider)).ToNot(HaveOccurred())

		r := newCredentialReconciler(provider)

		// syncSecrets returns an error when the Rancher secret cannot be located,
		// and the failure condition is set on the in-memory provider object.
		Eventually(func(g Gomega) {
			g.Expect(cl.Get(ctx, client.ObjectKeyFromObject(provider), provider)).ToNot(HaveOccurred())
			r.GenericProviderReconciler.Provider = provider
			r.syncSecrets(ctx) //nolint:errcheck // error is expected; we care about the condition, not the returned error
			g.Expect(conditions.IsFalse(provider, string(turtlesv1.RancherCredentialsSecretCondition))).To(BeTrue())
			g.Expect(conditions.GetMessage(provider, string(turtlesv1.RancherCredentialsSecretCondition))).
				To(ContainSubstring("key not found"))
		}).WithTimeout(10 * time.Second).Should(Succeed())
	})

	It("Should set a failure condition when required credential keys are absent from the Rancher secret", func() {
		provider := &turtlesv1.CAPIProvider{
			ObjectMeta: metav1.ObjectMeta{Name: "aws", Namespace: ns.Name},
			Spec: turtlesv1.CAPIProviderSpec{
				Type: turtlesv1.Infrastructure,
				Credentials: &turtlesv1.Credentials{
					RancherCloudCredential: "aws-cred",
				},
			},
		}
		Expect(cl.Create(ctx, provider)).ToNot(HaveOccurred())

		// Rancher secret exists but does not contain the expected AWS keys.
		rancherSecret.Annotations = map[string]string{
			sync.NameAnnotation:       "aws-cred",
			sync.DriverNameAnnotation: "aws",
		}
		rancherSecret.Data = map[string][]byte{
			"unrelated-key": []byte("some-value"),
		}
		Expect(cl.Create(ctx, rancherSecret)).ToNot(HaveOccurred())

		r := newCredentialReconciler(provider)

		Eventually(func(g Gomega) {
			g.Expect(runSyncSecrets(g, r, provider)).ToNot(HaveOccurred())
			g.Expect(conditions.IsFalse(provider, string(turtlesv1.RancherCredentialsSecretCondition))).To(BeTrue())
			g.Expect(conditions.GetMessage(provider, string(turtlesv1.RancherCredentialsSecretCondition))).
				To(ContainSubstring("key not found: amazonec2credentialConfig-accessKey"))
		}).WithTimeout(10 * time.Second).Should(Succeed())
	})

	It("Should map credentials when the Rancher secret is referenced by namespace and name", func() {
		provider := &turtlesv1.CAPIProvider{
			ObjectMeta: metav1.ObjectMeta{Name: "digitalocean", Namespace: ns.Name},
			Spec: turtlesv1.CAPIProviderSpec{
				Type: turtlesv1.Infrastructure,
				Credentials: &turtlesv1.Credentials{
					// Direct namespace:name reference — no annotation lookup required.
					RancherCloudCredentialNamespaceName: ns.Name + ":do-secret",
				},
			},
		}
		Expect(cl.Create(ctx, provider)).ToNot(HaveOccurred())

		// Place the Rancher secret in the provider's own namespace (not cattle-global-data).
		directSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "do-secret",
				Namespace: ns.Name,
			},
			Data: map[string][]byte{
				"digitaloceancredentialConfig-accessToken": []byte("direct-token"),
			},
		}
		Expect(cl.Create(ctx, directSecret)).ToNot(HaveOccurred())

		r := newCredentialReconciler(provider)
		providerSecret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: provider.Name, Namespace: ns.Name}}

		Eventually(func(g Gomega) {
			g.Expect(runSyncSecrets(g, r, provider)).ToNot(HaveOccurred())
			g.Expect(cl.Get(ctx, client.ObjectKeyFromObject(providerSecret), providerSecret)).ToNot(HaveOccurred())
			g.Expect(providerSecret.Data).To(HaveKeyWithValue("DIGITALOCEAN_ACCESS_TOKEN", []byte("direct-token")))
			g.Expect(providerSecret.Data).To(HaveKeyWithValue(
				"DO_B64ENCODED_CREDENTIALS",
				[]byte(base64.StdEncoding.EncodeToString([]byte("direct-token"))),
			))
		}).WithTimeout(10 * time.Second).Should(Succeed())

		Expect(conditions.IsTrue(provider, string(turtlesv1.RancherCredentialsSecretCondition))).To(BeTrue())
	})
})
