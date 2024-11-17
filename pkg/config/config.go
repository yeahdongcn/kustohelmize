package config

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/dlclark/regexp2"
	"github.com/go-logr/logr"
	"github.com/yeahdongcn/kustohelmize/pkg/chart"
	"github.com/yeahdongcn/kustohelmize/pkg/util"
	"gopkg.in/yaml.v2"
)

type KeyType string

const (
	KeyTypeFile     KeyType = "file"
	KeyTypeBuiltIn  KeyType = "builtin"
	KeyTypeShared   KeyType = "shared"
	KeyTypeHelpers  KeyType = "helpers"
	KeyTypeNotFound KeyType = "notfound"
)

func (t KeyType) IsHelpersType() bool {
	return t == KeyTypeHelpers
}

type XPathStrategy string

const (
	XPathStrategyInline        XPathStrategy = "inline"
	XPathStrategyInlineYAML    XPathStrategy = "inline-yaml"
	XPathStrategyNewline       XPathStrategy = "newline"
	XPathStrategyNewlineYAML   XPathStrategy = "newline-yaml"
	XPathStrategyControlIf     XPathStrategy = "control-if"
	XPathStrategyControlIfYAML XPathStrategy = "control-if-yaml"
	XPathStrategyControlWith   XPathStrategy = "control-with"
	XPathStrategyControlRange  XPathStrategy = "control-range"
	XPathStrategyFileIf        XPathStrategy = "file-if"
	XPathStrategyInlineRegex   XPathStrategy = "inline-regex"
)

type XPathConfig struct {
	Strategy      XPathStrategy   `yaml:"strategy"`
	Key           string          `yaml:"key"`
	Value         interface{}     `yaml:"value,omitempty"`
	DefaultValue  interface{}     `yaml:"defaultValue,omitempty"`
	Regex         string          `yaml:"regex,omitempty"`
	RegexCompiled *regexp2.Regexp `yaml:"-"`
}

func (x *XPathConfig) ValueRequiresQuote() bool {
	if x.Value == nil {
		return false
	}

	switch value := x.Value.(type) {
	case bool, int, float64:
		return true
	case string:
		str := util.String(value)
		return str.IsBool() || str.IsNumeric() || str.IsWhiteSpace()
	default:
		return false
	}
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
	if s == "" {
		if sliceIndex == XPathSliceIndexNone {
			return XPath(xpath)
		}
		return XPath(fmt.Sprintf("%s[%d]", xpath, sliceIndex))
	} else {
		if sliceIndex == XPathSliceIndexNone {
			return XPath(fmt.Sprintf("%s.%s", xpath, s))
		}
		return XPath(fmt.Sprintf("%s[%d].%s", xpath, sliceIndex, s))
	}
}

type Config map[XPath]XPathConfigs

