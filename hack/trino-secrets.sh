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
# Create the Trino TLS keystore and password Secrets plus the operator engine
# credentials. The keystore and password file are generated only when the
# trino-tls Secret is absent, so reruns do not churn the engine pod. The
# operator credential Secrets are always upserted and match the password file.
#
# No python and no htpasswd: the bcrypt $2y password file comes from the Go
# helper in hack/bcrypt, and the keystore comes from openssl.
set -euo pipefail

ROOT_DIR="$(git rev-parse --show-toplevel)"
CLUSTER_NAME="${KIND_CLUSTER_NAME:-semantic-operator-dev}"
KUBECONFIG_PATH="${KIND_KUBECONFIG:-$ROOT_DIR/.kube/config}"
NAMESPACE="${KIND_NAMESPACE:-semantic-system}"
MANAGER_USER="${TRINO_MANAGER_USER:-semantic-manager}"
MANAGER_PW="${TRINO_MANAGER_PASSWORD:-manager}"
SERVER_USER="${TRINO_SERVER_USER:-semantic-server}"
SERVER_PW="${TRINO_SERVER_PASSWORD:-server}"

KUBECTL=(kubectl --kubeconfig "$KUBECONFIG_PATH" --context "kind-$CLUSTER_NAME" --namespace "$NAMESPACE")

"${KUBECTL[@]}" create namespace "$NAMESPACE" --dry-run=client -o yaml | "${KUBECTL[@]}" apply -f -

# Operator engine credentials. The manager introspects and publishes views; the
# server only selects. Both authenticate to Trino with these static passwords,
# which must match the password file generated below.
"${KUBECTL[@]}" create secret generic engine-manager-cred \
  --from-literal=password="$MANAGER_PW" --dry-run=client -o yaml | "${KUBECTL[@]}" apply -f -
"${KUBECTL[@]}" create secret generic engine-server-cred \
  --from-literal=password="$SERVER_PW" --dry-run=client -o yaml | "${KUBECTL[@]}" apply -f -

if "${KUBECTL[@]}" get secret trino-tls >/dev/null 2>&1; then
  echo "trino-tls exists; keeping the current keystore and password file"
  exit 0
fi

work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

# Self-signed cert and PKCS12 keystore (password: changeit). Clients skip
# verification for this isolated test engine.
openssl req -x509 -newkey rsa:2048 -nodes -days 365 \
  -keyout "$work/tls.key" -out "$work/tls.crt" -subj "/CN=trino" \
  -addext "subjectAltName=DNS:trino,DNS:trino.$NAMESPACE.svc,DNS:trino.$NAMESPACE.svc.cluster.local,DNS:localhost,IP:127.0.0.1"
openssl pkcs12 -export -inkey "$work/tls.key" -in "$work/tls.crt" \
  -out "$work/keystore.p12" -passout pass:changeit -name trino

# bcrypt $2y password file via the Go helper.
( cd "$ROOT_DIR" && CGO_ENABLED=0 go run ./hack/bcrypt \
    "$MANAGER_USER=$MANAGER_PW" "$SERVER_USER=$SERVER_PW" ) > "$work/password.db"

"${KUBECTL[@]}" create secret generic trino-tls \
  --from-file=keystore.p12="$work/keystore.p12" --dry-run=client -o yaml | "${KUBECTL[@]}" apply -f -
"${KUBECTL[@]}" create secret generic trino-passwords \
  --from-file=password.db="$work/password.db" --dry-run=client -o yaml | "${KUBECTL[@]}" apply -f -

echo "trino-tls, trino-passwords, engine-manager-cred, engine-server-cred are ready in $NAMESPACE"
