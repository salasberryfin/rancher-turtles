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

package specs

import (
	"k8s.io/utils/ptr"
	"sigs.k8s.io/cluster-api/test/framework"
	"sigs.k8s.io/cluster-api/test/framework/clusterctl"

	"github.com/rancher/turtles/test/e2e"
	turtlesframework "github.com/rancher/turtles/test/framework"
)

// GitOpsTestBuilder provides a fluent interface for constructing GitOps test specifications.
// It reduces boilerplate by providing sensible defaults and chainable configuration methods.
type GitOpsTestBuilder struct {
	input CreateUsingGitOpsSpecInput
}

// NewGitOpsTest creates a new GitOpsTestBuilder with sensible defaults.
// It sets up common labels and default machine counts.
func NewGitOpsTest(clusterName string) *GitOpsTestBuilder {
	return &GitOpsTestBuilder{
		input: CreateUsingGitOpsSpecInput{
			ClusterName:                    clusterName,
			ControlPlaneMachineCount:       ptr.To(1),
			WorkerMachineCount:             ptr.To(1),
			LabelNamespace:                 true,
			CAPIClusterCreateWaitName:      "wait-rancher",
			DeleteClusterWaitName:          "wait-controllers",
			CapiClusterOwnerLabel:          e2e.CapiClusterOwnerLabel,
			CapiClusterOwnerNamespaceLabel: e2e.CapiClusterOwnerNamespaceLabel,
			OwnedLabelName:                 e2e.OwnedLabelName,
		},
	}
}

// WithE2EConfig sets the E2E configuration.
func (b *GitOpsTestBuilder) WithE2EConfig(config *clusterctl.E2EConfig) *GitOpsTestBuilder {
	b.input.E2EConfig = config
	return b
}

// WithBootstrapClusterProxy sets the bootstrap cluster proxy.
func (b *GitOpsTestBuilder) WithBootstrapClusterProxy(proxy framework.ClusterProxy) *GitOpsTestBuilder {
	b.input.BootstrapClusterProxy = proxy
	return b
}

// WithRancherServerURL sets the Rancher server URL.
func (b *GitOpsTestBuilder) WithRancherServerURL(url string) *GitOpsTestBuilder {
	b.input.RancherServerURL = url
	return b
}

// WithTopologyNamespace sets the topology namespace.
func (b *GitOpsTestBuilder) WithTopologyNamespace(namespace string) *GitOpsTestBuilder {
	b.input.TopologyNamespace = namespace
	return b
}

// WithClusterTemplate sets the cluster template.
func (b *GitOpsTestBuilder) WithClusterTemplate(template []byte) *GitOpsTestBuilder {
	b.input.ClusterTemplate = template
	return b
}

// WithControlPlaneMachineCount sets the control plane machine count.
func (b *GitOpsTestBuilder) WithControlPlaneMachineCount(count int) *GitOpsTestBuilder {
	b.input.ControlPlaneMachineCount = ptr.To(count)
	return b
}

// WithWorkerMachineCount sets the worker machine count.
func (b *GitOpsTestBuilder) WithWorkerMachineCount(count int) *GitOpsTestBuilder {
	b.input.WorkerMachineCount = ptr.To(count)
	return b
}

// WithClusterReimport enables or disables cluster reimport testing.
func (b *GitOpsTestBuilder) WithClusterReimport(enabled bool) *GitOpsTestBuilder {
	b.input.TestClusterReimport = enabled
	return b
}

// WithAdditionalTemplateVariables adds template variables.
func (b *GitOpsTestBuilder) WithAdditionalTemplateVariables(vars map[string]string) *GitOpsTestBuilder {
	if b.input.AdditionalTemplateVariables == nil {
		b.input.AdditionalTemplateVariables = make(map[string]string)
	}
	for k, v := range vars {
		b.input.AdditionalTemplateVariables[k] = v
	}
	return b
}

// WithAdditionalFleetGitRepos adds additional Fleet GitRepos.
func (b *GitOpsTestBuilder) WithAdditionalFleetGitRepos(repos ...turtlesframework.FleetCreateGitRepoInput) *GitOpsTestBuilder {
	b.input.AdditionalFleetGitRepos = append(b.input.AdditionalFleetGitRepos, repos...)
	return b
}

