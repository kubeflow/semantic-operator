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
# Deploy the single TLS+auth Trino engine. It mounts the trino-tls and
# trino-passwords Secrets created by hack/trino-secrets.sh (a Make
# prerequisite), so those must exist first. Readiness is the engine's own
# HTTPS /v1/info/state probe, so a successful rollout means Trino is serving.
set -euo pipefail

ROOT_DIR="$(git rev-parse --show-toplevel)"
CLUSTER_NAME="${KIND_CLUSTER_NAME:-semantic-operator-dev}"
KUBECONFIG_PATH="${KIND_KUBECONFIG:-$ROOT_DIR/.kube/config}"
NAMESPACE="${KIND_NAMESPACE:-semantic-system}"
RESOURCES="$ROOT_DIR/test/e2e/trino/resources.yaml"

KUBECTL=(kubectl --kubeconfig "$KUBECONFIG_PATH" --context "kind-$CLUSTER_NAME")
KUBECTL_NS=("${KUBECTL[@]}" --namespace "$NAMESPACE")

"${KUBECTL[@]}" create namespace "$NAMESPACE" --dry-run=client -o yaml |
  "${KUBECTL[@]}" apply -f -

"${KUBECTL_NS[@]}" apply -f "$RESOURCES"
# The configuration uses subPath mounts, so restart after every apply.
"${KUBECTL_NS[@]}" rollout restart deployment/trino
"${KUBECTL_NS[@]}" rollout status deployment/trino --timeout=5m

echo "Trino is ready at https://trino.$NAMESPACE.svc.cluster.local:8443"
