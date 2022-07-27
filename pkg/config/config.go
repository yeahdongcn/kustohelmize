package config

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/yeahdongcn/kustohelmize/pkg/chart"
	"github.com/yeahdongcn/kustohelmize/pkg/util"
	"gopkg.in/yaml.v2"
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

type XPathConfigs []XPathConfig

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

type Config map[XPath]XPathConfigs

func newGlobalConfig(chartname string) *Config {
	return &Config{
		"metadata.name": []XPathConfig{
			{
				Strategy: XPathStrategyInline,
				Key:      fmt.Sprintf(chart.NameFormat, chartname),
			},
		},
		"metadata.labels": []XPathConfig{
			{
				Strategy: XPathStrategyNewline,
				Key:      fmt.Sprintf(chart.CommonLabelsFormat, chartname),
			},
		},
	}
}

type GenericMap map[string]interface{}

type ChartConfig struct {
	Chartname    string            `yaml:"chartname"`
	SharedValues GenericMap        `yaml:"sharedValues"`
	GlobalConfig Config            `yaml:"globalConfig"`
	FileConfig   map[string]Config `yaml:"fileConfig"`
}

func NewChartConfig(chartname string) *ChartConfig {
	config := &ChartConfig{
		Chartname:    chartname,
		SharedValues: GenericMap{},
		GlobalConfig: *newGlobalConfig(chartname),
		FileConfig:   map[string]Config{},
	}
	return config
}

func (c *ChartConfig) Values() (string, error) {
	str := ""
	// 1. SharedValues
	out, err := yaml.Marshal(c.SharedValues)
	if err != nil {
		return str, err
	}
	str += fmt.Sprintf("%s\n", string(out))

	// 2. FileConfig
	root := GenericMap{}
	for filename, fileConfig := range c.FileConfig {
		key := util.LowerCamelFilenameWithoutExt(filepath.Base(filename))
		root[key] = GenericMap{}
		fileRoot := root[key].(GenericMap)

		for _, v := range fileConfig {
			for _, c := range v {
				configRoot := fileRoot
				substrings := strings.Split(c.Key, XPathSeparator)
				for i, substring := range substrings {
					if i == 0 && substring == SharedValues {
						break
					}
					if configRoot[substring] == nil {
						configRoot[substring] = GenericMap{}
					}
					if i < len(substrings)-1 {
						configRoot = configRoot[substring].(GenericMap)
					} else {
						configRoot[substring] = c.Value
					}
				}
			}
		}
	}
	out, err = yaml.Marshal(root)
	if err != nil {
		return str, nil
	}

	str += fmt.Sprintf("%s\n", string(out))
	return str, nil
}

func (c *ChartConfig) GetKeyFromSharedValues(xc *XPathConfig) (string, bool) {
	key := xc.Key
	if strings.HasPrefix(key, SharedValues) {
		substrings := strings.Split(key, XPathSeparator)
		if len(substrings) <= 1 {
			panic("Invalid key")
		}
		key = key[len(SharedValues)+len(XPathSeparator):]
	}
	return key, c.SharedValues[key] != nil
}
