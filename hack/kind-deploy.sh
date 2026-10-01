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
# Create or reuse the local kind cluster and write its context to .kube/config.
set -euo pipefail

ROOT_DIR="$(git rev-parse --show-toplevel)"
CLUSTER_NAME="${KIND_CLUSTER_NAME:-semantic-operator-dev}"
KUBECONFIG_PATH="${KIND_KUBECONFIG:-$ROOT_DIR/.kube/config}"
NODE_IMAGE="${KIND_NODE_IMAGE:-}"

mkdir -p "$(dirname "$KUBECONFIG_PATH")"

if kind get clusters | grep -Fxq "$CLUSTER_NAME"; then
  echo "Reusing kind cluster $CLUSTER_NAME"
  kind export kubeconfig --name "$CLUSTER_NAME" --kubeconfig "$KUBECONFIG_PATH"
else
  echo "Creating kind cluster $CLUSTER_NAME"
  args=(--name "$CLUSTER_NAME" --kubeconfig "$KUBECONFIG_PATH")
  if [[ -n "$NODE_IMAGE" ]]; then
    args+=(--image "$NODE_IMAGE")
  fi
  kind create cluster "${args[@]}"
fi

chmod 0600 "$KUBECONFIG_PATH"
echo "Kubeconfig: $KUBECONFIG_PATH"
echo "Context: kind-$CLUSTER_NAME"
