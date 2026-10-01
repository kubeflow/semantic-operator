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
# Shared Postgres (polaris + datahub databases).
. "$(dirname "$0")/lib.sh"

kubectl apply -f "$DEPLOY_DIR/namespaces.yaml"
kubectl apply -f "$DEPLOY_DIR/postgres/postgres.yaml"

log "waiting for postgres rollout"
kubectl -n account-demo rollout status deploy/postgres --timeout=180s

log "verifying databases exist"
dbs=$(kubectl -n account-demo exec deploy/postgres -- \
  psql -U postgres -tAc "SELECT datname FROM pg_database WHERE datname IN ('polaris','datahub') ORDER BY 1" \
  | tr -d '\r' | paste -sd, -)
log "databases present: ${dbs:-<none>}"
[ "$dbs" = "datahub,polaris" ] || die "expected databases datahub,polaris — got '${dbs}' (check init logs: kubectl -n account-demo logs deploy/postgres)"
log "postgres OK"
