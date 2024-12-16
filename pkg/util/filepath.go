package util

import (
	"path/filepath"
	"strings"

	"github.com/iancoleman/strcase"
)

// IsCustomResourceDefinition checks if the file is a Custom Resource Definition (CRD) based on its suffix.
func IsCustomResourceDefinition(path string) bool {
	return strings.HasSuffix(path, "-crd.yaml")
}

// IsNamespaceDefinition checks if the file is a Namespace Definition based on its suffix.
func IsNamespaceDefinition(path string) bool {
	return strings.HasSuffix(path, "-namespace.yaml")
}

// LowerCamelFilenameWithoutExt converts the filename (without extension) to lower camel case.
func LowerCamelFilenameWithoutExt(path string) string {
	filenameWithoutExt := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	return strcase.ToLowerCamel(filenameWithoutExt)
}