// WithWaitIntervals sets the wait interval names.
func (b *GitOpsTestBuilder) WithWaitIntervals(createWait, deleteWait string) *GitOpsTestBuilder {
	b.input.CAPIClusterCreateWaitName = createWait
	b.input.DeleteClusterWaitName = deleteWait
	return b
}

// WithSkipDeletionTest sets whether to skip the deletion test.
func (b *GitOpsTestBuilder) WithSkipDeletionTest(skip bool) *GitOpsTestBuilder {
	b.input.SkipDeletionTest = skip
	return b
}

// --- Provider-specific convenience methods ---

// WithDocker configures the test for Docker provider with Kubeadm.
func (b *GitOpsTestBuilder) WithDocker() *GitOpsTestBuilder {
	b.input.ClusterTemplate = e2e.CAPIDockerKubeadmTopology
	b.input.CAPIClusterCreateWaitName = "wait-rancher"
	b.input.DeleteClusterWaitName = "wait-controllers"
	return b
}

// WithDockerRKE2 configures the test for Docker provider with RKE2.
func (b *GitOpsTestBuilder) WithDockerRKE2() *GitOpsTestBuilder {
	b.input.ClusterTemplate = e2e.CAPIDockerRKE2Topology
	b.input.CAPIClusterCreateWaitName = "wait-rancher"
	b.input.DeleteClusterWaitName = "wait-controllers"
	return b
}

// WithAWSEKS configures the test for AWS EKS.
func (b *GitOpsTestBuilder) WithAWSEKS() *GitOpsTestBuilder {
	b.input.ClusterTemplate = e2e.CAPIAwsEKSTopology
	b.input.CAPIClusterCreateWaitName = "wait-capa-create-cluster"
	b.input.DeleteClusterWaitName = "wait-eks-delete"
	return b
}

// WithAWSKubeadm configures the test for AWS with Kubeadm.
func (b *GitOpsTestBuilder) WithAWSKubeadm() *GitOpsTestBuilder {
	b.input.ClusterTemplate = e2e.CAPIAwsKubeadmTopology
	b.input.CAPIClusterCreateWaitName = "wait-capa-create-cluster"
	b.input.DeleteClusterWaitName = "wait-eks-delete"
	return b
}

// WithAWSRKE2 configures the test for AWS with RKE2.
func (b *GitOpsTestBuilder) WithAWSRKE2() *GitOpsTestBuilder {
	b.input.ClusterTemplate = e2e.CAPIAwsEC2RKE2Topology
	b.input.CAPIClusterCreateWaitName = "wait-capa-create-cluster"
	b.input.DeleteClusterWaitName = "wait-eks-delete"
	return b
}

// WithAzureAKS configures the test for Azure AKS.
func (b *GitOpsTestBuilder) WithAzureAKS() *GitOpsTestBuilder {
	b.input.ClusterTemplate = e2e.CAPIAzureAKSTopology
	b.input.CAPIClusterCreateWaitName = "wait-capz-create-cluster"
	b.input.DeleteClusterWaitName = "wait-aks-delete"
	return b
}

// WithAzureKubeadm configures the test for Azure with Kubeadm.
func (b *GitOpsTestBuilder) WithAzureKubeadm() *GitOpsTestBuilder {
	b.input.ClusterTemplate = e2e.CAPIAzureKubeadmTopology
	b.input.CAPIClusterCreateWaitName = "wait-capz-create-cluster"
	b.input.DeleteClusterWaitName = "wait-aks-delete"
	return b
}

// WithAzureRKE2 configures the test for Azure with RKE2.
func (b *GitOpsTestBuilder) WithAzureRKE2() *GitOpsTestBuilder {
	b.input.ClusterTemplate = e2e.CAPIAzureRKE2Topology
	b.input.CAPIClusterCreateWaitName = "wait-capz-create-cluster"
	b.input.DeleteClusterWaitName = "wait-aks-delete"
	return b
}

// WithGCPGKE configures the test for GCP GKE.
func (b *GitOpsTestBuilder) WithGCPGKE() *GitOpsTestBuilder {
	b.input.ClusterTemplate = e2e.CAPIGCPGKE
	b.input.CAPIClusterCreateWaitName = "wait-capg-create-cluster"
	b.input.DeleteClusterWaitName = "wait-gke-delete"
	return b
}

