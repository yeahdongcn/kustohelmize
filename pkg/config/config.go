package config

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/yeahdongcn/kustohelmize/pkg/chart"
	"github.com/yeahdongcn/kustohelmize/pkg/util"
	"gopkg.in/yaml.v2"
)

type KeyType string

const (
	KeyTypeFile     KeyType = "file"
	KeyTypeShared   KeyType = "shared"
	KeyTypeGlobal   KeyType = "global"
	KeyTypeNotFound KeyType = "notfound"
)

func (t KeyType) IsGlobalType() bool {
	return t == KeyTypeGlobal
}

type XPathStrategy string

const (
	XPathStrategyInline       XPathStrategy = "inline"
	XPathStrategyInlineYAML   XPathStrategy = "inline-yaml"
	XPathStrategyNewline      XPathStrategy = "newline"
	XPathStrategyNewlineYAML  XPathStrategy = "newline-yaml"
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
				Key:      fmt.Sprintf(chart.FullNameFormat, chartname),
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
		Chartname: chartname,
		SharedValues: GenericMap{
			"kustohelmize": "https://github.com/yeahdongcn/kustohelmize/",
		},
		GlobalConfig: *newGlobalConfig(chartname),
		FileConfig:   map[string]Config{},
	}
	return config
}

func (cc *ChartConfig) Values() (string, error) {
	str := ""
	// 1. SharedValues
	out, err := yaml.Marshal(cc.SharedValues)
	if err != nil {
		return str, err
	}
	str += fmt.Sprintf("%s\n", string(out))

	// 2. FileConfig
	root := GenericMap{}
	for filename, fileConfig := range cc.FileConfig {
		key := util.LowerCamelFilenameWithoutExt(filepath.Base(filename))
		root[key] = GenericMap{}
		fileRoot := root[key].(GenericMap)

		for _, v := range fileConfig {
			for _, c := range v {
				configRoot := fileRoot
				substrings := strings.Split(c.Key, XPathSeparator)
				for i, substring := range substrings {
					// XXX: For shared values and global defined values, we should not extend values.yaml
					if i == 0 && (substring == SharedValues || substring == cc.Chartname) {
						break
					}
					if configRoot[substring] == nil {
						configRoot[substring] = GenericMap{}
					}
					if i < len(substrings)-1 {
						configRoot = configRoot[substring].(GenericMap)
					} else {
						if len(c.Value) == 0 {
							// XXX: Handle annotations: {}
							configRoot[substring] = GenericMap{}
						} else {
							// XXX: Handle replicas: 1
							n, err := strconv.Atoi(c.Value)
							if err == nil {
								configRoot[substring] = n
							} else {
								configRoot[substring] = c.Value
							}
						}
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

func (c *ChartConfig) GetFormattedKeyWithDefaultValue(xc *XPathConfig, prefix string) (string, KeyType) {
	key, keyType := c.getKey(xc)
	if keyType == KeyTypeFile {
		key = fmt.Sprintf(".Values.%s.%s", prefix, key)
	} else if keyType == KeyTypeShared {
		key = fmt.Sprintf(".Values.%s", key)
	}
	if xc.DefaultValue != "" {
		key = fmt.Sprintf("%s | default %s", key, xc.DefaultValue)
	}
	return key, keyType
}

func (c *ChartConfig) getKey(xc *XPathConfig) (string, KeyType) {
	key := xc.Key
	if strings.HasPrefix(key, SharedValues) {
		substrings := strings.Split(key, XPathSeparator)
		if len(substrings) <= 1 {
			return key, KeyTypeNotFound
		}
		key = key[len(SharedValues)+len(XPathSeparator):]
		if c.SharedValues[key] == nil {
			return key, KeyTypeNotFound
		}
		return key, KeyTypeShared
	} else if strings.HasPrefix(key, c.Chartname) {
		return key, KeyTypeGlobal
	}
	return key, KeyTypeFile
}
