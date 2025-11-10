//go:build e2e
// +build e2e

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

package chart_upgrade

import (
	_ "embed"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "sigs.k8s.io/controller-runtime/pkg/envtest/komega"

	"github.com/rancher/turtles/test/e2e"
	"github.com/rancher/turtles/test/e2e/specs"
	"github.com/rancher/turtles/test/framework"
	"github.com/rancher/turtles/test/testenv"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	capiframework "sigs.k8s.io/cluster-api/test/framework"
	"sigs.k8s.io/cluster-api/util"
)

var _ = Describe("Chart upgrade functionality should work", Ordered, Label(e2e.ShortTestLabel), func() {
	var (
		clusterName       string
		topologyNamespace = "creategitops-docker-rke2"
	)

	BeforeAll(func() {
		clusterName = fmt.Sprintf("docker-rke2-%s", util.RandomString(6))

		SetClient(bootstrapClusterProxy.GetClient())
		SetContext(ctx)
	})

	// Note that this test suite tests migration from v0.24.x to v0.25.x
	// which includes migration from helm-based installation to the new system chart controller architecture.
	// The old version (v0.24.x) already has the embedded cluster-api-operator.
	// The old version is installed using helm install, and the upgrade
	// uses the system chart controller via Gitea chart repository.
	It("Should install old version of Turtles using helm", func() {
		rtInput := testenv.DeployRancherTurtlesInput{
			BootstrapClusterProxy: bootstrapClusterProxy,
			TurtlesChartRepoName:  "rancher-turtles",
			TurtlesChartUrl:       "https://rancher.github.io/turtles",
			Version:               "v0.24.3",
			AdditionalValues: map[string]string{
				"rancherTurtles.namespace": e2e.RancherTurtlesNamespace,
			},
		}
		testenv.DeployRancherTurtles(ctx, rtInput)

		By("Waiting for Turtles controller Deployment to be ready")
		capiframework.WaitForDeploymentsAvailable(ctx, capiframework.WaitForDeploymentsAvailableInput{
			Getter: bootstrapClusterProxy.GetClient(),
			Deployment: &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{
				Name:      "rancher-turtles-controller-manager",
				Namespace: e2e.RancherTurtlesNamespace,
			}},
		}, e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Deploying CAPI providers via providers chart")
		testenv.DeployRancherTurtlesProviders(ctx, testenv.DeployRancherTurtlesProvidersInput{
			BootstrapClusterProxy:        bootstrapClusterProxy,
			WaitDeploymentsReadyInterval: e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers"),
			UseLegacyCAPINamespace:       true, // Using old version (v0.24.3) which uses capi-system
			// Limit providers to what matches isolated-kind topology to avoid Helm ownership
			// conflicts with legacy chart RBAC (e.g., Azure aggregated role), while keeping
			// migration enabled for namespaced resources.
			ProviderList: "kubeadm,docker",
			// CAAPF and RKE2 providers are enabled by default
			// Adding Kubeadm and Docker for comprehensive testing
			AdditionalValues: map[string]string{
				"providers.bootstrapKubeadm.enabled":     "true",
				"providers.controlplaneKubeadm.enabled":  "true",
				"providers.infrastructureDocker.enabled": "true",
				// Explicitly disable cloud providers to prevent duplicate cluster-scoped RBAC
				// from the legacy chart causing Helm ownership conflicts during install.
				"providers.infrastructureAWS.enabled":     "false",
				"providers.infrastructureAzure.enabled":   "false",
				"providers.infrastructureGCP.enabled":     "false",
				"providers.infrastructureVSphere.enabled": "false",
			},
		})
	})

	Context("Provisioning a workload Cluster", func() {
		// Provision a workload Cluster.
		// This ensures that upgrading the chart will not unexpectedly lead to unready Cluster or Machines.
		specs.CreateUsingGitOpsSpec(ctx, func() specs.CreateUsingGitOpsSpecInput {
			return specs.CreateUsingGitOpsSpecInput{
				E2EConfig:                      e2e.LoadE2EConfig(),
				BootstrapClusterProxy:          bootstrapClusterProxy,
				ClusterTemplate:                e2e.CAPIDockerRKE2Topology,
				ClusterName:                    clusterName,
				ControlPlaneMachineCount:       ptr.To(1),
				WorkerMachineCount:             ptr.To(1),
				LabelNamespace:                 true,
				TestClusterReimport:            false,
				RancherServerURL:               hostName,
				CAPIClusterCreateWaitName:      "wait-rancher",
				DeleteClusterWaitName:          "wait-controllers",
				CapiClusterOwnerLabel:          e2e.CapiClusterOwnerLabel,
				CapiClusterOwnerNamespaceLabel: e2e.CapiClusterOwnerNamespaceLabel,
				OwnedLabelName:                 e2e.OwnedLabelName,
				TopologyNamespace:              topologyNamespace,
				SkipCleanup:                    true,
				SkipDeletionTest:               true,
				// AdditionalTemplateVariables: map[string]string{
				// 	e2e.KubernetesVersionVar: e2e.LoadE2EConfig().GetVariableOrEmpty(e2e.KubernetesVersionVar),
				// },
				AdditionalFleetGitRepos: []framework.FleetCreateGitRepoInput{
					{
						Name:            "docker-cluster-classes-regular",
						Paths:           []string{"examples/clusterclasses/docker/rke2"},
						ClusterProxy:    bootstrapClusterProxy,
						TargetNamespace: topologyNamespace,
					},
					{
						Name:            "docker-cni",
						Paths:           []string{"examples/applications/cni/calico"},
						ClusterProxy:    bootstrapClusterProxy,
						TargetNamespace: topologyNamespace,
					},
				},
			}
		})
	})

	It("Should upgrade Turtles via system chart controller and validate providers", func() {
		// There can be only one core CAPI provider installed at a time.
		By("Remove the core CAPI provider before upgrade")
		testenv.RemoveCAPIProvider(ctx, testenv.RemoveCAPIProviderInput{
			BootstrapClusterProxy: bootstrapClusterProxy,
			ProviderName:          "cluster-api",
			ProviderNamespace:     "capi-system",
		})

		By("Configuring Rancher to use Gitea chart repository for system chart controller")
		// Update Rancher deployment with environment variables to enable system chart controller
		// This simulates upgrading Rancher to a version with system chart controller support
		// The chart version was passed from the setup phase where it was populated from RANCHER_CHART_DEV_VERSION
		testenv.UpdateRancherDeploymentWithChartConfig(ctx, testenv.UpdateRancherDeploymentWithChartConfigInput{
			BootstrapClusterProxy: bootstrapClusterProxy,
			ChartRepoURL:          chartsResult.ChartRepoHTTPURL,
			ChartRepoBranch:       chartsResult.Branch,
			ChartVersion:          chartsResult.ChartVersion,
		})

		By("Waiting for Rancher to restart with new configuration")
		// Wait for Rancher deployment to be ready after update
		capiframework.WaitForDeploymentsAvailable(ctx, capiframework.WaitForDeploymentsAvailableInput{
			Getter: bootstrapClusterProxy.GetClient(),
			Deployment: &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{
				Name:      "rancher",
				Namespace: e2e.RancherNamespace,
			}},
		}, e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-rancher")...)

		By("Waiting for Turtles controller deployment to be upgraded")
		capiframework.WaitForDeploymentsAvailable(ctx, capiframework.WaitForDeploymentsAvailableInput{
			Getter: bootstrapClusterProxy.GetClient(),
			Deployment: &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{
				Name:      "rancher-turtles-controller-manager",
				Namespace: e2e.RancherTurtlesNamespace,
			}},
		}, e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Verifying CAPI providers are still running after upgrade")
		// Since providers were installed before upgrade, they should remain operational
		// and keep managing the existing workload cluster
		framework.WaitForCAPIProviderRollout(ctx, framework.WaitForCAPIProviderRolloutInput{
			Getter:    bootstrapClusterProxy.GetClient(),
			Name:      "cluster-api",
			Namespace: "capi-system",
		}, e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		framework.WaitForCAPIProviderRollout(ctx, framework.WaitForCAPIProviderRolloutInput{
			Getter:    bootstrapClusterProxy.GetClient(),
			Name:      "kubeadm-bootstrap",
			Namespace: "capi-kubeadm-bootstrap-system",
		}, e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		framework.WaitForCAPIProviderRollout(ctx, framework.WaitForCAPIProviderRolloutInput{
			Getter:    bootstrapClusterProxy.GetClient(),
			Name:      "kubeadm-control-plane",
			Namespace: "capi-kubeadm-control-plane-system",
		}, e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		framework.WaitForCAPIProviderRollout(ctx, framework.WaitForCAPIProviderRolloutInput{
			Getter:    bootstrapClusterProxy.GetClient(),
			Name:      "rke2-bootstrap",
			Namespace: "rke2-bootstrap-system",
		}, e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		framework.WaitForCAPIProviderRollout(ctx, framework.WaitForCAPIProviderRolloutInput{
			Getter:    bootstrapClusterProxy.GetClient(),
			Name:      "rke2-control-plane",
			Namespace: "rke2-control-plane-system",
		}, e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		framework.WaitForCAPIProviderRollout(ctx, framework.WaitForCAPIProviderRolloutInput{
			Getter:    bootstrapClusterProxy.GetClient(),
			Name:      "fleet",
			Namespace: "rancher-turtles-system",
		}, e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		framework.WaitForCAPIProviderRollout(ctx, framework.WaitForCAPIProviderRolloutInput{
			Getter:    bootstrapClusterProxy.GetClient(),
			Name:      "docker",
			Namespace: "capd-system",
		}, e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Expanding providers under system chart controller (post-upgrade)")
		testenv.DeployRancherTurtlesProviders(ctx, testenv.DeployRancherTurtlesProvidersInput{
			BootstrapClusterProxy:        bootstrapClusterProxy,
			WaitDeploymentsReadyInterval: e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers"),
			UseLegacyCAPINamespace:       false, // After upgrade, using new version with cattle-capi-system
			// Enable additional cloud providers now that the system chart controller is active
			ProviderList: "aws,gcp,vsphere",
			AdditionalValues: map[string]string{
				"providers.infrastructureAWS.enabled":     "true",
				"providers.infrastructureGCP.enabled":     "true",
				"providers.infrastructureVSphere.enabled": "true",
			},
		})

		By("Verifying workload cluster is still healthy after upgrade")
		framework.VerifyCluster(ctx, framework.VerifyClusterInput{
			BootstrapClusterProxy:   bootstrapClusterProxy,
			Name:                    clusterName,
			DeleteAfterVerification: true,
		})
	})
})
