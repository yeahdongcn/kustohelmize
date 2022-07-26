package util

import "strings"

func IsCustomResourceDefinition(path string) bool {
	return strings.HasSuffix(path, "-crd.yaml")
}
