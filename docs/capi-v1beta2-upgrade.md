# Upgrading to CAPI v1beta2

This release of Rancher Turtles adopts the CAPI `v1beta2` contract, which comes with CAPI core v1.12.2. If you are using CAPI-based provisioning, there are a few things to review before upgrading.

The changes fall into three areas: how `ClusterClass` resources are structured, how `Cluster` topology references its class, and what changed in the RKE2 provider. Read through the sections that apply to your setup before making any changes.

## ClusterClass

### Template references

The most visible change is in how a `ClusterClass` points to its templates. In `v1beta1`, the pointer field was called `ref`. In `v1beta2` it is called `templateRef`. This applies to `spec.infrastructure`, `spec.controlPlane`, and `spec.controlPlane.machineInfrastructure`.

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

### Workers section

In `v1beta1`, bootstrap and infrastructure references for machine deployments were nested under a `template` wrapper. That wrapper is gone in `v1beta2`. `bootstrap` and `infrastructure` are now direct children of the machine deployment entry.

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

### API versions

You also need to update the `apiVersion` string on each embedded template. Resources like `KubeadmControlPlaneTemplate`, `KubeadmConfigTemplate`, and `DockerMachineTemplate` now carry `v1beta2`. The same applies to the `ClusterClass` object itself.

Infrastructure providers that have not yet adopted `v1beta2` (CAPZ, CAPG, and CAPV at the time of this release) keep their existing `v1beta1` version. See the updated examples under `examples/clusterclasses/` for the correct version string for each cloud.

## Cluster

The `Cluster` resource now references its `ClusterClass` through a structured object rather than an inline string. In `v1beta1`, the class name and namespace were specified as `spec.topology.class` and `spec.topology.classNamespace`. In `v1beta2` both fields are replaced by a single `spec.topology.classRef` object.

Before:
```yaml
apiVersion: cluster.x-k8s.io/v1beta1
kind: Cluster
spec:
  topology:
    class: my-clusterclass
    classNamespace: capi-system
```

After:
```yaml
apiVersion: cluster.x-k8s.io/v1beta2
kind: Cluster
spec:
  topology:
    classRef:
      name: my-clusterclass
      namespace: capi-system
```

The rest of the `Cluster` spec (network settings, `controlPlane.replicas`, `workers.machineDeployments`, `variables`, and `version`) is unchanged.

## RKE2 provider (CAPRKE2)

This release ships CAPRKE2 v0.23.1, up from v0.22.1. The `v1beta2` API for the RKE2 bootstrap and control-plane resources was introduced in v0.22.0 alongside support for CAPI v1.11, and v0.23.0 completed that work by bumping to CAPI v1.12.2. Both v0.22.1 and v0.23.1 are patch releases on top of those respective minor versions.

### New v1beta2 API

CAPRKE2 now serves its own `v1beta2` API for `RKE2ControlPlane`, `RKE2ControlPlaneTemplate`, and `RKE2Config`/`RKE2ConfigTemplate`. The `v1beta1` API remains available but several fields that had been marked deprecated in `v1beta1` are now removed in `v1beta2`.

### Removed fields in RKE2ControlPlane

The following fields have been removed from `RKE2ControlPlane.spec` in `v1beta2`:

- `infrastructureRef` — use `spec.machineTemplate.spec.infrastructureRef` instead.
- `nodeDrainTimeout` — use `spec.machineTemplate.spec.deletion.nodeDrainTimeout` instead.

### Renamed timeout fields

The timeout fields in `RKE2ControlPlaneMachineTemplate` have been moved under a `deletion` sub-object and renamed. They have also changed type from `metav1.Duration` to `int32`, and now expect values expressed in seconds rather than a duration string.

| v1beta1 | v1beta2 |
|---|---|
| `nodeDrainTimeout` | `deletion.nodeDrainTimeoutSeconds` |
| `nodeVolumeDetachTimeout` | `deletion.nodeVolumeDetachTimeoutSeconds` |
| `nodeDeletionTimeout` | `deletion.nodeDeletionTimeoutSeconds` |

The `RKE2ControlPlaneMachineTemplate` object also now requires a `spec` field.

### Status fields

Several status fields have moved under `status.deprecated` in `v1beta2`, to allow a transition period while controllers and tools catch up:

- `conditions`
- `failureReason` and `failureMessage`
- `updatedReplicas`, `readyReplicas`, and `unavailableReplicas`

The following fields have been removed from the top-level status entirely:

- `ready`
- `initialized`
- `dataSecretName`

An RKE2 cluster is now considered initialized when `status.initialization.controlPlaneInitialized` is `true`. Status conditions are also using the standard `metav1.Conditions` type rather than the CAPI-specific condition type, which aligns them with the broader Kubernetes ecosystem.

### v1alpha1 deprecation

The `v1alpha1` API was deprecated in CAPRKE2 v0.22.0. If you are still using `v1alpha1` resources, now is a good time to migrate to at least `v1beta1`.

### Deletion bug fix

Both v0.22.1 and v0.23.1 include a fix for an intermittent issue where cluster deletion could stall indefinitely due to resources not being cleaned up in the correct order.

## Provider versions

All bundled providers have been updated alongside the core CAPI bump. Every provider in this release is at v1.11 or later. The full list is in `internal/controllers/clusterctl/config-prime.yaml`. If you pin provider versions yourself through a `CAPIProvider` resource or a `ClusterctlConfig`, verify that the versions you use are compatible with the `v1beta2` contract.

## Backward compatibility

CAPI v1.12 continues to serve `v1beta1` resources alongside `v1beta2`. Existing resources on your cluster will keep working without an immediate migration, as the API server converts between the two versions transparently. That said, any new manifests you write should target `v1beta2`, and you should plan to migrate your existing ones over time. The `v1beta1` API is expected to be removed in a future CAPI release once the ecosystem has had time to settle on `v1beta2`.

## Steps to take before upgrading

1. Update any `ClusterClass` manifests you maintain: rename `ref` to `templateRef`, remove the `template` wrapper in the workers section, and update core CAPI `apiVersion` strings to `v1beta2`.
2. Update any `Cluster` manifests: replace `spec.topology.class` and `spec.topology.classNamespace` with `spec.topology.classRef.name` and `spec.topology.classRef.namespace`.
3. If you use CAPRKE2 `v1beta2` resources, remove any direct `infrastructureRef` or `nodeDrainTimeout` fields from `RKE2ControlPlane.spec` and move them to `spec.machineTemplate.spec`.
4. Update any code or tooling that reads RKE2 status conditions: they are now standard `metav1.Conditions` and initialization state is reported at `status.initialization.controlPlaneInitialized`.
5. If you pin provider versions manually, ensure they meet the v1.11 minimum.
6. Check the updated examples in `examples/clusterclasses/` and `test/e2e/data/cluster-templates/` for complete, working manifests for each cloud.
