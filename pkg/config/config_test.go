package config

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func TestKeyExist2(t *testing.T) {
	logger := zap.New()
	config := NewChartConfig(logger, "chart")
	config.SharedValues["replicas"] = 1
	xc := XPathConfig{
		Key: fmt.Sprintf("%s%s%s", sharedValuesPrefix, XPathSeparator, "replicas"),
	}
	key, exist := config.keyExist(xc.Key)
	require.True(t, exist)
	require.Equal(t, key, "replicas")
}

func TestKeyExist1(t *testing.T) {
	logger := zap.New()
	config := NewChartConfig(logger, "chart")
	var empty struct{}
	config.SharedValues["this"] = map[string]interface{}{"is": map[string]interface{}{"a": map[string]interface{}{"nodeSelector": empty}}}
	xc := XPathConfig{
		Strategy: XPathStrategyInline,
		Key:      fmt.Sprintf("%s%s%s", sharedValuesPrefix, XPathSeparator, "this.is.a.nodeSelector"),
	}
	key, exist := config.keyExist(xc.Key)
	require.True(t, exist)
	require.Equal(t, key, "this.is.a.nodeSelector")

	xc = XPathConfig{
		Strategy: XPathStrategyInline,
		Key:      fmt.Sprintf("%s%s%s", sharedValuesPrefix, XPathSeparator, "this.isa.nodeSelector"),
	}
	_, exist = config.keyExist(xc.Key)
	require.False(t, exist)

	xc = XPathConfig{
		Strategy: XPathStrategyInline,
		Key:      fmt.Sprintf("%s%s%s", sharedValuesPrefix, XPathSeparator, "this.is.a.nodeSelectorX"),
	}
	_, exist = config.keyExist(xc.Key)
	require.False(t, exist)
}

func TestGetKeyFromSharedValues(t *testing.T) {
	logger := zap.New()
	config := NewChartConfig(logger, "chart")
	var empty struct{}
	config.SharedValues["this"] = map[string]interface{}{"is": map[string]interface{}{"a": map[string]interface{}{"nodeSelector": empty}}}
	xc := XPathConfig{
		Strategy: XPathStrategyInline,
		Key:      fmt.Sprintf("%s%s%s", sharedValuesPrefix, XPathSeparator, "this.is.a.nodeSelector"),
	}
	key, keyType := config.determineKeyType(xc.Key)
	require.Equal(t, keyType, KeyTypeShared)
	require.Equal(t, key, "this.is.a.nodeSelector")

	xc = XPathConfig{
		Strategy: XPathStrategyInline,
		Key:      fmt.Sprintf("%s%s%s", sharedValuesPrefix, XPathSeparator, "this.is.b.nodeSelector"),
	}
	key, keyType = config.determineKeyType(xc.Key)
	require.Equal(t, keyType, KeyTypeNotFound)
	require.Equal(t, key, "this.is.b.nodeSelector")
}

func TestValues(t *testing.T) {
	logger := zap.New()
	config := NewChartConfig(logger, "chart")
	var empty struct{}
	config.SharedValues["nodeSelector"] = empty
	config.FileConfig["deployment.yaml"] = Config{
		"abc": []XPathConfig{
			{
				Strategy: XPathStrategyInline,
				Key:      "image.repository",
				Value:    "nginx",
			},
			{
				Strategy:     XPathStrategyInline,
				Key:          "image.tag",
				Value:        "latest",
				DefaultValue: ".Chart.AppVersion",
			},
			{
				Strategy: XPathStrategyInline,
				Key:      "image.pullPolicy",
				Value:    "Always",
			},
			{
				Strategy: XPathStrategyInline,
				Key:      "aa.bb.cc.dd",
				Value:    "ff",
			},
			{
				Strategy: XPathStrategyInline,
				Key:      "aa.bb.cc.ee",
				Value:    "ff",
			},
		},
	}
	config.FileConfig["daemonset.yaml"] = Config{
		"abc": []XPathConfig{
			{
				Strategy: XPathStrategyInline,
				Key:      "image.repository",
				Value:    "nginx",
			},
			{
				Strategy:     XPathStrategyInline,
				Key:          "image.tag",
				Value:        "latest",
				DefaultValue: ".Chart.AppVersion",
			},
			{
				Strategy: XPathStrategyInline,
				Key:      "image.pullPolicy",
				Value:    "Always",
			},
			{
				Strategy: XPathStrategyInline,
				Key:      "aa.bb.cc.dd",
				Value:    "ff",
			},
			{
				Strategy: XPathStrategyInline,
				Key:      "aa.bb.cc.ee",
				Value:    "ff",
			},
		},
	}
	values, err := config.Values()
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(values), 1)
}

