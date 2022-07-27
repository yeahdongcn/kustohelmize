package config

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetKeyFromSharedValues(t *testing.T) {
	config := NewChartConfig("chart")
	var empty struct{}
	config.SharedValues["this.is.a.nodeSelector"] = empty
	xc := XPathConfig{
		Strategy: XPathStrategyInline,
		Key:      fmt.Sprintf("%s%s%s", SharedValues, XPathSeparator, "this.is.a.nodeSelector"),
	}
	key, found := config.GetKeyFromSharedValues(&xc)
	require.True(t, found)
	require.Equal(t, key, "this.is.a.nodeSelector")

	xc = XPathConfig{
		Strategy: XPathStrategyInline,
		Key:      fmt.Sprintf("%s%s%s", SharedValues, XPathSeparator, "this.is.b.nodeSelector"),
	}
	key, found = config.GetKeyFromSharedValues(&xc)
	require.False(t, found)
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
