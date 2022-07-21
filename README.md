# kustohelmize

```
# Split kustomized YAML file into multiple YAML files
./bin/kubernetes-split-yaml --outdir ./test/testdata/generated ./test/testdata/mt-gpu-operator.yaml

./bin/kustohelmize create --outdir ./test/testdata/generated --yaml ./test/testdata/mt-gpu-operator.yaml xyz
```