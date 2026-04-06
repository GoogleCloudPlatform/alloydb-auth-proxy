# Copyright 2026 Google LLC
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

# Registry proxy for building images
REGISTRY_PROXY ?=

BASE_IMAGE_alpine = alpine

ifdef REGISTRY_PROXY
  IMAGE_alpine = $(REGISTRY_PROXY)/$(BASE_IMAGE_alpine)
else
  IMAGE_alpine = $(BASE_IMAGE_alpine)
endif

# Single platform build defaults
TARGETOS ?= linux
TARGETARCH ?= amd64

# Multi-platform build defaults
PLATFORMS ?= linux/amd64,linux/arm64

# Translate platforms (e.g. linux/amd64,linux/arm64) to binary targets (e.g. alloydb-auth-proxy.linux.amd64 alloydb-auth-proxy.linux.arm64)
comma := ,
PLATFORM_LIST = $(subst /,.,$(subst $(comma), ,$(PLATFORMS)))
BINARIES = $(patsubst %, alloydb-auth-proxy.%, $(PLATFORM_LIST))

# Default target
all: build-image

# Pattern rule for building binaries
alloydb-auth-proxy.%:
	@OS_ARCH="$*"; \
	OS=$${OS_ARCH%%.*}; \
	ARCH=$${OS_ARCH##*.}; \
	echo "Building binary for $$OS/$$ARCH..."; \
	CGO_ENABLED=0 GOOS=$$OS GOARCH=$$ARCH go build \
		-ldflags "-X github.com/GoogleCloudPlatform/alloydb-auth-proxy/cmd.metadataString=container" \
		-o bin/binary/$@

.PHONY: build-binary
build-binary: alloydb-auth-proxy.$(TARGETOS).$(TARGETARCH)

.PHONY: build-binaries-multi
build-binaries-multi: $(BINARIES)

.PHONY: build-image
build-image: build-binary
	docker build \
		--build-arg TARGETOS=$(TARGETOS) \
		--build-arg TARGETARCH=$(TARGETARCH) \
		-t alloydb-auth-proxy:latest \
		-f Dockerfile .

.PHONY: build-image-alpine
build-image-alpine: build-binary
	docker build \
		--build-arg TARGETOS=$(TARGETOS) \
		--build-arg TARGETARCH=$(TARGETARCH) \
		--build-arg IMAGE_NAME_WITH_PROXY=$(IMAGE_alpine) \
		-t alloydb-auth-proxy:latest-alpine \
		-f Dockerfile.alpine .

# Multi-arch build targets using docker buildx and outputs to file not docker cache
.PHONY: build-image-multi-default
build-image-multi-default: build-binaries-multi
	docker buildx build \
		--platform $(PLATFORMS) \
  	    --output=type=oci,dest="bin/image/image-default.tar" \
		-f Dockerfile $(BUILDX_ARGS) .

.PHONY: build-image-multi-alpine
build-image-multi-alpine: build-binaries-multi
	docker buildx build \
		--platform $(PLATFORMS) \
		--build-arg IMAGE_NAME_WITH_PROXY=$(IMAGE_alpine) \
  	    --output=type=oci,dest="bin/image/image-alpine.tar" \
		-f Dockerfile.alpine $(BUILDX_ARGS) .

.PHONY: clean
clean:
	rm -f alloydb-auth-proxy.*
	rm -f bin/image/*
