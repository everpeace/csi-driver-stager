NAME          := csi-driver-stager
VERSION       := $(if $(VERSION),$(VERSION),$(shell cat ./VERSION)-dev)
REVISION      := $(shell git rev-parse --short HEAD)
IMAGE_PREFIX  ?= everpeace/
IMAGE_TAG     ?= $(VERSION)
LDFLAGS       := -ldflags="-s -w -X \"main.Version=$(VERSION)\" -X \"main.Revision=$(REVISION)\" -extldflags \"-static\""
OUTDIR        ?= ./dist
BUILADH_IMG   := quay.io/buildah/stable:v1.12.0

.DEFAULT_GOAL := build

# env
export GO111MODULE=on
export CGO_ENABLED=0

.PHONY: setup
setup:
	go get -u github.com/golangci/golangci-lint/cmd/golangci-lint
	go get -u golang.org/x/tools/cmd/goimports

.PHONY: fmt
fmt:
	goimports -w cmd/ pkg/

.PHONY: lint
lint: fmt
	golangci-lint run --config .golangci.yml --deadline 30m

.PHONY: build
build: fmt lint
	go build -tags netgo -installsuffix netgo $(LDFLAGS) -o $(OUTDIR)/$(NAME) cmd/stager/image/main.go

.PHONY: test
test: fmt lint
	go test  ./...

test-debug: fmt lint
	dlv test --headless --listen=:2345 --api-version=2 $(WHAT)

.PHONY: devcontainer clean-devcontainer
devcontainer:
	cd .devcontainer && docker-compose up -d
clean-devcontainer:
	cd .devcontainer && docker-compose down -v

.PHONY: test-in-devcontainer
test-in-devcontainer: devcontainer
	cd .devcontainer && docker-compose exec workspace make test

.PHONY: test-debug-in-devcontainer
test-debug-in-devcontainer: devcontainer
	cd .devcontainer && docker-compose exec workspace make test-debug WHAT=$(WHAT)

.PHONY: clean
clean:
	rm -rf "$(OUTDIR)"

.PHONY: build-docker
build-docker:
	docker build -t $(shell make docker-tag) --build-arg BUILADH_IMG=$(BUILADH_IMG) --target runtime .

.PHONY: docker-tag
docker-tag:
	@echo $(IMAGE_PREFIX)$(NAME):$(IMAGE_TAG)
