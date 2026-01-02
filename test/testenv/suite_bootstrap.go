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

package testenv

import (
	"context"

	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/cluster-api/test/framework"
	"sigs.k8s.io/cluster-api/test/framework/clusterctl"

	"github.com/rancher/turtles/test/e2e"
	turtlesframework "github.com/rancher/turtles/test/framework"
)

const (
	// ChartUpgradeKubernetesVersion is the Kubernetes version required for chart-upgrade tests
	// to maintain compatibility with Rancher 2.12.x and avoid issues with CAAPF on v1.33.x.
	// See: test/e2e/suites/chart-upgrade/suite_test.go
	ChartUpgradeKubernetesVersion = "v1.32.0"
)

// SuiteConfig defines the configuration for setting up a test suite.
// It provides a unified way to configure what components are needed for each suite.
type SuiteConfig struct {
	// Name is the unique identifier for this suite (e.g., "import-gitops", "capiprovider")
	Name string

	// NeedsCertManager indicates whether cert-manager should be deployed
	NeedsCertManager bool

	// NeedsIngress indicates whether ingress should be deployed
	NeedsIngress bool

	// NeedsGitea indicates whether Gitea should be deployed for chart repository
	NeedsGitea bool

	// NeedsRancher indicates whether Rancher should be deployed
	NeedsRancher bool

	// NeedsTurtles indicates whether Turtles controller should be deployed via system charts
	NeedsTurtles bool

	// NeedsTurtlesProviders indicates whether Turtles providers should be deployed
	NeedsTurtlesProviders bool

	// UseLegacyCAPINamespace specifies whether to use the legacy CAPI namespace
	UseLegacyCAPINamespace bool

	// RancherPatches are additional patches to apply to Rancher configuration
	RancherPatches [][]byte

	// KubernetesVersion overrides the default Kubernetes version for the management cluster
	KubernetesVersion string

	// CustomSetup is an optional function for suite-specific setup after standard components
	CustomSetup func(ctx context.Context, result *SuiteSetupResult) error

	// CustomCleanup is an optional function for suite-specific cleanup before standard teardown
	CustomCleanup func(ctx context.Context, result *SuiteSetupResult) error
}

// SuiteSetupResult contains the results of setting up a test suite.
type SuiteSetupResult struct {
	// ClusterResult contains the bootstrap cluster setup result
	ClusterResult *SetupTestClusterResult

	// GiteaResult contains the Gitea deployment result (if deployed)
	GiteaResult *DeployGiteaResult

	// ChartsResult contains the charts push result (if charts were pushed)
	ChartsResult *PushRancherChartsToGiteaResult

	// RancherHostname is the hostname of the Rancher installation
	RancherHostname string

	// E2EConfig is the E2E configuration used for this suite
	E2EConfig *clusterctl.E2EConfig

	// Logger is the structured logger for this suite
	Logger *turtlesframework.TestLogger
}

// DefaultSuiteConfig returns a SuiteConfig with common defaults for most test suites.
func DefaultSuiteConfig(name string) *SuiteConfig {
	return &SuiteConfig{
		Name:                   name,
		NeedsCertManager:       true,
		NeedsIngress:           true,
		NeedsGitea:             true,
		NeedsRancher:           true,
		NeedsTurtles:           true,
		NeedsTurtlesProviders:  true,
		UseLegacyCAPINamespace: false,
		RancherPatches:         [][]byte{e2e.RancherSettingPatch},
	}
}

// ImportGitopsSuiteConfig returns the configuration for the import-gitops test suite.
func ImportGitopsSuiteConfig() *SuiteConfig {
	config := DefaultSuiteConfig("import-gitops")
	return config
}

// CAPIProviderSuiteConfig returns the configuration for the capiprovider test suite.
func CAPIProviderSuiteConfig() *SuiteConfig {
	config := DefaultSuiteConfig("capiprovider")
	config.NeedsTurtlesProviders = false // capiprovider tests manage providers directly
	return config
}