// WithGCPKubeadm configures the test for GCP with Kubeadm.
func (b *GitOpsTestBuilder) WithGCPKubeadm() *GitOpsTestBuilder {
	b.input.ClusterTemplate = e2e.CAPIGCPKubeadmTopology
	b.input.CAPIClusterCreateWaitName = "wait-capg-create-cluster"
	b.input.DeleteClusterWaitName = "wait-gke-delete"
	return b
}

// WithVSphereKubeadm configures the test for vSphere with Kubeadm.
func (b *GitOpsTestBuilder) WithVSphereKubeadm() *GitOpsTestBuilder {
	b.input.ClusterTemplate = e2e.CAPIvSphereKubeadmTopology
	b.input.CAPIClusterCreateWaitName = "wait-capv-create-cluster"
	b.input.DeleteClusterWaitName = "wait-vsphere-delete"
	return b
}

// WithVSphereRKE2 configures the test for vSphere with RKE2.
func (b *GitOpsTestBuilder) WithVSphereRKE2() *GitOpsTestBuilder {
	b.input.ClusterTemplate = e2e.CAPIvSphereRKE2Topology
	b.input.CAPIClusterCreateWaitName = "wait-capv-create-cluster"
	b.input.DeleteClusterWaitName = "wait-vsphere-delete"
	return b
}

// --- Fleet GitRepo helpers ---

// AddClusterClassGitRepo adds a Fleet GitRepo for cluster classes.
func (b *GitOpsTestBuilder) AddClusterClassGitRepo(name, path string, proxy framework.ClusterProxy) *GitOpsTestBuilder {
	b.input.AdditionalFleetGitRepos = append(b.input.AdditionalFleetGitRepos, turtlesframework.FleetCreateGitRepoInput{
		Name:            name,
		Paths:           []string{path},
		ClusterProxy:    proxy,
		TargetNamespace: b.input.TopologyNamespace,
	})
	return b
}

// AddCNIGitRepo adds a Fleet GitRepo for CNI.
func (b *GitOpsTestBuilder) AddCNIGitRepo(name, path string, proxy framework.ClusterProxy) *GitOpsTestBuilder {
	b.input.AdditionalFleetGitRepos = append(b.input.AdditionalFleetGitRepos, turtlesframework.FleetCreateGitRepoInput{
		Name:            name,
		Paths:           []string{path},
		ClusterProxy:    proxy,
		TargetNamespace: b.input.TopologyNamespace,
	})
	return b
}

// AddCCMGitRepo adds a Fleet GitRepo for Cloud Controller Manager.
func (b *GitOpsTestBuilder) AddCCMGitRepo(name, path string, proxy framework.ClusterProxy) *GitOpsTestBuilder {
	b.input.AdditionalFleetGitRepos = append(b.input.AdditionalFleetGitRepos, turtlesframework.FleetCreateGitRepoInput{
		Name:            name,
		Paths:           []string{path},
		ClusterProxy:    proxy,
		TargetNamespace: b.input.TopologyNamespace,
	})
	return b
}

// AddCSIGitRepo adds a Fleet GitRepo for CSI.
func (b *GitOpsTestBuilder) AddCSIGitRepo(name, path string, proxy framework.ClusterProxy) *GitOpsTestBuilder {
	b.input.AdditionalFleetGitRepos = append(b.input.AdditionalFleetGitRepos, turtlesframework.FleetCreateGitRepoInput{
		Name:            name,
		Paths:           []string{path},
		ClusterProxy:    proxy,
		TargetNamespace: b.input.TopologyNamespace,
	})
	return b
}

// Build returns the configured CreateUsingGitOpsSpecInput.
func (b *GitOpsTestBuilder) Build() CreateUsingGitOpsSpecInput {
	return b.input
}

// BuildFunc returns a function that returns the configured CreateUsingGitOpsSpecInput.
// This is useful for passing to CreateUsingGitOpsSpec which expects a getter function.
func (b *GitOpsTestBuilder) BuildFunc() func() CreateUsingGitOpsSpecInput {
	input := b.input
	return func() CreateUsingGitOpsSpecInput {
		return input
	}
}
