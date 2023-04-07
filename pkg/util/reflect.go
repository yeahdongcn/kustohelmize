package util

import (
	"reflect"
	"regexp"
	"strings"
)

// Match xPath of containers/initContainers
var (
	containerPathRegex = regexp.MustCompile(`\.spec\.(init)?[cC]ontainers$`)
	manifestFirst      = map[string]string{
		"apiVersion": "1",
		"kind":       "2",
		"metadata":   "3",
	}

	// Order these keys first at top of each container
	containerFirst = map[string]string{
		"name":    "1",
		"image":   "2",
		"command": "3",
		"agrs":    "4",
	}

	metadataFirst = map[string]string{
		"name":      "1",
		"namespace": "2",
	}
)

func ReflectValue(v reflect.Value) reflect.Value {
	for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		v = v.Elem()
	}
	return v
}

// Take the reflected map keys which have no guaranteed order and sort them
// Apply special ordering for certain keys at root, metadata or container level,
// and sort remaning keys naturally.
func SortedMapKeys(m reflect.Value, root string) []reflect.Value {

	if ReflectValue(m).Kind() != reflect.Map {
		panic("Expected map")
	}

	// Defermine ordering from current XPath

	order := manifestFirst

	if containerPathRegex.MatchString(root) {
		order = containerFirst
	} else if strings.HasSuffix(root, "metadata") {
		order = metadataFirst
	}

	keyValues := m.MapKeys()

	// Rubbish bubble sort - but there's never more than 20 odd keys to sort.
	for i := 0; i < len(keyValues); i++ {
		for j := 0; j < len(keyValues)-i-1; j++ {
			j0 := ReflectValue(keyValues[j]).String()
			j1 := ReflectValue(keyValues[j+1]).String()

			if v, ok := order[j0]; ok {
				j0 = v
			}

			if v, ok := order[j1]; ok {
				j1 = v
			}

			if j0 > j1 {
				keyValues[j], keyValues[j+1] = keyValues[j+1], keyValues[j]
			}
		}
	}

	return keyValues
}
