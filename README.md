# kustohelmize

## User scenario

### Work with [kustomize](https://kustomize.io/).

Say you have a project created by [Operator SDK](https://sdk.operatorframework.io/) and the `Makefile` should look like this:

```Makefile
.PHONY: deploy
deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMAGE}
	$(KUSTOMIZE) build config/default | kubectl apply -f -
```

`make deploy` will create the YAML file with `kustomize` and deploy it to the cluster. This might be good enough during development, but may not very helpful for end-users.

We can slightly duplicate the target and update it like this:

```Makefile
.PHONY: helm
helm: manifests kustomize kustohelmize
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMAGE}
	$(KUSTOMIZE) build config/default > /home/config/production.yaml
    $(KUSTOHELMIZE) --from=/home/config/production.yaml create mychart
```

### Some Notes

```bash
# Split kustomized YAML file into multiple YAML files
./bin/kubernetes-split-yaml --outdir ./test/testdata/generated ./test/testdata/mt-gpu-operator.yaml

./bin/kustohelmize create --from=test/testdata/mt-gpu-operator.yaml --intermediate-dir=mychart-generated mychart
```