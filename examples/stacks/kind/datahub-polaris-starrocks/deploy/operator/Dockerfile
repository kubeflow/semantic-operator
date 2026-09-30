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
# Prebuilt-binary images for the local demo. Fully offline: the static
# manager/server binaries are compiled on the host from the repo root (CGO off,
# using the host Go module cache) and copied into the cached distroless base,
# so this build never runs `go mod download`. Build context is the directory
# holding the two binaries. Mirrors the runtime shape of the repo root Dockerfile.
FROM gcr.io/distroless/static:nonroot AS manager
WORKDIR /
COPY manager /manager
USER 65532:65532
ENTRYPOINT ["/manager"]

FROM gcr.io/distroless/static:nonroot AS server
WORKDIR /
COPY server /server
USER 65532:65532
ENTRYPOINT ["/server"]