// ChartUpgradeSuiteConfig returns the configuration for the chart-upgrade test suite.
func ChartUpgradeSuiteConfig() *SuiteConfig {
	config := DefaultSuiteConfig("chart-upgrade")
	config.NeedsRancher = false // chart-upgrade tests install Rancher with specific versions
	config.NeedsTurtles = false
	config.NeedsTurtlesProviders = false
	config.KubernetesVersion = ChartUpgradeKubernetesVersion
	return config
}

// V2ProvSuiteConfig returns the configuration for the v2prov test suite.
func V2ProvSuiteConfig() *SuiteConfig {
	config := DefaultSuiteConfig("v2prov")
	return config
}

// SetupSuite sets up the test suite based on the provided configuration.
// It handles all the common setup logic and can be extended via CustomSetup.
func SetupSuite(ctx context.Context, config *SuiteConfig) *SuiteSetupResult {
	Expect(config).ToNot(BeNil(), "SuiteConfig is required for SetupSuite")
	Expect(config.Name).ToNot(BeEmpty(), "Suite name is required")

	result := &SuiteSetupResult{
		Logger: turtlesframework.NewTestLogger(config.Name, "setup"),
	}

	// Load E2E config and set management cluster name
	e2eConfig := e2e.LoadE2EConfig()
	e2eConfig.ManagementClusterName = e2eConfig.ManagementClusterName + "-" + config.Name
	result.E2EConfig = e2eConfig

	// Setup the test cluster
	result.Logger.Step("Setting up test cluster for suite: %s", config.Name)
	setupInput := SetupTestClusterInput{
		E2EConfig: e2eConfig,
		Scheme:    e2e.InitScheme(),
	}
	if config.KubernetesVersion != "" {
		setupInput.KubernetesVersion = config.KubernetesVersion
	}
	result.ClusterResult = SetupTestCluster(ctx, setupInput)

	// Deploy cert-manager if needed
	if config.NeedsCertManager {
		result.Logger.Step("Deploying cert-manager")
		DeployCertManager(ctx, DeployCertManagerInput{
			BootstrapClusterProxy: result.ClusterResult.BootstrapClusterProxy,
		})
	}

	// Deploy ingress if needed
	if config.NeedsIngress {
		result.Logger.Step("Deploying ingress")
		RancherDeployIngress(ctx, RancherDeployIngressInput{
			BootstrapClusterProxy:     result.ClusterResult.BootstrapClusterProxy,
			CustomIngress:             e2e.NginxIngress,
			CustomIngressLoadBalancer: e2e.NginxIngressLoadBalancer,
			DefaultIngressClassPatch:  e2e.IngressClassPatch,
		})
	}

	// Deploy Gitea if needed
	if config.NeedsGitea {
		result.Logger.Step("Deploying Gitea for chart repository")
		result.GiteaResult = DeployGitea(ctx, DeployGiteaInput{
			BootstrapClusterProxy: result.ClusterResult.BootstrapClusterProxy,
			ValuesFile:            e2e.GiteaValues,
			CustomIngressConfig:   e2e.GiteaIngress,
		})

		result.Logger.Step("Pushing Rancher charts to Gitea")
		result.ChartsResult = PushRancherChartsToGitea(ctx, PushRancherChartsToGiteaInput{
			BootstrapClusterProxy: result.ClusterResult.BootstrapClusterProxy,
			GiteaServerAddress:    result.GiteaResult.GitAddress,
			GiteaRepoName:         "charts",
		})
	}

	// Deploy Rancher if needed
	// Note: This setup uses Gitea-based chart deployment (UpgradeInstallRancherWithGitea).
	// For Rancher deployment without Gitea, use CustomSetup or a different suite configuration.
	if config.NeedsRancher {
		if result.GiteaResult == nil || result.ChartsResult == nil {
			// If Rancher is needed but Gitea/Charts are not configured, skip with a warning
			result.Logger.Info("Skipping Rancher deployment: requires Gitea chart repository but Gitea is not deployed or charts failed to push")
		} else {
			result.Logger.Step("Installing Rancher with Gitea chart repository")
			rancherHookResult := UpgradeInstallRancherWithGitea(ctx, UpgradeInstallRancherWithGiteaInput{
				BootstrapClusterProxy: result.ClusterResult.BootstrapClusterProxy,
				ChartRepoURL:          result.ChartsResult.ChartRepoHTTPURL,
				ChartRepoBranch:       result.ChartsResult.Branch,
				ChartVersion:          result.ChartsResult.ChartVersion,
				TurtlesImageRepo:      "ghcr.io/rancher/turtles-e2e",
				TurtlesImageTag:       "v0.0.1",
				RancherWaitInterval:   e2eConfig.GetIntervals(result.ClusterResult.BootstrapClusterProxy.GetName(), "wait-rancher"),
				RancherPatches:        config.RancherPatches,
			})
			result.RancherHostname = rancherHookResult.Hostname

			result.Logger.Step("Waiting for Rancher to be ready")
			framework.WaitForDeploymentsAvailable(ctx, framework.WaitForDeploymentsAvailableInput{
				Getter: result.ClusterResult.BootstrapClusterProxy.GetClient(),
				Deployment: &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{
					Name:      "rancher",
					Namespace: e2e.RancherNamespace,
				}},
			}, e2eConfig.GetIntervals(result.ClusterResult.BootstrapClusterProxy.GetName(), "wait-rancher")...)

			// Wait for Turtles if deployed via system charts
			if config.NeedsTurtles {
				result.Logger.Step("Waiting for Turtles controller to be installed by system chart controller")
				framework.WaitForDeploymentsAvailable(ctx, framework.WaitForDeploymentsAvailableInput{
					Getter: result.ClusterResult.BootstrapClusterProxy.GetClient(),
					Deployment: &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{
						Name:      "rancher-turtles-controller-manager",
						Namespace: e2e.NewRancherTurtlesNamespace,
					}},
				}, e2eConfig.GetIntervals(result.ClusterResult.BootstrapClusterProxy.GetName(), "wait-controllers")...)

				result.Logger.Step("Applying test ClusterctlConfig")
				Expect(turtlesframework.Apply(ctx, result.ClusterResult.BootstrapClusterProxy, e2e.ClusterctlConfig)).To(Succeed())
			}

			// Deploy Turtles providers if needed
			if config.NeedsTurtlesProviders {
				result.Logger.Step("Deploying Rancher Turtles providers")
				DeployRancherTurtlesProviders(ctx, DeployRancherTurtlesProvidersInput{
					BootstrapClusterProxy:   result.ClusterResult.BootstrapClusterProxy,
					UseLegacyCAPINamespace:  config.UseLegacyCAPINamespace,
					RancherTurtlesNamespace: e2e.NewRancherTurtlesNamespace,
				})
			}
		}
	}

	// Run custom setup if provided
	if config.CustomSetup != nil {
		result.Logger.Step("Running custom suite setup")
		Expect(config.CustomSetup(ctx, result)).To(Succeed(), "Custom setup failed")
	}

	result.Logger.Info("Suite setup completed in %s", result.Logger.GetElapsed())

	return result
}

// CleanupSuite performs cleanup for a test suite based on the configuration.
func CleanupSuite(ctx context.Context, config *SuiteConfig, result *SuiteSetupResult, skipCleanup bool) {
	if result == nil || result.ClusterResult == nil {
		return
	}

	logger := turtlesframework.NewTestLogger(config.Name, "cleanup")

	// Dump artifacts
	logger.Step("Dumping artifacts from the bootstrap cluster")
	DumpBootstrapCluster(ctx, result.ClusterResult.BootstrapClusterProxy.GetKubeconfigPath())

	if skipCleanup {
		logger.Info("Skipping resource cleanup as requested")
		return
	}

	// Run custom cleanup if provided
	if config.CustomCleanup != nil {
		logger.Step("Running custom suite cleanup")
		if err := config.CustomCleanup(ctx, result); err != nil {
			logger.Error("Custom cleanup failed: %v", err)
		}
	}

	// Cleanup cluster
	logger.Step("Tearing down the management cluster")
	CleanupTestCluster(ctx, CleanupTestClusterInput{
		SetupTestClusterResult: *result.ClusterResult,
	})

	logger.Info("Suite cleanup completed")
}
