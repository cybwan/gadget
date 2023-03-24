#!make

TARGETS      	:= darwin/amd64 darwin/arm64 linux/amd64 linux/arm64
BINNAME      	?= gadget
DIST_DIRS    	:= find * -type d -exec
LOCAL_REGISTRY 	?= local.registry
CTR_REGISTRY 	?= cybwan
CTR_TAG      	?= latest
BUILDX  	 	?= osm

GOPATH = $(shell go env GOPATH)
GOBIN  = $(GOPATH)/bin
GOX    = go run github.com/mitchellh/gox
SHA256 = sha256sum
ifeq ($(shell uname),Darwin)
	SHA256 = shasum -a 256
endif

DOCKER_REGISTRY ?= docker.io/library
UBUNTU_VERSION ?= 20.04
KERNEL_VERSION ?= v5.4

VERSION ?= dev
BUILD_DATE ?=
GIT_SHA=$$(git rev-parse HEAD)
DOCKER_GO_VERSION = 1.19
DOCKER_BUILDX_PLATFORM ?= linux/amd64
# Value for the --output flag on docker buildx build.
# https://docs.docker.com/engine/reference/commandline/buildx_build/#output
DOCKER_BUILDX_OUTPUT ?= type=registry

LDFLAGS ?= "-X $(BUILD_DATE_VAR)=$(BUILD_DATE) -X $(BUILD_VERSION_VAR)=$(VERSION) -X $(BUILD_GITCOMMIT_VAR)=$(GIT_SHA) -s -w"

# Installed Go version
# This is the version of Go going to be used to compile this project.
# It will be compared with the minimum requirements for ECNET.
GO_VERSION_MAJOR = $(shell go version | cut -c 14- | cut -d' ' -f1 | cut -d'.' -f1)
GO_VERSION_MINOR = $(shell go version | cut -c 14- | cut -d' ' -f1 | cut -d'.' -f2)
GO_VERSION_PATCH = $(shell go version | cut -c 14- | cut -d' ' -f1 | cut -d'.' -f3)
ifeq ($(GO_VERSION_PATCH),)
GO_VERSION_PATCH := 0
endif

check-env:
ifndef CTR_REGISTRY
	$(error CTR_REGISTRY environment variable is not defined; see the .env.example file for more information; then source .env)
endif
ifndef CTR_TAG
	$(error CTR_TAG environment variable is not defined; see the .env.example file for more information; then source .env)
endif

.PHONY: go-vet
go-vet:
	go vet ./...

.PHONY: go-lint
go-lint:
	docker run --rm -v $$(pwd):/app -w /app golangci/golangci-lint:v1.50 golangci-lint run --config .golangci.yml

.PHONY: go-fmt
go-fmt:
	go fmt ./...

lint-c:
	clang-format --Werror -n *.c *.h

format-c:
	find . -regex '.*\.\(c\|h\)' -exec clang-format -style=file -i {} \;

.PHONY: docker-build-gadget-ubuntu
docker-build-gadget-ubuntu:
	docker buildx build --builder ${BUILDX} \
	--platform=$(DOCKER_BUILDX_PLATFORM) \
	-o $(DOCKER_BUILDX_OUTPUT) \
	-t $(LOCAL_REGISTRY)/ubuntu:$(UBUNTU_VERSION) \
	--build-arg DOCKER_REGISTRY=$(DOCKER_REGISTRY) \
	--build-arg UBUNTU_VERSION=$(UBUNTU_VERSION) \
	-f ./dockerfiles/Dockerfile.ubuntu .

.PHONY: docker-build-gadget-alpine
docker-build-gadget-alpine:
	docker buildx build --builder ${BUILDX} \
	--platform=$(DOCKER_BUILDX_PLATFORM) -o $(DOCKER_BUILDX_OUTPUT) \
	--build-arg DOCKER_REGISTRY=$(DOCKER_REGISTRY) \
	--build-arg CTR_REGISTRY=$(CTR_REGISTRY) \
	--build-arg LOCAL_REGISTRY=$(LOCAL_REGISTRY) \
	--build-arg CTR_TAG=$(CTR_TAG) \
	-t $(CTR_REGISTRY)/alpine:$(CTR_TAG) \
	-f dockerfiles/Dockerfile.alpine .

.PHONY: docker-build-gadget-dnsserver
docker-build-gadget-dnsserver:
	docker buildx build --builder ${BUILDX} \
	--platform=$(DOCKER_BUILDX_PLATFORM) -o $(DOCKER_BUILDX_OUTPUT) \
	--build-arg DOCKER_REGISTRY=$(DOCKER_REGISTRY) \
	--build-arg CTR_REGISTRY=$(CTR_REGISTRY) \
	--build-arg LOCAL_REGISTRY=$(LOCAL_REGISTRY) \
	--build-arg CTR_TAG=$(CTR_TAG) \
	--build-arg GO_VERSION=$(DOCKER_GO_VERSION) \
	--build-arg LDFLAGS=$(LDFLAGS) \
	-t $(CTR_REGISTRY)/dnsserver:$(CTR_TAG) \
	-f dockerfiles/Dockerfile.dnsserver .

GADGET_TARGETS = gadget-alpine gadget-dnsserver
DOCKER_GADGET_TARGETS = $(addprefix docker-build-, $(GADGET_TARGETS))

.PHONY: docker-build
docker-build: $(DOCKER_GADGET_TARGETS)

.PHONY: buildx-context
buildx-context:
	@if ! docker buildx ls | grep -q "^${BUILDX}"; then docker buildx create --name ${BUILDX} --driver-opt network=host; fi

.PHONY: docker-build-cross-gadget docker-build-cross
docker-build-cross-gadget: DOCKER_BUILDX_PLATFORM=linux/amd64,linux/arm64
docker-build-cross-gadget: docker-build
docker-build-cross: docker-build-cross-gadget

.PHONY: shellcheck
shellcheck:
	shellcheck -x $(shell find . -name '*.sh')

