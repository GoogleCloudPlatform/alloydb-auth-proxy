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

# Use the latest stable golang 1.x to compile to a binary
FROM --platform=$BUILDPLATFORM golang:1 as build

WORKDIR /go/src/alloydb-auth-proxy
COPY . .

ARG TARGETOS
ARG TARGETARCH

RUN go get ./...
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags "-X github.com/GoogleCloudPlatform/alloydb-auth-proxy/cmd.metadataString=container"

# Final Stage
FROM gcr.io/distroless/static:nonroot@sha256:627d6c5a23ad24e6bdff827f16c7b60e0289029b0c79e9f7ccd54ae3279fb45f

LABEL org.opencontainers.image.source="https://github.com/GoogleCloudPlatform/alloydb-auth-proxy"

COPY --from=build --chown=nonroot /go/src/alloydb-auth-proxy/alloydb-auth-proxy /alloydb-auth-proxy
# set the uid as an integer for compatibility with runAsNonRoot in Kubernetes
USER 65532
ENTRYPOINT ["/alloydb-auth-proxy"]
