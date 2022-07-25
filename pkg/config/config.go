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
	Value    string        `yaml:"value"`
}

type XPath string

func (xpath XPath) IsRoot() bool {
	return xpath == XPathRoot
}

func (xpath XPath) NewChild(s string) XPath {
	if xpath.IsRoot() {
		return XPath(s)
	}
	return XPath(fmt.Sprintf("%s.%s", xpath, s))
}

type FileConfig map[XPath]XPathConfig

type GlobalConfig FileConfig

func NewGlobalConfig(chartname string) *GlobalConfig {
	return &GlobalConfig{
		"metadata.name": {
			Strategy: XPathStrategyInline,
			Value:    fmt.Sprintf(chart.NameFormat, chartname),
		},
		"metadata.labels": {
			Strategy: XPathStrategyNewline,
			Value:    fmt.Sprintf(chart.CommonLabelsFormat, chartname),
		},
	}
}

type Config struct {
	Chartname     string                `yaml:"chartname"`
	GlobalConfig  GlobalConfig          `yaml:"globalConfig"`
	FileConfigMap map[string]FileConfig `yaml:"fileConfigMap"`
}
