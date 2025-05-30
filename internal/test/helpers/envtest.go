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

package helpers

import (
	"context"
	"fmt"
	"go/build"
	"os"
	"path"
	"path/filepath"
	"regexp"
	goruntime "runtime"
	"strings"

	"github.com/onsi/ginkgo/v2"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/komega"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"

	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	logf "sigs.k8s.io/cluster-api/cmd/clusterctl/log"

	turtlesv1 "github.com/rancher/turtles/api/v1alpha1"
)

var root string

func init() {
	klog.InitFlags(nil)
	// additionally force all the controllers to use the Ginkgo logger.
	ctrl.SetLogger(klog.Background())
	logf.SetLogger(klog.Background())
	// add logger for ginkgo
	klog.SetOutput(ginkgo.GinkgoWriter)

	// Calculate the scheme.
	utilruntime.Must(apiextensionsv1.AddToScheme(clientgoscheme.Scheme))
	utilruntime.Must(admissionv1.AddToScheme(clientgoscheme.Scheme))
	utilruntime.Must(clusterv1.AddToScheme(clientgoscheme.Scheme))
	utilruntime.Must(turtlesv1.AddToScheme(clientgoscheme.Scheme))

	// Get the root of the current file to use in CRD paths.
	_, filename, _, _ := goruntime.Caller(0) //nolint
	root = path.Join(path.Dir(filename), "..", "..", "..")
}

// TestEnvironmentConfiguration is a wrapper configuration for envtest.
type TestEnvironmentConfiguration struct {
	env *envtest.Environment
}

// TestEnvironment encapsulates a Kubernetes local test environment.
type TestEnvironment struct {
	manager.Manager
	client.Client
	Config *rest.Config
	env    *envtest.Environment
	cancel context.CancelFunc
}

// Cleanup deletes all the given objects.
func (t *TestEnvironment) Cleanup(ctx context.Context, objs ...client.Object) error {
	errs := []error{}

	for _, o := range objs {
		err := t.Delete(ctx, o)
		if apierrors.IsNotFound(err) {
			continue
		}

		errs = append(errs, err)
	}

	return kerrors.NewAggregate(errs)
}

// CreateNamespace creates a new namespace with a generated name.
func (t *TestEnvironment) CreateNamespace(ctx context.Context, generateName string) (*corev1.Namespace, error) {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: generateName + "-",
			Labels: map[string]string{
				"testenv/original-name": generateName,
			},
		},
	}
	if err := t.Create(ctx, ns); err != nil {
		return nil, err
	}

	return ns, nil
}

// CreateNamespaceWithName creates a new namespace with a given name.
func (t *TestEnvironment) CreateNamespaceWithName(ctx context.Context, name string) (*corev1.Namespace, error) {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	if err := t.Create(ctx, ns); err != nil {
		return nil, err
	}

	return ns, nil
}

// NewTestEnvironmentConfiguration creates a new test environment configuration for running tests.
func NewTestEnvironmentConfiguration(crdDirectoryPaths ...string) *TestEnvironmentConfiguration {
	resolvedCrdDirectoryPaths := []string{}

	for _, group := range crdDirectoryPaths {
		resolvedCrdDirectoryPaths = append(
			resolvedCrdDirectoryPaths,
			path.Join([]string{root, group}...),
		)
	}

	resolvedCrdDirectoryPaths = append(
		resolvedCrdDirectoryPaths,
		getFilePathToAPI(root, "sigs.k8s.io", "cluster-api", "config/crd/bases"),
		getFilePathToAPI(root, "sigs.k8s.io", "cluster-api-operator", "config/crd/bases"),
	)

	return &TestEnvironmentConfiguration{
		env: &envtest.Environment{
			ErrorIfCRDPathMissing: true,
			CRDDirectoryPaths:     resolvedCrdDirectoryPaths,
		},
	}
}