func TestValidateGlobalConfigCannotHaveRootLevelEntry(t *testing.T) {
	logger := zap.New()
	config := NewChartConfig(logger, "chart")

	config.GlobalConfig = Config{
		"": []XPathConfig{
			{
				Strategy: XPathStrategyControlIf,
				Key:      "dont.care",
				Value:    "dont-care",
			},
		},
	}

	err := config.Validate()
	require.Error(t, err)
}

type errFunc func(require.TestingT, error, ...interface{})

func TestValidateRootFileConfigCanOnlyUseFileIf(t *testing.T) {

	tests := map[XPathStrategy]errFunc{
		XPathStrategyInline:        require.Error,
		XPathStrategyInlineYAML:    require.Error,
		XPathStrategyNewline:       require.Error,
		XPathStrategyNewlineYAML:   require.Error,
		XPathStrategyControlIf:     require.Error,
		XPathStrategyControlIfYAML: require.Error,
		XPathStrategyControlWith:   require.Error,
		XPathStrategyControlRange:  require.Error,
		XPathStrategyAppendWith:    require.Error,
		XPathStrategyFileIf:        require.NoError,
	}

	logger := zap.New()

	for testCase, checkFunc := range tests {
		t.Run(string(testCase), func(t *testing.T) {
			config := NewChartConfig(logger, "chart")

			config.FileConfig["daemonset.yaml"] = Config{
				"abc": []XPathConfig{
					{
						Strategy: XPathStrategyInline,
						Key:      "image.repository",
						Value:    "nginx",
					},
				},
				"": []XPathConfig{
					{
						Strategy: testCase,
						Key:      "dont.care",
						Value:    "dont-care",
					},
				},
			}

			err := config.Validate()
			checkFunc(t, err)
		})
	}
}

func TestValidateNonRootFileConfigCannotUseFileIf(t *testing.T) {

	tests := map[XPathStrategy]errFunc{
		XPathStrategyInline:        require.NoError,
		XPathStrategyInlineYAML:    require.NoError,
		XPathStrategyNewline:       require.NoError,
		XPathStrategyNewlineYAML:   require.NoError,
		XPathStrategyControlIf:     require.NoError,
		XPathStrategyControlIfYAML: require.NoError,
		XPathStrategyControlWith:   require.NoError,
		XPathStrategyControlRange:  require.NoError,
		XPathStrategyAppendWith:    require.NoError,
		XPathStrategyFileIf:        require.Error,
	}

	logger := zap.New()

	for testCase, checkFunc := range tests {
		t.Run(string(testCase), func(t *testing.T) {
			config := NewChartConfig(logger, "chart")

			config.FileConfig["daemonset.yaml"] = Config{
				"abc": []XPathConfig{
					{
						Strategy: testCase,
						Key:      "image.repository",
						Value:    "nginx",
					},
				},
			}

			err := config.Validate()
			checkFunc(t, err)
		})
	}
}

func TestRegexStrategyMustHaveRegexProperty(t *testing.T) {
	logger := zap.New()
	config := NewChartConfig(logger, "chart")

	config.FileConfig["daemonset.yaml"] = Config{
		"abc": []XPathConfig{
			{
				Strategy: XPathStrategyInlineRegex,
				Key:      "spec.template.spec.containers[0].args",
			},
		},
	}

	require.Error(t, config.Validate())
}

func TestRegexStrategyWithoutCaptureGroupFails(t *testing.T) {
	logger := zap.New()
	config := NewChartConfig(logger, "chart")

	config.FileConfig["daemonset.yaml"] = Config{
		"abc": []XPathConfig{
			{
				Strategy: XPathStrategyInlineRegex,
				Key:      "spec.template.spec.containers[0].args",
				Regex:    `--metrics-bind-address=127.0.0.1:8882`,
			},
		},
	}

	require.Error(t, config.Validate())
}

func TestRegexStrategyWithMoreThanOneCaptureGroupFails(t *testing.T) {
	logger := zap.New()
	config := NewChartConfig(logger, "chart")

	config.FileConfig["daemonset.yaml"] = Config{
		"abc": []XPathConfig{
			{
				Strategy: XPathStrategyInlineRegex,
				Key:      "spec.template.spec.containers[0].args",
				Regex:    `--metrics-bind-address=127.0.0.1:(\d)(\d)`,
			},
		},
	}

	require.Error(t, config.Validate())
}

func TestRegexStrategyWithOnlyOneCaptureGroupSucceeds(t *testing.T) {
	logger := zap.New()
	config := NewChartConfig(logger, "chart")

	config.FileConfig["daemonset.yaml"] = Config{
		"abc": []XPathConfig{
			{
				Strategy: XPathStrategyInlineRegex,
				Key:      "spec.template.spec.containers[0].args",
				Regex:    `--metrics-bind-address=127.0.0.1:(\d)`,
			},
		},
	}

	require.NoError(t, config.Validate())
}
