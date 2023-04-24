package template

import (
	"fmt"
	"testing"

	"github.com/dlclark/regexp2"
	"github.com/stretchr/testify/require"
)

type regexTest struct {
	expression  string
	input       string
	replacement string
	expected    string
}

func TestRegexReplace(t *testing.T) {

	testCases := []regexTest{
		{
			expression:  `--metrics-bind-address=127.0.0.1:(\d+)`,
			input:       "--metrics-bind-address=127.0.0.1:1234",
			replacement: "8081",
			expected:    "--metrics-bind-address=127.0.0.1:8081",
		},
		{
			expression:  `--metrics-bind-address=127.0.0.1:(\d+)-1234`,
			input:       "--metrics-bind-address=127.0.0.1:1234-1234",
			replacement: "{{ .Values.myoperator.manager.metrics.port }}",
			expected:    "--metrics-bind-address=127.0.0.1:{{ .Values.myoperator.manager.metrics.port }}-1234",
		},
	}

	for _, testCase := range testCases {
		t.Run(fmt.Sprintf("%s <- %s", testCase.expression, testCase.replacement), func(t *testing.T) {
			rx := regexp2.MustCompile(testCase.expression, regexp2.Multiline)
			require.Equal(t, testCase.expected, mustReplace(rx, testCase.input, testCase.replacement))
		})
	}
}
