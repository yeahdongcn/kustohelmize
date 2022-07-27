package util

import (
	"path/filepath"
	"strings"

	"github.com/iancoleman/strcase"
)

func IsCustomResourceDefinition(path string) bool {
	return strings.HasSuffix(path, "-crd.yaml")
}

func LowerCamelFilenameWithoutExt(path string) string {
	return strcase.ToLowerCamel(strings.TrimSuffix(path, filepath.Ext(path)))
}
