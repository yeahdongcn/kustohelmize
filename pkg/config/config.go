package config

import (
	"fmt"
	"strings"

	"github.com/yeahdongcn/kustohelmize/pkg/chart"
)

type XPathStrategy string

const (
	XPathStrategyInline       XPathStrategy = "inline"
	XPathStrategyNewline      XPathStrategy = "newline"
	XPathStrategyControlIf    XPathStrategy = "control-if"
	XPathStrategyControlWith  XPathStrategy = "control-with"
	XPathStrategyControlRange XPathStrategy = "control-range"
)

type XPathConfig struct {
	Strategy     XPathStrategy `yaml:"strategy"`
	Key          string        `yaml:"key"`
	Value        string        `yaml:"value,omitempty"`
	DefaultValue string        `yaml:"defaultValue,omitempty"`
}

type XPath string

func (xpath XPath) IsRoot() bool {
	return xpath == XPathRoot
}

func (xpath XPath) NewChild(s string, sliceIndex int) XPath {
	if xpath.IsRoot() {
		return XPath(s)
	}
	if sliceIndex == XPathSliceIndexNone {
		return XPath(fmt.Sprintf("%s.%s", xpath, s))
	}
	return XPath(fmt.Sprintf("%s[%d].%s", xpath, sliceIndex, s))
}

type Config map[XPath]XPathConfig

func newGlobalConfig(chartname string) *Config {
	return &Config{
		"metadata.name": {
			Strategy: XPathStrategyInline,
			Key:      fmt.Sprintf(chart.NameFormat, chartname),
		},
		"metadata.labels": {
			Strategy: XPathStrategyNewline,
			Key:      fmt.Sprintf(chart.CommonLabelsFormat, chartname),
		},
	}
}

type ChartConfig struct {
	Chartname    string                 `yaml:"chartname"`
	SharedValues map[string]interface{} `yaml:"sharedValues"`
	GlobalConfig Config                 `yaml:"globalConfig"`
	FileConfig   map[string]Config      `yaml:"fileConfig"`
}

func NewChartConfig(chartname string) *ChartConfig {
	config := &ChartConfig{
		Chartname:    chartname,
		SharedValues: map[string]interface{}{},
		GlobalConfig: *newGlobalConfig(chartname),
		FileConfig:   map[string]Config{},
	}
	return config
}

func (c *ChartConfig) GetKeyFromSharedValues(xc *XPathConfig) (string, bool) {
	key := xc.Key
	if strings.HasPrefix(key, "sharedValues") {
		substrings := strings.Split(key, ".")
		if len(substrings) <= 1 {
			panic("Invalid key")
		}
		key = substrings[1]
	}
	return key, c.SharedValues[key] != nil
}
