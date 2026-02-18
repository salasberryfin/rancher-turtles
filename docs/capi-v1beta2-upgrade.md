# Upgrading to CAPI v1beta2

This release of Rancher Turtles adopts the CAPI `v1beta2` contract, which comes with CAPI core v1.12.2. If you are using CAPI-based provisioning, there are a few things to review before upgrading.

## What changed

### ClusterClass structure

The most visible change is in how `ClusterClass` resources reference templates. In `v1beta1`, template pointers used a field called `ref`. In `v1beta2`, the same field is called `templateRef`. This affects `spec.infrastructure`, `spec.controlPlane`, and `spec.controlPlane.machineInfrastructure`.

Before:
```yaml
spec:
  infrastructure:
    ref:
      apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
      kind: DockerClusterTemplate
      name: my-cluster-template
```

After:
```yaml
spec:
  infrastructure:
    templateRef:
      apiVersion: infrastructure.cluster.x-k8s.io/v1beta2
      kind: DockerClusterTemplate
      name: my-cluster-template
```

The second change in `ClusterClass` is in the workers section. In `v1beta1`, bootstrap and infrastructure references for machine deployments were nested under a `template` wrapper. That wrapper is gone in `v1beta2`, and `bootstrap` and `infrastructure` are now direct children of the machine deployment entry.

Before:
```yaml
workers:
  machineDeployments:
    - class: default-worker
      template:
        bootstrap:
          ref:
            apiVersion: bootstrap.cluster.x-k8s.io/v1beta1
            kind: KubeadmConfigTemplate
            name: my-bootstrap-template
        infrastructure:
          ref:
            apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
            kind: DockerMachineTemplate
            name: my-worker-template
```

After:
```yaml
workers:
  machineDeployments:
    - class: default-worker
      bootstrap:
        templateRef:
          apiVersion: bootstrap.cluster.x-k8s.io/v1beta2
          kind: KubeadmConfigTemplate
          name: my-bootstrap-template
      infrastructure:
        templateRef:
          apiVersion: infrastructure.cluster.x-k8s.io/v1beta2
          kind: DockerMachineTemplate
          name: my-worker-template
```

### API version in templates

Beyond the structural changes, you also need to update the API version string wherever CAPI core types are referenced. Resources like `KubeadmControlPlaneTemplate`, `KubeadmConfigTemplate`, and `DockerMachineTemplate` now carry `v1beta2` in their `apiVersion`. The same applies to the `ClusterClass` object itself.

For infrastructure providers that are still on `v1beta1` (such as CAPZ, CAPG, and CAPV at the time of this release), you keep their existing API version. Only core CAPI types and providers that have explicitly adopted `v1beta2` need the version bump. See the updated examples under `examples/clusterclasses/` for the per-provider details.

### Provider versions

Along with the API contract change, all bundled providers have been updated. The minimum version for any provider in this release is v1.11, and most are higher. The full list is in `internal/controllers/clusterctl/config-prime.yaml`. If you manage provider versions yourself through a `CAPIProvider` resource or a `ClusterctlConfig`, make sure the versions you pin are compatible with `v1beta2`.

## Backward compatibility

CAPI v1.12 still serves `v1beta1` alongside `v1beta2`, so your existing resources on the cluster will continue to work without any immediate migration. The API server will accept and convert between the two versions transparently. This means you do not have to migrate everything at once, but any new `ClusterClass` resources you create should use `v1beta2` to stay current.

The `Cluster` resource itself is still commonly referenced as `v1beta1` in practice and that continues to be valid.

## Steps to take before upgrading

1. Review any `ClusterClass` manifests you maintain and update them to use `templateRef` and remove the `template` wrapper in the workers section.
2. Update the `apiVersion` on core CAPI objects to `v1beta2`. Leave provider-specific types at whatever version that provider currently ships.
3. If you pin provider versions manually, check that the versions you use support `v1beta2` (v1.11 or later for CAPI core providers).
4. Review the updated examples in `examples/clusterclasses/` for reference on what the correct structure looks like for each cloud.
