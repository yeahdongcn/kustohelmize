# Examples

## Extend [memcached-operator](https://sdk.operatorframework.io/docs/building-operators/golang/quickstart/)

`memcached-operator` could be easily created by following the quickstart guide.

Create a project directory for your project and initialize the project:

```bash
mkdir memcached-operator
cd memcached-operator
operator-sdk init --domain example.com --repo github.com/example/memcached-operator
```

Create a simple Memcached API:

```bash
operator-sdk create api --group cache --version v1alpha1 --kind Memcached --resource --controller
```

Update the Makefile to use `KustoHelmize`:

```Makefile
.PHONY: deploy-dryrun
deploy-dryrun: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default --output config/memcached-operator.yaml

.PHONY: helm
helm: deploy-dryrun kustohelmize
	$(KUSTOHELMIZE) create --from=config/memcached-operator.yaml deployments/memcached-operator
	helm lint deployments/memcached-operator

KUBERNETES-SPLIT-YAML ?= $(LOCALBIN)/kubernetes-split-yaml
KUSTOHELMIZE ?= $(LOCALBIN)/kustohelmize

.PHONY: kubernetes-split-yaml
kubernetes-split-yaml: $(KUBERNETES-SPLIT-YAML) ## Download kubernetes-split-yaml locally if necessary.
$(KUBERNETES-SPLIT-YAML): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install github.com/yeahdongcn/kubernetes-split-yaml@v0.4.0

.PHONY: kustohelmize
kustohelmize: $(KUSTOHELMIZE) ## Download kustohelmize locally if necessary.
$(KUSTOHELMIZE): $(LOCALBIN) kubernetes-split-yaml
	GOBIN=$(LOCALBIN) go install github.com/yeahdongcn/kustohelmize@latest
```

Run `make helm` to create the Helm Chart. Then update `memcached-operator/deployments/memcached-operator.config` to add your own config. For example:

```yaml
chartname: memcached-operator
sharedValues:
  namespace: memcached-operator
globalConfig:
  metadata.labels:
  - strategy: newline
    key: memcached-operator.labels
  metadata.namespace:
  - strategy: inline
    key: sharedValues.namespace
fileConfig:
  deployments/memcached-operator-generated/memcached-operator-controller-manager-deployment.yaml:
    spec.replicas:
    - strategy: inline
      key: replicas
      value: 1
```

To be continued...