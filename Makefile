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
	goimports -w cmd/ pkg/ main.go

.PHONY: lint
lint: fmt
	golangci-lint run --config .golangci.yml --deadline 30m

.PHONY: build
build: fmt lint
	go build -tags netgo -installsuffix netgo $(LDFLAGS) -o $(OUTDIR)/$(NAME) main.go

.PHONY: test
test: fmt lint
	go test -count=1 ./...

.PHONY: test-debug
test-debug: fmt lint
	dlv test --headless --listen=:2345 --api-version=2 $(WHAT)

.PHONY: run-driver
run-driver: build
	[ ! -e /tmp/$(NAME).sock ] || rm -rf /tmp/$(NAME).sock
	$(OUTDIR)/$(NAME) --logpretty --loglevel=trace --endpoint=unix:///tmp/$(NAME).sock --nodeid=testnode image

.PHONY: debug-driver
debug-driver:
	[ ! -e /tmp/$(NAME).sock ] || rm -rf /tmp/$(NAME).sock
	dlv debug --headless --listen=:2345 --api-version=2 -- --logpretty=true --loglevel=trace --endpoint=unix:///tmp/$(NAME).sock --nodeid=testnode image


# filtering test by ginkgo.focus because the driver provides only Identity/Node Services.
.PHONY: csi-sanity
csi-sanity:
	csi-sanity -ginkgo.v \
	  -ginkgo.focus "Identity Service|Node Service" \
	  -csi.endpoint unix:///tmp/$(NAME).sock \
      -csi.createstagingpathcmd ./scripts/csi-sanity/mkdir.sh \
      -csi.createmountpathcmd ./scripts/csi-sanity/mkdir.sh \
      -csi.removestagingpathcmd ./scripts/csi-sanity/mkdir/rmdir.sh \
      -csi.removemountpathcmd ./scripts/csi-sanity/mkdir/rmdir.sh

.PHONY: build-devcontainer devcontainer clean-devcontainer
build-devcontainer:
	cd .devcontainer && docker-compose build
devcontainer: build-devcontainer
	cd .devcontainer && docker-compose up -d
clean-devcontainer:
	cd .devcontainer && docker-compose down -v

.PHONY: clean
clean:
	rm -rf "$(OUTDIR)"

.PHONY: build-docker
build-docker:
	docker build -t $(shell make docker-tag) --build-arg BUILADH_IMG=$(BUILADH_IMG) --target runtime .

.PHONY: docker-tag
docker-tag:
	@echo $(IMAGE_PREFIX)$(NAME):$(IMAGE_TAG)
