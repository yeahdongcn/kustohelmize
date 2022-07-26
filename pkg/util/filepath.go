package util

import (
	"path/filepath"
	"strings"
)

func IsCustomResourceDefinition(path string) bool {
	return strings.HasSuffix(path, "-crd.yaml")
}

func FilenameWithoutExt(path string) string {
	return strings.TrimSuffix(path, filepath.Ext(path))
}
