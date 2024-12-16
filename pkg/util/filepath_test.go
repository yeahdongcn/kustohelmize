package util

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPaths(t *testing.T) {
	path := "/a/b/c/d/xxx-crd.yaml"
	require.True(t, IsCustomResourceDefinition(path))

	path = "/a/b/c/d/xxx-namespace.yaml"
	require.True(t, IsNamespaceDefinition(path))

	path = "/a/b/c/d/xxx.yaml"
	require.False(t, IsCustomResourceDefinition(path))
	require.False(t, IsNamespaceDefinition(path))

	path = "/a/b/c/d/xxx.yaml"
	require.Equal(t, "xxx", LowerCamelFilenameWithoutExt(path))
	path = "yyy.yaml"
	require.Equal(t, "yyy", LowerCamelFilenameWithoutExt(path))
}
