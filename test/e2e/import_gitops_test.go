//go:build e2e
// +build e2e

/*
Copyright 2023 SUSE.

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
	. "github.com/onsi/ginkgo/v2"
	"k8s.io/utils/ptr"
	. "sigs.k8s.io/controller-runtime/pkg/envtest/komega"
)

var _ = Describe("[Docker] [Kubeadm] Create and delete CAPI cluster functionality should work with namespace auto-import", Label(shortTestLabel, fullTestLabel), func() {

	BeforeEach(func() {
		SetClient(bootstrapClusterProxy.GetClient())
		SetContext(ctx)
	})

	CreateUsingGitOpsSpec(ctx, func() CreateUsingGitOpsSpecInput {
		return CreateUsingGitOpsSpecInput{
			E2EConfig:                 e2eConfig,
			BootstrapClusterProxy:     bootstrapClusterProxy,
			ClusterctlConfigPath:      clusterctlConfigPath,
			ClusterctlBinaryPath:      clusterctlBinaryPath,
			ArtifactFolder:            artifactFolder,
			ClusterTemplatePath:       "./data/cluster-templates/docker-kubeadm.yaml",
			ClusterName:               "cluster1",
			ControlPlaneMachineCount:  ptr.To[int](1),
			WorkerMachineCount:        ptr.To[int](1),
			GitAddr:                   gitAddress,
			GitAuthSecretName:         authSecretName,
			SkipCleanup:               false,
			SkipDeletionTest:          false,
			LabelNamespace:            true,
			RancherServerURL:          hostName,
			CAPIClusterCreateWaitName: "wait-rancher",
			DeleteClusterWaitName:     "wait-controllers",
		}
	})
})

var _ = Describe("[AWS] [EKS] Create and delete CAPI cluster functionality should work with namespace auto-import", Label(fullTestLabel), func() {

	BeforeEach(func() {
		SetClient(bootstrapClusterProxy.GetClient())
		SetContext(ctx)
	})

	CreateUsingGitOpsSpec(ctx, func() CreateUsingGitOpsSpecInput {
		return CreateUsingGitOpsSpecInput{
			E2EConfig:                 e2eConfig,
			BootstrapClusterProxy:     bootstrapClusterProxy,
			ClusterctlConfigPath:      clusterctlConfigPath,
			ClusterctlBinaryPath:      clusterctlBinaryPath,
			ArtifactFolder:            artifactFolder,
			ClusterTemplatePath:       "./data/cluster-templates/aws-eks-mmp.yaml",
			ClusterName:               "cluster2",
			ControlPlaneMachineCount:  ptr.To[int](1),
			WorkerMachineCount:        ptr.To[int](1),
			GitAddr:                   gitAddress,
			GitAuthSecretName:         authSecretName,
			SkipCleanup:               false,
			SkipDeletionTest:          false,
			LabelNamespace:            true,
			RancherServerURL:          hostName,
			CAPIClusterCreateWaitName: "wait-capa-create-cluster",
			DeleteClusterWaitName:     "wait-eks-delete",
		}
	})
})