package config

import "path/filepath"

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
	return XPath(filepath.Join(string(xpath), s))
}

type FileConfig map[XPath]XPathConfig

type GlobalConfig struct {
}

type Config struct {
	Chartname     string                `yaml:"chartname"`
	GlobalConfig  GlobalConfig          `yaml:"globalConfig"`
	FileConfigMap map[string]FileConfig `yaml:"fileConfigMap"`
}
