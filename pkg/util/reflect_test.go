package util

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestKeySortPod(t *testing.T) {

	manifest := map[string]interface{}{
		"metadata": map[string]interface{}{
			"name": "test",
		},
		"spec": map[string]interface{}{
			"serviceAccountName": "sa",
		},
		"apiVersion": "v1",
		"kind":       "pod",
	}

	expected := []string{"apiVersion", "kind", "metadata", "spec"}
	actual := make([]string, len(expected))

	for i, k := range SortedMapKeys(reflect.ValueOf(manifest), "") {
		actual[i] = ReflectValue(k).String()
	}

	require.Equal(t, expected, actual)
}

func TestKeySortConfigMap(t *testing.T) {

	manifest := map[string]interface{}{
		"metadata": map[string]interface{}{
			"name": "test",
		},
		"data": map[string]interface{}{
			"key": "value",
		},
		"apiVersion": "v1",
		"kind":       "ConfigMap",
	}

	expected := []string{"apiVersion", "kind", "metadata", "data"}
	actual := make([]string, len(expected))

	for i, k := range SortedMapKeys(reflect.ValueOf(manifest), "") {
		actual[i] = ReflectValue(k).String()
	}

	require.Equal(t, expected, actual)
}

func TestKeySortRole(t *testing.T) {

	manifest := map[string]interface{}{
		"metadata": map[string]interface{}{
			"name": "test",
		},
		"rules": []interface{}{
			map[string]interface{}{
				"apiGroups": []interface{}{
					"apps",
				},
			},
		},
		"apiVersion": "v1",
		"kind":       "Role",
	}

	expected := []string{"apiVersion", "kind", "metadata", "rules"}
	actual := make([]string, len(expected))

	for i, k := range SortedMapKeys(reflect.ValueOf(manifest), "") {
		actual[i] = ReflectValue(k).String()
	}

	require.Equal(t, expected, actual)
}

func TestKeySortContainers(t *testing.T) {

	manifest := map[string]interface{}{
		"terminationMessagePath": "/tmp",
		"image":                  "nginx",
		"name":                   "my-nginx",
		"args": []interface{}{
			"100",
		},
		"command": []interface{}{
			"sleep",
		},
	}

	expected := []string{"name", "image", "command", "args", "terminationMessagePath"}
	actual := make([]string, len(expected))

	for i, k := range SortedMapKeys(reflect.ValueOf(manifest), "spec.template.spec.containers") {
		actual[i] = ReflectValue(k).String()
	}

	require.Equal(t, expected, actual)
}

func TestKeySortInitContainers(t *testing.T) {

	manifest := map[string]interface{}{
		"terminationMessagePath": "/tmp",
		"image":                  "nginx",
		"name":                   "my-nginx",
		"args": []interface{}{
			"100",
		},
		"command": []interface{}{
			"sleep",
		},
	}

	expected := []string{"name", "image", "command", "args", "terminationMessagePath"}
	actual := make([]string, len(expected))

	for i, k := range SortedMapKeys(reflect.ValueOf(manifest), "spec.template.spec.initContainers") {
		actual[i] = ReflectValue(k).String()
	}

	require.Equal(t, expected, actual)
}

func TestKeySortMetadata(t *testing.T) {

	manifest := map[string]interface{}{
		"annotations": map[string]interface{}{
			"key": "value",
		},
		"namespace": "ns",
		"name":      "name",
	}

	expected := []string{"name", "namespace", "annotations"}
	actual := make([]string, len(expected))

	for i, k := range SortedMapKeys(reflect.ValueOf(manifest), "spec.template.metadata") {
		actual[i] = ReflectValue(k).String()
	}

	require.Equal(t, expected, actual)
}
