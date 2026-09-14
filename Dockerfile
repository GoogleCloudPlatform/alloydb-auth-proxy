# Copyright 2019 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

FROM gcr.io/distroless/static:nonroot@sha256:e2e927ec666bae08560abb3c55d0659eceabb657f56b6782ab500a9fc7f555e3

ARG TARGETOS
ARG TARGETARCH

LABEL org.opencontainers.image.source="https://github.com/GoogleCloudPlatform/alloydb-auth-proxy"
ENV ALLOYDB_PROXY_METADATA="container"

COPY --chown=nonroot bin/binary/alloydb-auth-proxy.${TARGETOS}.${TARGETARCH} /alloydb-auth-proxy
# set the uid as an integer for compatibility with runAsNonRoot in Kubernetes
USER 65532
ENTRYPOINT ["/alloydb-auth-proxy"]
