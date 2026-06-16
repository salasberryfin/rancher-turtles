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

package e2e

import (
	"sync"

	turtlesframework "github.com/rancher/turtles/test/framework"
)

// E2EEnvironment contains all environment variables used in E2E tests.
// This provides a centralized, documented source of all configuration options.
type E2EEnvironment struct {
	// Infrastructure Configuration
	ManagementClusterEnvironment ManagementClusterEnvironmentType `env:"MANAGEMENT_CLUSTER_ENVIRONMENT" envDefault:"isolated-kind"`
	UseExistingCluster           bool                             `env:"USE_EXISTING_CLUSTER" envDefault:"false"`
	SkipResourceCleanup          bool                             `env:"SKIP_RESOURCE_CLEANUP" envDefault:"false"`
	SkipDeletionTest             bool                             `env:"SKIP_DELETION_TEST" envDefault:"false"`

	// Kubernetes Configuration
	KubernetesVersion           string `env:"KUBERNETES_VERSION" envDefault:"v1.34.0"`
	KubernetesManagementVersion string `env:"KUBERNETES_MANAGEMENT_VERSION" envDefault:"v1.34.0"`
	RKE2Version                 string `env:"RKE2_VERSION" envDefault:"v1.34.1+rke2r1"`

	// AWS Configuration
	AWSKubernetesVersion          string `env:"AWS_KUBERNETES_VERSION" envDefault:"v1.34.1"`
	AWSEKSVersion                 string `env:"AWS_EKS_VERSION" envDefault:"v1.32.0"`
	AWSRegion                     string `env:"AWS_REGION" envDefault:"eu-west-2"`
	AWSControlPlaneMachineType    string `env:"AWS_CONTROL_PLANE_MACHINE_TYPE" envDefault:"t3.large"`
	AWSNodeMachineType            string `env:"AWS_NODE_MACHINE_TYPE" envDefault:"t3.large"`
	AWSRKE2ControlPlaneMachineType string `env:"AWS_RKE2_CONTROL_PLANE_MACHINE_TYPE" envDefault:"t3.xlarge"`
	AWSRKE2NodeMachineType        string `env:"AWS_RKE2_NODE_MACHINE_TYPE" envDefault:"t3.xlarge"`
	AWSAccessKeyID                string `env:"AWS_ACCESS_KEY_ID"`
	AWSSecretAccessKey            string `env:"AWS_SECRET_ACCESS_KEY"`
	AWSSSHKeyName                 string `env:"AWS_SSH_KEY_NAME"`

	// Azure Configuration
	AzureKubernetesVersion string `env:"AZURE_KUBERNETES_VERSION" envDefault:"v1.34.1"`
	AzureAKSVersion        string `env:"AZURE_AKS_VERSION" envDefault:"v1.33.3"`
	AzureSubscriptionID    string `env:"AZURE_SUBSCRIPTION_ID"`
	AzureClientID          string `env:"AZURE_CLIENT_ID"`
	AzureClientSecret      string `env:"AZURE_CLIENT_SECRET"`
	AzureTenantID          string `env:"AZURE_TENANT_ID"`

	// GCP Configuration
	GCPKubernetesVersion string `env:"GCP_KUBERNETES_VERSION" envDefault:"v1.34.1"`
	GCPMachineType       string `env:"GCP_MACHINE_TYPE" envDefault:"n1-standard-2"`
	GCPRegion            string `env:"GCP_REGION" envDefault:"europe-west2"`
	GCPProject           string `env:"GCP_PROJECT"`
	GCPNetworkName       string `env:"GCP_NETWORK_NAME"`
	GCPImageID           string `env:"GCP_IMAGE_ID"`
	CAPGEncodedCreds     string `env:"CAPG_ENCODED_CREDS"`

	// vSphere Configuration
	VSphereTLSThumbprint      string `env:"VSPHERE_TLS_THUMBPRINT"`
	VSphereServer             string `env:"VSPHERE_SERVER"`
	VSphereDatacenter         string `env:"VSPHERE_DATACENTER"`
	VSphereDatastore          string `env:"VSPHERE_DATASTORE"`
	VSphereFolder             string `env:"VSPHERE_FOLDER"`
	VSphereTemplate           string `env:"VSPHERE_TEMPLATE"`
	VSphereNetwork            string `env:"VSPHERE_NETWORK"`
	VSphereResourcePool       string `env:"VSPHERE_RESOURCE_POOL"`
	VSphereUsername           string `env:"VSPHERE_USERNAME"`
	VSpherePassword           string `env:"VSPHERE_PASSWORD"`
	VSphereKubeVIPIPKubeadm   string `env:"VSPHERE_KUBE_VIP_IP_KUBEADM"`
	VSphereKubeVIPIPRKE2      string `env:"VSPHERE_KUBE_VIP_IP_RKE2"`

	// Rancher Configuration
	RancherVersion     string `env:"RANCHER_VERSION" envDefault:"v2.13.0-rc1"`
	RancherRepoName    string `env:"RANCHER_REPO_NAME" envDefault:"rancher-latest"`
	RancherPath        string `env:"RANCHER_PATH" envDefault:"rancher-latest/rancher"`
	RancherURL         string `env:"RANCHER_URL" envDefault:"https://releases.rancher.com/server-charts/latest"`
	RancherHostname    string `env:"RANCHER_HOSTNAME" envDefault:"localhost"`
	RancherPassword    string `env:"RANCHER_PASSWORD" envDefault:"rancheradmin"`
	RancherDebug       bool   `env:"RANCHER_DEBUG" envDefault:"true"`

	// Turtles Configuration
	TurtlesVersion         string `env:"TURTLES_VERSION" envDefault:"v0.0.1"`
	TurtlesImage           string `env:"TURTLES_IMAGE" envDefault:"ghcr.io/rancher/turtles-e2e"`
	TurtlesImageRegistry   string `env:"TURTLES_IMAGE_REGISTRY" envDefault:"ghcr.io"`
	TurtlesImageRepository string `env:"TURTLES_IMAGE_REPOSITORY" envDefault:"rancher/turtles-e2e"`
	TurtlesPath            string `env:"TURTLES_PATH"`
	TurtlesRepoName        string `env:"TURTLES_REPO_NAME" envDefault:"turtles"`
	TurtlesURL             string `env:"TURTLES_URL" envDefault:"https://rancher.github.io/turtles"`
	TurtlesProviders       string `env:"TURTLES_PROVIDERS" envDefault:"ALL"`
	TurtlesProvidersPath   string `env:"TURTLES_PROVIDERS_PATH"`

	// Gitea Configuration
	GiteaRepoName     string `env:"GITEA_REPO_NAME" envDefault:"gitea-charts"`
	GiteaRepoURL      string `env:"GITEA_REPO_URL" envDefault:"https://dl.gitea.com/charts/"`
	GiteaChartName    string `env:"GITEA_CHART_NAME" envDefault:"gitea"`
	GiteaChartVersion string `env:"GITEA_CHART_VERSION" envDefault:"12.4.0"`
	GiteaUserName     string `env:"GITEA_USER_NAME" envDefault:"gitea_admin"`
	GiteaUserPassword string `env:"GITEA_USER_PWD" envDefault:"password"`

	// Ngrok Configuration (for kind environments)
	NgrokAPIKey    string `env:"NGROK_API_KEY"`
	NgrokAuthToken string `env:"NGROK_AUTHTOKEN"`
	NgrokRepoName  string `env:"NGROK_REPO_NAME" envDefault:"ngrok"`
	NgrokURL       string `env:"NGROK_URL" envDefault:"https://charts.ngrok.com"`
	NgrokPath      string `env:"NGROK_PATH" envDefault:"ngrok/ngrok-operator"`

	// CLI Tool Paths
	HelmBinaryPath       string `env:"HELM_BINARY_PATH" envDefault:"helm"`
	ClusterctlBinaryPath string `env:"CLUSTERCTL_BINARY_PATH"`

	// Artifacts Configuration
	ArtifactsFolder       string `env:"ARTIFACTS_FOLDER" envDefault:"_artifacts"`
	HelmExtraValuesFolder string `env:"HELM_EXTRA_VALUES_FOLDER" envDefault:"/tmp"`

	// Docker Registry Configuration
	DockerRegistryToken    string `env:"DOCKER_REGISTRY_TOKEN"`
	DockerRegistryUsername string `env:"DOCKER_REGISTRY_USERNAME"`
	DockerRegistryConfig   string `env:"DOCKER_REGISTRY_CONFIG"`

	// GitHub Configuration
	GitHubUsername string `env:"GITHUB_USERNAME"`
	GitHubToken    string `env:"GITHUB_TOKEN"`
	SourceRepo     string `env:"SOURCE_REPO"`
	GitHubHeadRef  string `env:"GITHUB_HEAD_REF"`

	// Cert Manager Configuration
	CertManagerRepoName string `env:"CERT_MANAGER_REPO_NAME" envDefault:"jetstack"`
	CertManagerURL      string `env:"CERT_MANAGER_URL" envDefault:"https://charts.jetstack.io"`
	CertManagerPath     string `env:"CERT_MANAGER_PATH" envDefault:"jetstack/cert-manager"`
}