func defaultGlobalConfig(chartname string) Config {
	return Config{
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

func defaultSharedValues() GenericMap {
	return GenericMap{
		"resources":          GenericMap{},
		"nodeSelector":       GenericMap{},
		"tolerations":        GenericMap{},
		"affinity":           GenericMap{},
		"podSecurityContext": GenericMap{},
		"securityContext":    GenericMap{},
	}
}

type ChartConfig struct {
	Logger       logr.Logger
	Chartname    string            `yaml:"chartname"`
	SharedValues GenericMap        `yaml:"sharedValues"`
	GlobalConfig Config            `yaml:"globalConfig"`
	FileConfig   map[string]Config `yaml:"fileConfig"`
}

func NewChartConfig(logger logr.Logger, chartname string) *ChartConfig {
	config := &ChartConfig{
		Logger:       logger,
		Chartname:    chartname,
		SharedValues: defaultSharedValues(),
		GlobalConfig: defaultGlobalConfig(chartname),
		FileConfig:   map[string]Config{},
	}
	return config
}

// Sort the XPathConfig keys such that strategies that have the most non-nil values are processed first.
func sortConfigKeys(config Config) []XPath {

	configKeys := make([]XPath, 0, len(config))

	for k := range config {
		configKeys = append(configKeys, k)
	}

	sort.SliceStable(configKeys, func(i, j int) bool {
		valueCount := func(strategies XPathConfigs) int {
			values := 0
			for _, strategy := range strategies {
				if strategy.Value != nil {
					values++
				}
			}
			return values
		}

		return valueCount(config[configKeys[i]]) > valueCount(config[configKeys[j]])
	})

	return configKeys
}

func (cc *ChartConfig) Values() (string, error) {
	str := ""
	var err error
	var out []byte
	// 1. SharedValues
	if len(cc.SharedValues) > 0 {
		out, err := yaml.Marshal(cc.SharedValues)
		if err != nil {
			return str, err
		}
		str += fmt.Sprintf("%s\n", string(out))
	}

	// 2. FileConfig
	root := GenericMap{}
	for filename, fileConfig := range cc.FileConfig {
		key := util.LowerCamelFilenameWithoutExt(filepath.Base(filename))
		root[key] = GenericMap{}
		fileRoot := root[key].(GenericMap)

		// Memoize values seen at various XPaths
		rememberedValues := make(map[string]interface{})

		// Order each fileConfig by whether or not any of its strategies have values
		for _, xpath := range sortConfigKeys(fileConfig) {
			for _, c := range fileConfig[xpath] {
				configRoot := fileRoot
				substrings := strings.Split(c.Key, XPathSeparator)
				if _, ok := rememberedValues[c.Key]; !ok {
					// Init rememberdValues for this key
					rememberedValues[c.Key] = nil
				}
				for i, substring := range substrings {
					// XXX: For shared values and global defined values, we should not extend values.yaml
					if i == 0 && (substring == sharedValuesPrefix || substring == cc.Chartname || substring == "") {
						break
					}
					if i == 1 && substring == "Chart" {
						break
					}
					if configRoot[substring] == nil {
						configRoot[substring] = GenericMap{}
					}
					if i < len(substrings)-1 {
						configRoot = configRoot[substring].(GenericMap)
					} else {
						previousValue, ok := rememberedValues[c.Key]
						if configRoot[substring] != nil && ok && previousValue != nil {
							c.Value = previousValue
						}
						if c.Value == nil {
							cc.Logger.Info(fmt.Sprintf("%s: %s", c.Key, "nil"))
							delete(configRoot, substring)
						} else {
							rememberedValues[c.Key] = c.Value
							switch v := c.Value.(type) {
							case int:
								cc.Logger.V(10).Info("type int", "key", c.Key, "value", v)
								configRoot[substring] = v
							case string:
								cc.Logger.V(10).Info("type string", "key", c.Key, "value", v)
								configRoot[substring] = v
							case map[interface{}]interface{}:
								cc.Logger.V(10).Info("type map[interface{}]interface{}", "key", c.Key)
								if len(v) == 0 {
									delete(configRoot, substring)
								} else {
									configRoot[substring] = v
								}
							case []interface{}:
								cc.Logger.V(10).Info("type []interface{}", "key", c.Key)
								if len(v) == 0 {
									delete(configRoot, substring)
								} else {
									configRoot[substring] = v
								}
							default:
								cc.Logger.V(10).Info("type default", "key", c.Key)
								configRoot[substring] = v
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
	} else if keyType == KeyTypeNotFound {
		if xc.Strategy == XPathStrategyControlIf || xc.Strategy == XPathStrategyControlIfYAML {
			key = fmt.Sprintf(".Values.%s", key)
		} else {
			panic(fmt.Sprintf("%s not found", xc.Key))
		}
	}
	if xc.DefaultValue != nil {
		key = fmt.Sprintf("%s | default %s", key, xc.DefaultValue)
	}
	return key, keyType
}

func (c *ChartConfig) Validate() error {
	// Validates
	// - file-if can only be present at root level file configs
	// - globalConfig cannot contain a root level entry
	// - inline-regex must have regex property, and the regex must compile and contain exactly one capture group
	if _, ok := c.GlobalConfig[XPathRoot]; ok {
		return fmt.Errorf("cannot have root level config in GlobalConfig")
	}

	for manifest, config := range c.FileConfig {
		for xpath, xpathConfigs := range config {
			for i, xpathConfig := range xpathConfigs {
				strategy := xpathConfig.Strategy
				if (xpath == XPathRoot) != (strategy == XPathStrategyFileIf) {
					return fmt.Errorf("'%s' cannot use strategy '%s' at '%s'", manifest, strategy, xpath)
				}
				if strategy == XPathStrategyInlineRegex {
					if xpathConfig.Regex == "" {
						return fmt.Errorf("'%s' strategy '%s' must have 'regex' property", manifest, strategy)
					}
					rx := regexp2.MustCompile(xpathConfig.Regex, regexp2.Compiled)
					if len(rx.GetGroupNumbers()) != 2 {
						// groups[0] is the entire match. groups[1] is the bit within ()
						return fmt.Errorf("'%s' strategy '%s': regular expression '%s' must have exactly one replacement group", manifest, strategy, xpathConfig.Regex)
					}
					xpathConfigs[i].RegexCompiled = rx
				}
			}
		}
	}

	return nil
}

func (c *ChartConfig) keyExist(key string) (string, bool) {
	substrings := strings.Split(key, XPathSeparator)
	if len(substrings) <= 1 {
		return key, false
	}
	key = key[len(sharedValuesPrefix)+len(XPathSeparator):]
	current := c.SharedValues
	for index, substring := range substrings {
		if substring == sharedValuesPrefix {
			continue
		}
		out, err := yaml.Marshal(current[substring])
		if err != nil {
			return key, false
		}
		next := GenericMap{}
		err = yaml.Unmarshal(out, &next)
		if err != nil {
			if index == len(substrings)-1 {
				continue
			}
			return key, false
		}
		if next == nil {
			return key, false
		} else {
			current = next
		}
	}
	return key, true
}

func (c *ChartConfig) getKey(xc *XPathConfig) (string, KeyType) {
	key := xc.Key
	if strings.HasPrefix(key, sharedValuesPrefix) {
		key, exist := c.keyExist(key)
		if !exist {
			return key, KeyTypeNotFound
		}
		return key, KeyTypeShared
	} else if strings.HasPrefix(key, c.Chartname) {
		return key, KeyTypeHelpers
	} else if strings.HasPrefix(key, builtInValuesPrefix) {
		return key, KeyTypeBuiltIn
	}
	return key, KeyTypeFile
}
