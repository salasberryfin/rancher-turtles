#!/bin/bash

# script-specific variables
CAPI_VERSION="${CAPI_VERSION:-latest}"
CAPI_RELEASE_URL="${CAPI_RELEASE_URL:-https://github.com/rancher-sandbox/cluster-api/releases/${CAPI_VERSION}/core-components.yaml}"
CORE_CAPI_NAMESPACE="${CORE_CAPI_NAMESPACE:-capi-system}"
OUTPUT_DIR="${OUTPUT_DIR:-/tmp}"
OUTPUT_FILE="${OUTPUT_FILE:-core-provider-configmap.yaml}"
CLUSTERCTL_DOWNLOAD_URL="${CAPI_UPSTREAM_URL:-https://github.com/kubernetes-sigs/cluster-api/releases/download/${CAPI_VERSION}/clusterctl-linux-amd64}"
CLUSTERCTL_CONFIG_PATH="${CLUSTERCTL_CONFIG_PATH:-/tmp/clusterctl.yaml}"

#cat >${CLUSTERCTL_CONFIG_PATH} <<EOF
#providers:
#  - name: "cluster-api"
#    url: ${CAPI_RELEASE_URL}
#    type: "CoreProvider"
#EOF

# parameters that must be substituted in CAPI manifest
export CAPI_DIAGNOSTICS_ADDRESS=${CAPI_DIAGNOSTICS_ADDRESS:=:8443}
export CAPI_INSECURE_DIAGNOSTICS=${CAPI_INSECURE_DIAGNOSTICS:=false}
export EXP_MACHINE_POOL=${EXP_MACHINE_POOL:=true}
export EXP_CLUSTER_RESOURCE_SET=${EXP_CLUSTER_RESOURCE_SET:=true}
export CLUSTER_TOPOLOGY=${CLUSTER_TOPOLOGY:=true}
export EXP_RUNTIME_SDK=${EXP_RUNTIME_SDK:=false}
export EXP_MACHINE_SET_PREFLIGHT_CHECKS=${EXP_MACHINE_SET_PREFLIGHT_CHECKS:=true}
export EXP_MACHINE_WAITFORVOLUMEDETACH_CONSIDER_VOLUMEATTACHMENTS=${EXP_MACHINE_WAITFORVOLUMEDETACH_CONSIDER_VOLUMEATTACHMENTS:=true}
export EXP_PRIORITY_QUEUE=${EXP_PRIORITY_QUEUE:=false}

## install clusterctl from source
#echo "Installing clusterctl from ${CLUSTERCTL_DOWNLOAD_URL}"
#curl -L ${CLUSTERCTL_DOWNLOAD_URL} -o clusterctl
#sudo install -o root -g root -m 0755 clusterctl /usr/local/bin/clusterctl

# install CAPI Operator plugin
kubectl krew index add operator https://github.com/kubernetes-sigs/cluster-api-operator.git
kubectl krew install operator/clusterctl-operator
kubectl operator version

# first set the conditional for this template
cat <<EOF >${OUTPUT_DIR}/${OUTPUT_FILE}
{{- if and (index .Values "cluster-api-operator" "cluster-api" "enabled") (index .Values "airGapped") }}
EOF

## use `clusterctl` to generate the body of the core CAPI template
#clusterctl generate provider --config ${CLUSTERCTL_CONFIG_PATH} --core cluster-api >>${OUTPUT_DIR}/core-provider-airgapped.yaml
#sed -i '/{{[^-]/d' ${OUTPUT_DIR}/core-provider-airgapped.yaml # this is needed to remove comments in the yaml manifest that contain '{{' which breaks Helm parsing

# use CAPI Operator plugin to generate ConfigMap with core CAPI components
kubectl operator preload --core cluster-api:${CORE_CAPI_NAMESPACE} -u ${CAPI_RELEASE_URL} >>${OUTPUT_DIR}/${OUTPUT_FILE}
sed -i '/{{[^-]/d' ${OUTPUT_DIR}/${OUTPUT_FILE} # this is needed to remove comments in the yaml manifest that contain '{{' which breaks Helm parsing

# close conditional for this template
cat <<EOF >>${OUTPUT_DIR}/${OUTPUT_FILE}
{{- end }}
EOF

# embed this in Turtles chart
mv ${OUTPUT_DIR}/${OUTPUT_FILE} ./charts/rancher-turtles/templates/${OUTPUT_FILE}
