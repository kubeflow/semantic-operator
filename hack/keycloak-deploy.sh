#!/usr/bin/env bash
# Copyright The Kubeflow Authors.
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
# Deploy Keycloak and import the semantic realm. The single Trino engine enables
# JWT authentication and fetches the realm JWKS when it starts, so Keycloak must
# be running before the engine. Keycloak runs in dev mode with an in-memory
# database, so a pod restart re-imports the realm from the ConfigMap.
set -euo pipefail

ROOT_DIR="$(git rev-parse --show-toplevel)"
CLUSTER_NAME="${KIND_CLUSTER_NAME:-semantic-operator-dev}"
KUBECONFIG_PATH="${KIND_KUBECONFIG:-$ROOT_DIR/.kube/config}"
NAMESPACE="${KIND_NAMESPACE:-semantic-system}"
REALM="$ROOT_DIR/test/e2e/auth/keycloak/realm.json"
RESOURCES="$ROOT_DIR/test/e2e/auth/keycloak/resources.yaml"

KUBECTL=(kubectl --kubeconfig "$KUBECONFIG_PATH" --context "kind-$CLUSTER_NAME" --namespace "$NAMESPACE")

"${KUBECTL[@]}" create namespace "$NAMESPACE" --dry-run=client -o yaml | "${KUBECTL[@]}" apply -f -
"${KUBECTL[@]}" create configmap keycloak-realm \
  --from-file=realm.json="$REALM" --dry-run=client -o yaml | "${KUBECTL[@]}" apply -f -
"${KUBECTL[@]}" apply -f "$RESOURCES"
"${KUBECTL[@]}" rollout status deployment/keycloak --timeout=5m

echo "Keycloak is ready at http://keycloak.$NAMESPACE.svc:8080"
