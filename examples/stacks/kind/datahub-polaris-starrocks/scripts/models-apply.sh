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
# Apply the local SemanticModels (catalog=iceberg, database
# saas_accounts_demo) and show reconciliation status. The manager
# validates each model, drift-checks its bindings against the live StarRocks
# iceberg catalog, publishes the compiled artifact, and creates governed views.
. "$(dirname "$0")/lib.sh"

kubectl get namespace semantic-system >/dev/null 2>&1 || die "namespace semantic-system missing; run 'make operator-up' first"

log "applying local SemanticModels from $DEPLOY_DIR/models"
kubectl apply -f "$DEPLOY_DIR/models/"

log "waiting for models to validate and publish (Published implies no blocking drift)"
kubectl -n semantic-system wait --for=condition=Published --all --timeout=180s semanticmodels \
  || warn "not all models reached Published within the timeout; inspect the status below"

log "status (want Validated=True, Published=True, Drift=False):"
kubectl -n semantic-system get semanticmodels