// Build creates a new environment spinning up a local api-server.
// This function should be called only once for each package you're running tests within,
// usually the environment is initialized in a suite_test.go file within a `BeforeSuite` ginkgo block.
func (t *TestEnvironmentConfiguration) Build() (*TestEnvironment, error) {
	if _, err := t.env.Start(); err != nil {
		panic(err)
	}

	options := manager.Options{
		Scheme: clientgoscheme.Scheme,
		Metrics: server.Options{
			BindAddress: "0",
		},
		Client: client.Options{
			Cache: &client.CacheOptions{
				DisableFor: []client.Object{
					&turtlesv1.ClusterctlConfig{},
				},
			},
		},
	}

	mgr, err := ctrl.NewManager(t.env.Config, options)
	if err != nil {
		klog.Fatalf("Failed to start testenv manager: %v", err)
	}

	komega.SetClient(mgr.GetClient())

	return &TestEnvironment{
		Manager: mgr,
		Client:  mgr.GetClient(),
		Config:  mgr.GetConfig(),
		env:     t.env,
	}, nil
}

// StartManager starts the test controller against the local API server.
func (t *TestEnvironment) StartManager(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	t.cancel = cancel

	return t.Start(ctx)
}

// Stop stops the test environment.
func (t *TestEnvironment) Stop() error {
	t.cancel()

	return t.env.Stop()
}

// KeyAs is an opposite of client.ObjectKeyFromObject, to construct a new client.Object from provider namespace/name.
// Example:
//
//	emptyCluster := testEnv.KeyAs(client.ObjectKeyFromObject(rancherCluster), &provisioningv1.Cluster{})
//	Expect(cl.Get(ctx, emptyCluster)).To(Succeed())
func (t *TestEnvironment) KeyAs(key client.ObjectKey, to client.Object) client.Object {
	to.SetName(key.Name)
	to.SetNamespace(key.Namespace)

	return to
}

// GetAs is a wrapper on komega.Object helper, allowing to construct a new object from an existing object metadata
// and match for object state in kubernetes cluster.
// Example:
//
//	 // Expecting cluster to be present in the kubernetes API server and be stored in &provisioningv1.Cluster{}.
//	 // rancherCluster will not be modified during this operation
//		g.Eventually(testEnv.GetKeyAs(rancherCluster, &provisioningv1.Cluster{})).ShouldNot(BeNil())
func (t *TestEnvironment) GetAs(obj, as client.Object) func() (client.Object, error) {
	return komega.Object(t.KeyAs(client.ObjectKeyFromObject(obj), as))
}

// GetKeyAs is a wrapper on komega.Object helper, allowing to construct a new object from a key
// and match for object state in kubernetes cluster.
// Example:
//
//	 // Expecting cluster to be present in the kubernetes API server and be stored in &provisioningv1.Cluster{}
//		rancherClusterKey := types.NamespaceName{Name: "cluster", Namespace: "default"}
//		g.Eventually(testEnv.GetKeyAs(rancherClusterKey, &provisioningv1.Cluster{})).ShouldNot(BeNil())
func (t *TestEnvironment) GetKeyAs(key client.ObjectKey, as client.Object) func() (client.Object, error) {
	return komega.Object(t.KeyAs(key, as))
}

func getFilePathToAPI(root, org, pkg, apis string) string {
	modBits, err := os.ReadFile(filepath.Join(root, "go.mod")) //nolint:gosec
	if err != nil {
		return ""
	}

	var packageVersion string

	packageVersionRegex := regexp.MustCompile(fmt.Sprintf(`^(.*)%s/%s *v(.+)`, org, pkg))

	for _, line := range strings.Split(string(modBits), "\n") {
		matches := packageVersionRegex.FindStringSubmatch(line)
		if len(matches) > 0 {
			packageVersion = matches[2]
			break
		}
	}

	if packageVersion == "" {
		return ""
	}

	gopath := envOr("GOPATH", build.Default.GOPATH)

	return filepath.Join(gopath, "pkg", "mod", org, fmt.Sprintf("%s@v%s", pkg, packageVersion), apis)
}

func envOr(envKey, defaultValue string) string {
	if value, ok := os.LookupEnv(envKey); ok {
		return value
	}

	return defaultValue
}
