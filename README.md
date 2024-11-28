# Kustohelmize
[![Go Report Card](https://goreportcard.com/badge/github.com/yeahdongcn/kustohelmize)](https://goreportcard.com/report/github.com/yeahdongcn/kustohelmize)
[![Contributors](https://img.shields.io/github/contributors/yeahdongcn/kustohelmize)](https://github.com/yeahdongcn/kustohelmize/graphs/contributors)
![GitHub last commit](https://img.shields.io/github/last-commit/yeahdongcn/kustohelmize/main)
![GitHub license](https://img.shields.io/github/license/yeahdongcn/kustohelmize)

Kustohelmize allows you to easily create a Helm Chart from a [kustomized](https://github.com/kubernetes-sigs/kustomize) YAML file.

- [Kustohelmize](#kustohelmize)
  - [CLI](#cli)
    - [kustohelmize](#kustohelmize-1)
    - [kustohelmize create](#kustohelmize-create)
  - [User Scenario](#user-scenario)
    - [Working with kustomize](#working-with-kustomize)
  - [Community](#community)

## CLI

### kustohelmize

```sh
❯ ./kustohelmize
Automate Helm chart creation from any existing Kubernetes manifests

Usage:
  kustohelmize [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  create      Create a chart from a given YAML file
  help        Help about any command
  version     Print the client version information

Flags:
  -h, --help   help for kustohelmize

Use "kustohelmize [command] --help" for more information about a command.
```

### kustohelmize create

```sh
❯ ./kustohelmize create --help
Create a new Helm chart

Usage:
  kustohelmize create NAME [flags]

Flags:
  -a, --app-version string                     The version of the application enclosed inside of this chart
  -d, --description string                     A one-sentence description of the chart
  -f, --from string                            The path to a kustomized YAML file
  -h, --help                                   Help for create
  -k, --kubernetes-split-yaml-command string   Command to split Kubernetes YAML (default "kubernetes-split-yaml")
  -p, --starter string                         The name or absolute path to Helm starter scaffold
  -s, --suppress-namespace                     Suppress creation of namespace resource, which Kustomize will emit. RBAC bindings for SAs will be to {{ .Release.Namespace }}
  -v, --version string                         A SemVer 2 conformant version string of the chart
```

## User Scenario

### Working with [kustomize](https://kustomize.io/)

Suppose you have a project created by [Operator SDK](https://sdk.operatorframework.io/). The `Makefile` should look like this:

```Makefile
.PHONY: deploy
deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
    cd config/manager && $(KUSTOMIZE) edit set image controller=${IMAGE}
    $(KUSTOMIZE) build config/default | kubectl apply -f -
```

Running `make deploy` will create the YAML file with `kustomize` and deploy it into the cluster. This might be sufficient during development but may not be very helpful for end-users.

We can slightly modify the target and update it like this:

```Makefile
.PHONY: helm
helm: manifests kustomize kustohelmize
    cd config/manager && $(KUSTOMIZE) edit set image controller=${IMAGE}
    $(KUSTOMIZE) build config/default --output config/production.yaml
    $(KUSTOHELMIZE) --from=config/production.yaml create mychart
```

This will create a Helm chart with default configurations. The directory structure will look like this:

```
.
├── mychart
├── mychart-generated
└── mychart.config
```

A complete example from scratch can be found in the [examples](https://github.com/yeahdongcn/kustohelmize/tree/main/examples) directory.

You can use this tool in an ad-hoc manner against any YAML file containing multiple resources to generate a Helm chart skeleton simply by pointing `--from` at that file.

## Community

* [Open an issue](https://github.com/yeahdongcn/kustohelmize/issues/new)