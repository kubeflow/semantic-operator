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
# /// script
# requires-python = ">=3.10"
# dependencies = [
#   "acryl-datahub[iceberg]==1.6.0",
# ]
# ///
"""Run a DataHub ingestion recipe via uv with inline (PEP 723) dependencies.

No uv project: `uv run datahub_ingest.py <recipe.yaml>` builds an ephemeral env
with the DataHub CLI + the Iceberg source and runs it. The DataHub config loader
expands ${ENV_VAR} references in the recipe (datahub-ingest.sh exports them).

The acryl-datahub version is pinned to match the deployed GMS (v1.6.0); adjust
the single dependency line if the server version changes.
"""
import sys

from datahub.entrypoints import main

if __name__ == "__main__":
    recipe = sys.argv[1] if len(sys.argv) > 1 else "iceberg-recipe.yaml"
    sys.argv = ["datahub", "ingest", "-c", recipe]
    main()
