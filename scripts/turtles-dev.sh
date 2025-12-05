#!/usr/bin/env bash

# Copyright © 2023 - 2024 SUSE LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -xe

RANCHER_HOSTNAME=$1
if [ -z "$RANCHER_HOSTNAME" ]; then
    echo "You must pass a rancher host name"
    exit 1
fi

RANCHER_CHANNEL=${RANCHER_CHANNEL:-latest}
RANCHER_PASSWORD=${RANCHER_PASSWORD:-rancheradmin}
RANCHER_VERSION=${RANCHER_VERSION:-v2.13.0-rc1}
RANCHER_IMAGE_TAG=${RANCHER_IMAGE_TAG:-$RANCHER_VERSION} # Set RANCHER_IMAGE_TAG=head to test with latest build
RANCHER_IMAGE=${RANCHER_IMAGE:-rancher/rancher:$RANCHER_IMAGE_TAG}
CLUSTER_NAME=${CLUSTER_NAME:-capi-test}
USE_TILT_DEV=${USE_TILT_DEV:-true}
TURTLES_VERSION=${TURTLES_VERSION:-dev}
TURTLES_IMAGE=${TURTLES_IMAGE:-ghcr.io/rancher/turtles:$TURTLES_VERSION}

GITEA_PASSWORD=${GITEA_PASSWORD:-giteaadmin}
GITEA_HOSTNAME="gitea.${RANCHER_HOSTNAME}"
RANCHER_CHARTS_REPO_DIR=${RANCHER_CHARTS_REPO_DIR}
RANCHER_CHART_DEV_VERSION=${RANCHER_CHART_DEV_VERSION}
RANCHER_CHARTS_BASE_BRANCH=${RANCHER_CHARTS_BASE_BRANCH}

PANGOLIN_ENDPOINT=${PANGOLIN_ENDPOINT}
NEWT_NAMESPACE=${NEWT_NAMESPACE:-newt}
NEWT_ID=${NEWT_ID}
NEWT_SECRET=${NEWT_SECRET}

if [ -z "$NEWT_ID" ] || [ -z "$NEWT_SECRET" ] || [ -z "$PANGOLIN_ENDPOINT" ]; then
    echo "Error: NEWT_ID, NEWT_SECRET and PANGOLIN_ENDPOINT cannot be empty."
    exit 1
fi

BASEDIR=$(dirname "$0")

kind create cluster --config "$BASEDIR/kind-cluster-with-extramounts.yaml" --name $CLUSTER_NAME
docker pull $RANCHER_IMAGE
kind load docker-image $RANCHER_IMAGE --name $CLUSTER_NAME

kubectl rollout status deployment coredns -n kube-system --timeout=90s

helm repo add rancher-$RANCHER_CHANNEL https://releases.rancher.com/server-charts/$RANCHER_CHANNEL --force-update
helm repo add jetstack https://charts.jetstack.io --force-update
helm repo add gitea-charts https://dl.gitea.com/charts/ --force-update
helm repo add fossorial https://charts.fossorial.io --force-update
helm repo update

helm install cert-manager jetstack/cert-manager \
    --namespace cert-manager \
    --create-namespace \
    --set crds.enabled=true

# configure and install Newt for Pangolin
kubectl create ns ${NEWT_NAMESPACE}
kubectl create secret generic newt-cred -n ${NEWT_NAMESPACE} \
    --from-literal=PANGOLIN_ENDPOINT=${PANGOLIN_ENDPOINT} \
    --from-literal=NEWT_ID=${NEWT_ID} \
    --from-literal=NEWT_SECRET=${NEWT_SECRET}

helm install my-newt fossorial/newt \
    -n ${NEWT_NAMESPACE} --create-namespace \
    -f test/e2e/data/newt/values.yaml

helm install gitea gitea-charts/gitea \
    -f test/e2e/data/gitea/values.yaml \
    --set gitea.admin.password=$GITEA_PASSWORD \
    --wait

# Build and load the controller image
make docker-build-prime
kind load docker-image $TURTLES_IMAGE --name $CLUSTER_NAME

# Wait for Gitea to be accessible locally
until curl -s -o /dev/null -w "%{http_code}" https://${GITEA_HOSTNAME} | grep -q "200\|302\|301"; do
    echo "Waiting for Gitea to be accessible on ${GITEA_HOSTNAME}..."
    sleep 2
done
echo "Gitea is accessible locally!"

