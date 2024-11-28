GO ?= go

include $(CURDIR)/versions.mk

.PHONY: all
all: build

##@ General

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Build

BINARY_VERSION ?= $(GIT_TAG)
ifdef VERSION
	BINARY_VERSION = $(VERSION)
endif

# Only set Version if building a tag or VERSION is set
ifneq ($(BINARY_VERSION),)
	LDFLAGS += -X github.com/yeahdongcn/kustohelmize/internal/version.version=$(BINARY_VERSION)
endif

VERSION_METADATA = unreleased
# Clear the "unreleased" string in BuildMetadata
ifneq ($(GIT_TAG),)
	VERSION_METADATA =
endif

LDFLAGS += -X github.com/yeahdongcn/kustohelmize/internal/version.metadata=$(VERSION_METADATA)
LDFLAGS += -X github.com/yeahdongcn/kustohelmize/internal/version.gitCommit=$(GIT_COMMIT)
LDFLAGS += -X github.com/yeahdongcn/kustohelmize/internal/version.gitTreeState=$(GIT_DIRTY)
LDFLAGS += $(EXT_LDFLAGS)

.PHONY: build
build: ## Build the binary.
	GO111MODULE=on CGO_ENABLED=0 $(GO) build -o bin/kustohelmize -ldflags '$(LDFLAGS)' $(CURDIR)/main.go

##@ Test

.PHONY: examples
examples: build kubernetes-split-yaml ## Test the binary against the examples.
	cd examples/memcached-operator; KUSTOHELMIZE=../../bin/kustohelmize make helm

.PHONY: test
test: go-test build kubernetes-split-yaml 0100 0200 0300 0400 0500 ## Test the binary.

.PHONY: go-test
go-test:
	$(GO) test ./...

.PHONY: 0100
0100: build
	bin/kustohelmize create --from=test/testdata/0100_deployment.yaml test/output/0100/mychart

.PHONY: 0200
0200: build
	bin/kustohelmize create --from=test/testdata/0200_sample.yaml --version=1.0.0 --app-version=1.0.0 --description="Helm chart for testing" test/output/0200/mychart

.PHONY: 0300
0300: build
	bin/kustohelmize create --from=test/testdata/0300_sample.yaml --suppress-namespace --version=1.0.0 --app-version=1.0.0 --description="Helm chart with suppressed namespace" test/output/0300/no-ns-chart

.PHONY: 0400
0400: build
	bin/kustohelmize create --from=test/testdata/0400_issuer.yaml test/output/0400/mychart

.PHONY: 0500
0500: build
	bin/kustohelmize create --from=test/testdata/0500_deployment.yaml test/output/0500/mychart

##@ Tools

KUBERNETES-SPLIT-YAML = $(shell pwd)/bin/kubernetes-split-yaml
.PHONY: kubernetes-split-yaml
kubernetes-split-yaml: ## Download kubernetes-split-yaml locally if necessary.
	$(call go-install-tool,$(KUBERNETES-SPLIT-YAML),github.com/mogensen/kubernetes-split-yaml@v0.4.0)

# go-install-tool will 'go install' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-install-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
$(GO) mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin $(GO) install $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef