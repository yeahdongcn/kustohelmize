package template

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFormat(t *testing.T) {
	expected := []string{
		"{{ include \"%s\" . }}",
		"{{ %s }}",

		"{{- include \"%s\" . | nindent %d }}",
		"{{- %s | nindent %d }}",

		`{{- with %s }}
%s:
  {{- toYaml . | nindent %d }}
{{- end }}`,
	}

	actual := []string{
		singleIncludeFormat,
		singleValueFormat,

		newlineIncludeFormat,
		newlineValueFormat,

		withFormat,
	}

	require.Equal(t, expected, actual)
}
