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
# Deploy the DataHub MCP server into the cluster and wait for it to be ready.
# Requires DataHub (GMS) already up (make datahub-up) and the image built
# (make datahub-mcp-build).
. "$(dirname "$0")/lib.sh"

log "deploying DataHub MCP server (namespace datahub)"
kubectl apply -f "$DEPLOY_DIR/datahub/mcp/deployment.yaml"

log "waiting for datahub-mcp rollout"
kubectl -n datahub rollout status deploy/datahub-mcp --timeout=120s

log "DataHub MCP ready — endpoint http://localhost:8091/mcp"
log "next: point your own MCP client (Claude Code, Kiro CLI) at the operator and DataHub MCP endpoints; see the docs"
