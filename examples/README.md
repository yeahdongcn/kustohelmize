# Examples

- [Examples](#examples)
  - [Update `memcached-operator` to Work With Kustohelmize](#update-memcached-operator-to-work-with-kustohelmize)
  - [Configuration File](#configuration-file)
    - [Sections](#sections)
    - [Strategies](#strategies)

## Update `memcached-operator` to Work With [Kustohelmize](https://github.com/yeahdongcn/kustohelmize)

The `memcached-operator` can be easily created by following the [quickstart guide](https://sdk.operatorframework.io/docs/building-operators/golang/quickstart/).

First, create a project directory and initialize the project:

```bash
mkdir memcached-operator
cd memcached-operator
operator-sdk init --domain example.com --repo github.com/example/memcached-operator
```

Next, create a simple Memcached API:

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
  GOBIN=$(LOCALBIN) go install github.com/mogensen/kubernetes-split-yaml@v0.4.0

.PHONY: kustohelmize
kustohelmize: $(KUSTOHELMIZE) ## Download kustohelmize locally if necessary.
$(KUSTOHELMIZE): $(LOCALBIN) kubernetes-split-yaml
  GOBIN=$(LOCALBIN) go install github.com/yeahdongcn/kustohelmize@v0.4.0
```

Run `make helm` to create the Helm Chart. Then update `memcached-operator/deployments/memcached-operator.config` to add your own configuration. For example:

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
      defaultValue: .Chart.AppVersion
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

## Configuration File

### Sections

The configuration file consists of four sections:

1. `chartname`

    The name of the Helm Chart.

1. `sharedValues`

    User-defined values that will be shared within the Helm Chart. These values should not belong to a single template.

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

    The above configuration defines a namespace `memcached-operator` and resource limits and requests.

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

    You can then refer to any of them in the configuration file.

    ```yaml
    key: sharedValues.namespace
    ```

1. `globalConfig`

    With `globalConfig`, you can apply values from `_helpers.tpl` to all templates.

    For example:

    `namespace` and `labels` are usually shared within all templates. You can define them in `globalConfig` and then use them in all templates.

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

1. `fileConfig`

    This is a per-file configuration. You can set values for a specific template with various configurations.

    For example:

    If you want to configure the image of the controller manager with `repository` and `tag` in `values.yaml` from a Helm Chart, you can use the following configuration:

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

### Strategies

We also introduce the `strategy` in the configuration file.

1. `inline`

    ```yaml
    namespace: {{ .Values.namespace }}
    ```

1. `inline-yaml`

    ```yaml
    namespace: {{ toYaml .Values.namespace }}
    ```

1. `newline`

    ```yaml
    - name:
      {{ .Values.memcachedOperatorControllerManagerDeployment.manager.name | nindent 12 }}
    ```

1. `newline-yaml`

    ```yaml
    - name:
      {{ toYaml .Values.memcachedOperatorControllerManagerDeployment.manager.name | nindent 12 }}
    ```

1. `control-with`

    ```
    {{- with .Values.resources }}
    resources:
      {{- toYaml . | nindent 12 }}
    {{- end }}
    ```

1. `control-if`

    ```yaml
    {{- if .Values.operator.initContainer.imagePullPolicy }}
    imagePullPolicy: {{ .Values.operator.initContainer.imagePullPolicy }}
    {{- end }}
    ```

    The `control-if` strategy also supports specifying a condition:

    ```yaml
    spec.replicas:
    - strategy: control-if
      condition: autoscaling.enable
      key: replicas
      value: 1
    ```

    This generates the following Helm template:

    ```yaml
    {{- if .Values.nginxDeploymentDeployment.autoscaling.enable }}
    replicas: {{ .Values.nginxDeploymentDeployment.replicas }}
    {{- end }}
    ```

    You can use `conditionValue` to set a default value for the condition. If not specified, the condition defaults to false and is stored in `values.yaml`.</br>
    `condition` can be used without a `key` and with `!` to negate the condition.

    ```yaml
    spec.replicas:
    - strategy: control-if
      condition: "!autoscaling.enable"
    ```

    This generates the following Helm template:

    ```yaml
    {{- if not .Values.nginxDeploymentDeployment.autoscaling.enable }}
    replicas: 2
    {{- end }}
    ```

1. `control-if-yaml`

    ```
    {{- if .Values.operator.initContainer.imagePullSecrets }}
    imagePullSecrets: {{ toYaml .Values.operator.initContainer.imagePullSecrets | nindent 8 }}
    {{- end }}
    ```

    The above conditional control is also available in the `control-if-yaml` strategy.

    ```yaml
    spec.template.spec.containers[0].ports:
    - strategy: control-if-yaml
      condition: expose.enable
      key: ports
      value:
      - name: http
        containerPort: 80
        protocol: TCP
    ```

    This generates the following Helm template:

    ```yaml
    {{- if .Values.nginxDeploymentDeployment.expose.enable }}
    ports: {{ toYaml .Values.nginxDeploymentDeployment.ports | nindent 12 }}
    {{- end }}
    ```

1. `control-range`

    ```
    imagePullSecrets:
    {{- range .Values.operator.initContainer.imagePullSecrets }}
      - name: {{ . }}
    {{- end }}
    ```

1. `file-if`

    Conditionally includes or omits an entire resource manifest.

    ```
    {{- if .Values.prometheus.enabled }}
    # Entire ServiceMonitor manifest
    {{- end }}
    ```

    First, update `sharedValues` in the chart configuration file to add the switch

    ```yaml
    sharedValues:
      prometheus:
        enabled: true
    ```

    `file-if` must be provided as a root-level configuration (empty XPath) in a `fileConfig` like this. It is an error to use it anywhere else.

    ```yaml
    path/to/my-operator-servicemonitor.yaml:
      "":
      - strategy: file-if
        key: sharedValues.promethues.enabled
    ```

1. `inline-regex`

    Allows insertion of a templated value as part of an overall string, such as the value for a pod's command line argument.

    Consider the following snippet from the [memcached deployment](../examples/memcached-operator/deployments/memcached-operator/templates/memcached-operator-controller-manager-deployment.yaml) in the examples folder:

    ```yaml
    - args:
        - --health-probe-bind-address=:8081
        - --metrics-bind-address=127.0.0.1:8080
        - --leader-elect
      livenessProbe:
        httpGet:
          path: /healthz
          port: 8081
        initialDelaySeconds: 15
        periodSeconds: 20
      readinessProbe:
        httpGet:
          path: /readyz
          port: 8081
        initialDelaySeconds: 5
        periodSeconds: 10
    ```

    You want to template the probe port. It therefore needs to be templated in the `args` section and also at `port` for each of `readinessProbe` and `livenessProbe`.

    This is how to do it.

    1. Edit the kustohelmize configuration file.
    1. Set up the deployment's configuration like this:

        ```yaml
        fileConfig:
          deployments/memcached-operator-generated/memcached-operator-controller-manager-deployment.yaml:
            spec.template.spec.containers[1].args:
            - strategy: inline-regex
              key: manager.probe.port
              regex: --health-probe-bind-address=:(\d+)
              value: 9010
            spec.template.spec.containers[1].readinessProbe.httpGet.port:
            - strategy: inline
              key: manager.probe.port
            spec.template.spec.containers[1].livenessProbe.httpGet.port:
            - strategy: inline
              key: manager.probe.port
        ```

    1. The emitted Helm template will be:

      ```yaml
          - args:
              - --health-probe-bind-address=:{{ .Values.memcachedOperatorControllerManagerDeployment.manager.probe.port }}
              - --metrics-bind-address=127.0.0.1:8080
              - --leader-elect
            livenessProbe:
              httpGet:
                path: /healthz
                port: {{ .Values.memcachedOperatorControllerManagerDeployment.manager.probe.port }}
              initialDelaySeconds: 15
              periodSeconds: 20
            readinessProbe:
              httpGet:
                path: /readyz
                port: {{ .Values.memcachedOperatorControllerManagerDeployment.manager.probe.port }}
              initialDelaySeconds: 5
              periodSeconds: 10
      ```

    The match group in the regex `(\d+)` is templated with the `.Values` identified by `key`. Currently, only one replacement per list item is possible.