package config

import (
	"fmt"

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
	Strategy XPathStrategy `yaml:"strategy"`
	Key      string        `yaml:"key"`
	Value    string        `yaml:"value,omitempty"`
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

func NewGlobalConfig(chartname string) *Config {
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
	Chartname     string            `yaml:"chartname"`
	GlobalConfig  Config            `yaml:"globalConfig"`
	PerFileConfig map[string]Config `yaml:"perFileConfig"`
}
