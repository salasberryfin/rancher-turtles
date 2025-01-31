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

package import_gitops

import (
	_ "embed"

	. "github.com/onsi/ginkgo/v2"
	. "sigs.k8s.io/controller-runtime/pkg/envtest/komega"

	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/envtest/komega"

	"github.com/rancher/turtles/test/e2e"
	"github.com/rancher/turtles/test/e2e/specs"
)

var _ = Describe("[Docker] [Kubeadm] Create and delete CAPI cluster functionality should work with namespace auto-import", func() {
	BeforeEach(func() {
		SetClient(bootstrapClusterProxy.GetClient())
		SetContext(ctx)
	})

	specs.CreateUsingGitOpsSpec(ctx, func() specs.CreateUsingGitOpsSpecInput {
		return specs.CreateUsingGitOpsSpecInput{
			E2EConfig:                 e2e.LoadE2EConfig(),
			BootstrapClusterProxy:     bootstrapClusterProxy,
			ClusterTemplate:           e2e.CAPIDockerKubeadm,
			AdditionalTemplates:       [][]byte{e2e.CAPIKindnet},
			ClusterName:               "clusterv1-docker-kubeadm",
			ControlPlaneMachineCount:  ptr.To[int](1),
			WorkerMachineCount:        ptr.To[int](1),
			GitAddr:                   gitAddress,
			SkipDeletionTest:          false,
			LabelNamespace:            true,
			TestClusterReimport:       true,
			RancherServerURL:          hostName,
			CAPIClusterCreateWaitName: "wait-rancher",
			DeleteClusterWaitName:     "wait-controllers",
		}
	})
})

var _ = Describe("[AWS] [EKS] Create and delete CAPI cluster functionality should work with namespace auto-import", Label(e2e.FullTestLabel), func() {
	BeforeEach(func() {
		komega.SetClient(bootstrapClusterProxy.GetClient())
		komega.SetContext(ctx)
	})

	specs.CreateUsingGitOpsSpec(ctx, func() specs.CreateUsingGitOpsSpecInput {
		return specs.CreateUsingGitOpsSpecInput{
			E2EConfig:                 e2e.LoadE2EConfig(),
			BootstrapClusterProxy:     bootstrapClusterProxy,
			ClusterTemplate:           e2e.CAPIAwsEKSMMP,
			ClusterName:               "clusterv1-eks",
			ControlPlaneMachineCount:  ptr.To[int](1),
			WorkerMachineCount:        ptr.To[int](1),
			GitAddr:                   gitAddress,
			SkipDeletionTest:          false,
			LabelNamespace:            true,
			RancherServerURL:          hostName,
			CAPIClusterCreateWaitName: "wait-capa-create-cluster",
			DeleteClusterWaitName:     "wait-eks-delete",
		}
	})
})

var _ = Describe("[vSphere] [Kubeadm] Create and delete CAPI cluster functionality should work with namespace auto-import", Label(e2e.LocalTestLabel), func() {
	BeforeEach(func() {
		SetClient(bootstrapClusterProxy.GetClient())
		SetContext(ctx)
	})

	specs.CreateUsingGitOpsSpec(ctx, func() specs.CreateUsingGitOpsSpecInput {
		return specs.CreateUsingGitOpsSpecInput{
			E2EConfig:                 e2e.LoadE2EConfig(),
			BootstrapClusterProxy:     bootstrapClusterProxy,
			ClusterTemplate:           e2e.CAPIvSphereKubeadm,
			ClusterName:               "cluster-vsphere-kubeadm",
			ControlPlaneMachineCount:  ptr.To[int](1),
			WorkerMachineCount:        ptr.To[int](1),
			GitAddr:                   gitAddress,
			SkipDeletionTest:          false,
			LabelNamespace:            true,
			RancherServerURL:          hostName,
			CAPIClusterCreateWaitName: "wait-capv-create-cluster",
			DeleteClusterWaitName:     "wait-vsphere-delete",
			AdditionalTemplateVariables: map[string]string{
				"NAMESPACE":             "default",
				"VIP_NETWORK_INTERFACE": "",
			},
		}
	})
})

var _ = Describe("[vSphere] [RKE2] Create and delete CAPI cluster functionality should work with namespace auto-import", Label(e2e.LocalTestLabel), func() {
	BeforeEach(func() {
		SetClient(bootstrapClusterProxy.GetClient())
		SetContext(ctx)
	})

	specs.CreateUsingGitOpsSpec(ctx, func() specs.CreateUsingGitOpsSpecInput {
		return specs.CreateUsingGitOpsSpecInput{
			E2EConfig:                 e2e.LoadE2EConfig(),
			BootstrapClusterProxy:     bootstrapClusterProxy,
			ClusterTemplate:           e2e.CAPIvSphereRKE2,
			ClusterName:               "cluster-vsphere-rke2",
			ControlPlaneMachineCount:  ptr.To[int](1),
			WorkerMachineCount:        ptr.To[int](1),
			GitAddr:                   gitAddress,
			SkipDeletionTest:          false,
			LabelNamespace:            true,
			RancherServerURL:          hostName,
			CAPIClusterCreateWaitName: "wait-capv-create-cluster",
			DeleteClusterWaitName:     "wait-vsphere-delete",
			AdditionalTemplateVariables: map[string]string{
				"NAMESPACE":             "default",
				"VIP_NETWORK_INTERFACE": "",
			},
		}
	})
})
