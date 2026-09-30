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
# Vendor the pinned DataHub charts so installation does not need a chart repo.
. "$(dirname "$0")/lib.sh"

DATAHUB_REPO="https://helm.datahubproject.io/"

log "adding and updating the DataHub Helm repository"
helm repo add datahub "$DATAHUB_REPO" >/dev/null 2>&1 || true
helm repo update datahub

log "available versions (pin them in deploy/versions.lock):"
echo "--- datahub/datahub ---"
helm search repo datahub/datahub --versions | head -5 || true
echo "--- datahub/datahub-prerequisites ---"
helm search repo datahub/datahub-prerequisites --versions | head -5 || true

pull() { # repo/chart  version  destdir
  local chart="$1" version="$2" dest="$3"
  [ -n "$version" ] || {
    warn "no pinned version for $chart; pin it in versions.lock and retry"
    return 0
  }
  mkdir -p "$dest"
  log "pull $chart@$version -> $dest"
  helm pull "$chart" --version "$version" -d "$dest" || die "failed to pull $chart@$version"
}

pull datahub/datahub "${DATAHUB_HELM_CHART_VERSION:-}" "$DEPLOY_DIR/datahub/charts"
pull datahub/datahub-prerequisites "${DATAHUB_PREREQ_CHART_VERSION:-}" "$DEPLOY_DIR/datahub/charts"

log "vendored charts:"
find "$DEPLOY_DIR" -name '*.tgz' | sort
