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
# Polaris bootstrap + server. Catalog creation (S3->Garage) is a
# separate step (polaris-catalog.sh), added after it's validated against Garage.
. "$(dirname "$0")/lib.sh"

kubectl apply -f "$DEPLOY_DIR/namespaces.yaml"
kubectl apply -f "$DEPLOY_DIR/polaris/polaris.yaml"

log "waiting for bootstrap job (creates realm POLARIS + root principal)"
if ! kubectl -n account-demo wait --for=condition=complete job/polaris-bootstrap --timeout=180s; then
  if kubectl -n account-demo logs job/polaris-bootstrap 2>/dev/null | grep -qi 'already'; then
    log "realm already bootstrapped"
  else
    kubectl -n account-demo logs job/polaris-bootstrap 2>&1 | tail -20
    die "bootstrap job did not complete"
  fi
fi

log "waiting for polaris server (readiness gated on /q/health)"
kubectl -n account-demo rollout status deploy/polaris --timeout=240s
log "polaris server OK — catalog API :8181 (host: http://localhost:8181)"
log "next: polaris-catalog creates the Iceberg REST catalog 'account-demo' on Garage"
