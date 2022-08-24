# Examples

## Update `memcached-operator` to Work With [Kustohelmize](https://github.com/yeahdongcn/kustohelmize)

`memcached-operator` could be easily created by following the [quickstart guide](https://sdk.operatorframework.io/docs/building-operators/golang/quickstart/).

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
  resources:
    limits:
      cpu: 500m
      memory: 128Mi
    requests:
      cpu: 5m
      memory: 64Mi
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
    spec.template.spec.containers[1].image:
    - strategy: inline
      key: manager.image.repository
      value: controller
    - strategy: inline
      key: manager.image.tag
      value: latest
    spec.template.spec.containers[1].name:
    - strategy: newline
      key: manager.name
      value: manager
    spec.template.spec.containers[1].resources:
    - strategy: control-with
      key: sharedValues.resources
  deployments/memcached-operator-generated/memcached-operator-system-namespace.yaml:
    metadata.name:
    - strategy: inline
      key: sharedValues.namespace
```

There are 4 sections in the config file:

1. `chartname`

    The name of the Helm Chart.

2. `sharedValues`

    User defined values that will be shared within the Helm Chart. These values should not belong to a single template.

    For example:

    ```yaml
    sharedValues:
      namespace: memcached-operator
      resources:
        limits:
          cpu: 500m
          memory: 128Mi
        requests:
          cpu: 5m
          memory: 64Mi
    ```

    The above config defines a namespace `memcached-operator` and limits && requests for the resources.

    This will automatically update the `values.yaml`.

    ```yaml
    namespace: memcached-operator
    resources:
      limits:
        cpu: 500m
        memory: 128Mi
      requests:
        cpu: 5m
        memory: 64Mi
    ```

    Then you can refer anyone of them in the config file.

    ```yaml
    key: sharedValues.namespace
    ```

3. `globalConfig`

    With `globalConfig`, one can apply the values from `_helpers.tpl` to all templates.

    For example:

    `namespace` and `labels` are usually share within all templates. You can define them in `globalConfig` and then use them in all templates.

    ```yaml
    metadata.labels:
    - strategy: newline
      key: memcached-operator.labels
    metadata.namespace:
    - strategy: inline
      key: sharedValues.namespace
    ```

    This will automatically update all templates.

    ```yaml
    metadata:
      namespace: {{ .Values.namespace }}
      labels:
        {{- include "memcached-operator.labels" . | nindent 4 }}
    ```

4. `fileConfig`

    This is per file config. One can set the values for a specific template with various of configurations.

    For example:

    If you want to configure the image of the controller manager with `repository` and `tag` in `values.yaml` from a Helm Chart, you can use the following config:

    ```yaml
    spec.template.spec.containers[1].image:
    - strategy: inline
      key: manager.image.repository
      value: controller
    - strategy: inline
      key: manager.image.tag
      value: latest
    ```

    This will automatically update the `values.yaml` and `templates/memcached-operator-controller-manager-deployment.yaml`.

    ```yaml
    memcachedOperatorControllerManagerDeployment:
      manager:
        image:
          repository: controller
          tag: latest
    ```

    ```yaml
    image: "{{ .Values.memcachedOperatorControllerManagerDeployment.manager.image.repository }}:{{ .Values.memcachedOperatorControllerManagerDeployment.manager.image.tag }}"
    ```

We also introduce the `strategy` in the config file.

1. `inline`

    ```yaml
    namespace: {{ .Values.namespace }}
    ```

2. `inline-yaml`

    ```yaml
    namespace: {{ toYaml .Values.namespace }}
    ```

3. `newline`

    ```yaml
    - name:
      {{ .Values.memcachedOperatorControllerManagerDeployment.manager.name | nindent 12 }}
    ```

4. `newline-yaml`

    ```yaml
    - name:
      {{ toYaml .Values.memcachedOperatorControllerManagerDeployment.manager.name | nindent 12 }}
    ```

5. `control-with`

    ```
    {{- with .Values.resources }}
    resources:
      {{- toYaml . | nindent 12 }}
    {{- end }}
    ```

6. `control-if`

    ```
    {{- if .Values.operator.initContainer.imagePullPolicy }}
    imagePullPolicy: {{ .Values.operator.initContainer.imagePullPolicy }}
    {{- end }}
    ```

6. `control-if-yaml`

    ```
    {{- if .Values.operator.initContainer.imagePullSecrets }}
    imagePullSecrets: {{ toYaml .Values.operator.initContainer.imagePullSecrets | nindent 8 }}
    {{- end }}
    ```

7. `control-range`

    ```
    imagePullSecrets:
    {{- range .Values.operator.initContainer.imagePullSecrets }}
      - name: {{ . }}
    {{- end }}
    ```