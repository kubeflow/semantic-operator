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
# Deploy OPA as the external, first-gate authorizer (namespace account-demo).
# The Rego in deploy/opa/policy.rego is the single source of truth; it is
# rendered into the opa-policy ConfigMap here, then the Deployment picks it up.
# Run after the operator is up so the server can reach it; models that set
# spec.governance.external.providerRef require it before they will serve queries.
. "$(dirname "$0")/lib.sh"

NS=account-demo
POLICY_FILE="$DEPLOY_DIR/opa/policy.rego"
[ -f "$POLICY_FILE" ] || die "policy not found: $POLICY_FILE"

log "creating opa-policy ConfigMap from policy.rego"
kubectl create configmap opa-policy -n "$NS" \
  --from-file=policy.rego="$POLICY_FILE" \
  --dry-run=client -o yaml | kubectl apply -f -

log "applying OPA Deployment and Service"
kubectl apply -f "$DEPLOY_DIR/opa/opa.yaml"

# Restart so a policy change is picked up even when the Deployment is unchanged.
kubectl -n "$NS" rollout restart deployment/opa >/dev/null
log "waiting for OPA rollout"
kubectl -n "$NS" rollout status deployment/opa --timeout=120s

log "OPA ready — decision endpoint http://opa.$NS.svc.cluster.local:8181/v1/data/semantic/query/allow"
log "next: models-apply"