var (
	globalEnv     *E2EEnvironment
	globalEnvOnce sync.Once
	globalEnvErr  error
)

// GetE2EEnvironment returns the global E2E environment configuration.
// It parses environment variables on first call and caches the result.
func GetE2EEnvironment() (*E2EEnvironment, error) {
	globalEnvOnce.Do(func() {
		globalEnv = &E2EEnvironment{}
		globalEnvErr = turtlesframework.Parse(globalEnv)
	})
	return globalEnv, globalEnvErr
}

// MustGetE2EEnvironment returns the global E2E environment configuration.
// It panics if parsing fails.
func MustGetE2EEnvironment() *E2EEnvironment {
	env, err := GetE2EEnvironment()
	if err != nil {
		panic("Failed to parse E2E environment: " + err.Error())
	}
	return env
}

// IsAWSConfigured returns true if AWS credentials are configured.
func (e *E2EEnvironment) IsAWSConfigured() bool {
	return e.AWSAccessKeyID != "" && e.AWSSecretAccessKey != ""
}

// IsAzureConfigured returns true if Azure credentials are configured.
func (e *E2EEnvironment) IsAzureConfigured() bool {
	return e.AzureSubscriptionID != "" && e.AzureClientID != "" &&
		e.AzureClientSecret != "" && e.AzureTenantID != ""
}

