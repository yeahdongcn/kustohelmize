GO ?= go

include $(CURDIR)/versions.mk

.PHONY: all
all: build

##@ General

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Build

.PHONY: build
build:
	$(GO) build -o bin/kustohelmize $(CURDIR)/main.go

##@ Test

.PHONY: test
test: build
	./bin/kustohelmize create --from=test/testdata/mt-gpu-operator.yaml mychart
# https://github.com/github/super-linter/issues/1601
#	@for f in $(shell ls -d mychart-generated/*.yaml); do kubeval $${f} --ignore-missing-schemas; done
	helm lint ./mychart

KUBERNETES-SPLIT-YAML = $(shell pwd)/bin/kubernetes-split-yaml
.PHONY: kubernetes-split-yaml
kubernetes-split-yaml: ## Download kubernetes-split-yaml locally if necessary.
	$(call go-install-tool,$(KUBERNETES-SPLIT-YAML),github.com/yeahdongcn/kubernetes-split-yaml@v0.4.0)

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