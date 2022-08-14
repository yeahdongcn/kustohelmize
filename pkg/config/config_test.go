package config

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestKeyExist2(t *testing.T) {
	config := NewChartConfig("chart")
	config.SharedValues["replicas"] = 1
	xc := XPathConfig{
		Strategy: XPathStrategyInline,
		Key:      fmt.Sprintf("%s%s%s", sharedValuesPrefix, XPathSeparator, "replicas"),
	}
	key, exist := config.keyExist(xc.Key)
	require.True(t, exist)
	require.Equal(t, key, "replicas")
}

func TestKeyExist1(t *testing.T) {
	config := NewChartConfig("chart")
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
	config := NewChartConfig("chart")
	var empty struct{}
	config.SharedValues["this"] = map[string]interface{}{"is": map[string]interface{}{"a": map[string]interface{}{"nodeSelector": empty}}}
	xc := XPathConfig{
		Strategy: XPathStrategyInline,
		Key:      fmt.Sprintf("%s%s%s", sharedValuesPrefix, XPathSeparator, "this.is.a.nodeSelector"),
	}
	key, keyType := config.getKey(&xc)
	require.Equal(t, keyType, KeyTypeShared)
	require.Equal(t, key, "this.is.a.nodeSelector")

	xc = XPathConfig{
		Strategy: XPathStrategyInline,
		Key:      fmt.Sprintf("%s%s%s", sharedValuesPrefix, XPathSeparator, "this.is.b.nodeSelector"),
	}
	key, keyType = config.getKey(&xc)
	require.Equal(t, keyType, KeyTypeNotFound)
	require.Equal(t, key, "this.is.b.nodeSelector")
}

func TestValues(t *testing.T) {
	config := NewChartConfig("chart")
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