// IsGCPConfigured returns true if GCP credentials are configured.
func (e *E2EEnvironment) IsGCPConfigured() bool {
	return e.GCPProject != "" && e.CAPGEncodedCreds != ""
}

// IsVSphereConfigured returns true if vSphere credentials are configured.
func (e *E2EEnvironment) IsVSphereConfigured() bool {
	return e.VSphereServer != "" && e.VSphereUsername != "" && e.VSpherePassword != ""
}

// IsNgrokConfigured returns true if Ngrok credentials are configured.
func (e *E2EEnvironment) IsNgrokConfigured() bool {
	return e.NgrokAPIKey != "" && e.NgrokAuthToken != ""
}

// GetProviderWaitIntervals returns the appropriate wait interval names for a given provider.
func (e *E2EEnvironment) GetProviderWaitIntervals(provider string) (createWait, deleteWait string) {
	switch provider {
	case "aws", "eks":
		return "wait-capa-create-cluster", "wait-eks-delete"
	case "azure", "aks":
		return "wait-capz-create-cluster", "wait-aks-delete"
	case "gcp", "gke":
		return "wait-capg-create-cluster", "wait-gke-delete"
	case "vsphere":
		return "wait-capv-create-cluster", "wait-vsphere-delete"
	default:
		return "wait-rancher", "wait-controllers"
	}
}