helm install rancher rancher-$RANCHER_CHANNEL/rancher \
    --namespace cattle-system \
    --create-namespace \
    --set bootstrapPassword=$RANCHER_PASSWORD \
    --set replicas=1 \
    --set hostname="$RANCHER_HOSTNAME" \
    --set image.tag=$RANCHER_IMAGE_TAG \
    --set debug=true \
    --version="$RANCHER_VERSION" \
    --wait

# Wait for Rancher to be accessible locally
until curl -s -o /dev/null -w "%{http_code}" https://${RANCHER_HOSTNAME} | grep -q "200\|302\|301"; do
    echo "Waiting for Rancher to be accessible on ${RANCHER_HOSTNAME}..."
    sleep 2
done
echo "Rancher is accessible!"

# Now setup Gitea repo and push charts
curl -X POST "https://gitea:$GITEA_PASSWORD@${GITEA_HOSTNAME}/api/v1/user/repos" \
    -H 'Accept: application/json' \
    -H 'Content-Type: application/json' \
    -d '{"name":"charts"}'

git -C $RANCHER_CHARTS_REPO_DIR remote add fork https://gitea:$GITEA_PASSWORD@${GITEA_HOSTNAME}/gitea/charts.git

echo "Pushing charts repository to Gitea (this may take several minutes)..."
PUSH_RETRIES=3
PUSH_COUNT=0
while [ $PUSH_COUNT -lt $PUSH_RETRIES ]; do
    PUSH_COUNT=$((PUSH_COUNT + 1))
    echo "Push attempt $PUSH_COUNT/$PUSH_RETRIES..."

    if git -C $RANCHER_CHARTS_REPO_DIR push fork --force 2>&1 | tee /tmp/git-push.log; then
        echo "Successfully pushed charts repository!"
        break
    else
        if [ $PUSH_COUNT -lt $PUSH_RETRIES ]; then
            echo "Push failed, waiting 10 seconds before retry..."
            sleep 10
        else
            echo "ERROR: Failed to push charts repository after $PUSH_RETRIES attempts."
            exit 1
        fi
    fi
done

envsubst <test/e2e/data/rancher/rancher-setting-patch.yaml | kubectl apply -f -
kubectl apply -f test/e2e/data/rancher/system-store-setting-patch.yaml

# Update Rancher deployment with environment variables pointing to Gitea charts
kubectl set env deployment/rancher -n cattle-system \
    CATTLE_CHART_DEFAULT_URL=https://gitea.$RANCHER_HOSTNAME/gitea/charts.git \
    CATTLE_CHART_DEFAULT_BRANCH=$RANCHER_CHARTS_BASE_BRANCH \
    CATTLE_RANCHER_TURTLES_VERSION=$RANCHER_CHART_DEV_VERSION

# Wait for Rancher to restart with new config
kubectl rollout status deployment/rancher -n cattle-system --timeout=300s

install_local_providers_chart() {
    make build-providers-chart

    # Wait for Turtles to be ready. This may take a few minutes before Rancher installs the system chart.
    # The providers chart depends on CAPIProvider crd.
    kubectl wait --for=create crds/capiproviders.turtles-capi.cattle.io --timeout=300s

    helm upgrade --install rancher-turtles-providers out/charts/rancher-turtles-providers \
        -n cattle-turtles-system \
        --set providers.bootstrapRKE2.manager.verbosity=5 \
        --set providers.controlplaneRKE2.manager.verbosity=5 \
        --set providers.bootstrapKubeadm.enabled=true \
        --set providers.bootstrapKubeadm.manager.verbosity=5 \
        --set providers.controlplaneKubeadm.enabled=true \
        --set providers.controlplaneKubeadm.manager.verbosity=5 \
        --set providers.infrastructureDocker.enabled=true \
        --set providers.infrastructureDocker.manager.verbosity=5 \
        --set providers.infrastructureAWS.enabled=true \
        --set providers.infrastructureAWS.manager.verbosity=5 \
        --set providers.infrastructureAzure.enabled=true \
        --set providers.infrastructureAzure.manager.verbosity=5 \
        --set providers.infrastructureGCP.enabled=true \
        --set providers.infrastructureGCP.manager.verbosity=5 \
        --set providers.infrastructureGCP.variables.GCP_B64ENCODED_CREDENTIALS="" \
        --set providers.infrastructureVSphere.enabled=true \
        --set providers.infrastructureVSphere.manager.verbosity=5 \
        --create-namespace --wait \
        --timeout 180s
}

echo "Installing local Rancher Turtles Providers..."
install_local_providers_chart

if [ "$USE_TILT_DEV" == "true" ]; then
    kubectl wait --for=create deployments/rancher-turtles-controller-manager --namespace cattle-turtles-system --timeout=300s
    echo "Using Tilt for development..."
    tilt up
fi
