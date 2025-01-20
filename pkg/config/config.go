package config

import (
	"fmt"
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
	XPathStrategyAppendWith    XPathStrategy = "append-with"
)

type Condition struct {
	Key   string `yaml:"key,omitempty"`
	Value bool   `yaml:"value,omitempty"`
}

type XPathConfig struct {
	Strategy          XPathStrategy   `yaml:"strategy"`
	Key               string          `yaml:"key"`
	Value             interface{}     `yaml:"value,omitempty"`
	DefaultValue      interface{}     `yaml:"defaultValue,omitempty"`
	Regex             string          `yaml:"regex,omitempty"`
	RegexCompiled     *regexp2.Regexp `yaml:"-"`
	Conditions        []Condition     `yaml:"conditions,omitempty"`
	ConditionOperator *string         `yaml:"conditionOperator,omitempty"`
	// Deprecated
	Condition      string `yaml:"condition,omitempty"`
	ConditionValue bool   `yaml:"conditionValue,omitempty"`
}

func (xc *XPathConfig) ValueRequiresQuote() bool {
	if xc.Value == nil {
		return false
	}

	switch value := xc.Value.(type) {
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

func (xpath XPath) NewElement(sliceIndex int) XPath {
	if sliceIndex == XPathSliceIndexNone {
		return xpath
	}
	return XPath(fmt.Sprintf("%s[%d]", xpath, sliceIndex))
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

type kvPair struct {
	Key   string
	Value interface{}
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
		key := util.LowerCamelFilenameWithoutExt(filename)
		root[key] = GenericMap{}
		fileRoot := root[key].(GenericMap)

		// Memoize values seen at various XPaths
		rememberedValues := make(map[string]interface{})

		// Order each fileConfig by whether or not any of its strategies have values
		for _, xpath := range sortConfigKeys(fileConfig) {
			for _, c := range fileConfig[xpath] {
				configRoot := fileRoot

				kvs := []kvPair{{c.Key, c.Value}}
				if c.Condition != "" {
					conditionKey := strings.TrimPrefix(c.Condition, "!")
					kvs = append(kvs, kvPair{conditionKey, c.ConditionValue})
				}
				for _, condition := range c.Conditions {
					conditionKey := strings.TrimPrefix(condition.Key, "!")
					kvs = append(kvs, kvPair{conditionKey, condition.Value})
				}
				for _, kv := range kvs {
					substrings := strings.Split(kv.Key, XPathSeparator)
					if _, ok := rememberedValues[kv.Key]; !ok {
						// Init rememberedValues for this key
						rememberedValues[kv.Key] = nil
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
							previousValue, ok := rememberedValues[kv.Key]
							if configRoot[substring] != nil && ok && previousValue != nil {
								kv.Value = previousValue
							}
							if kv.Value == nil {
								cc.Logger.Info(fmt.Sprintf("%s: %s", kv.Key, "nil"))
								delete(configRoot, substring)
							} else {
								rememberedValues[kv.Key] = kv.Value
								switch v := kv.Value.(type) {
								case int:
									cc.Logger.V(10).Info("type int", "key", kv.Key, "value", v)
									configRoot[substring] = v
								case string:
									cc.Logger.V(10).Info("type string", "key", kv.Key, "value", v)
									configRoot[substring] = v
								case map[interface{}]interface{}:
									cc.Logger.V(10).Info("type map[interface{}]interface{}", "key", kv.Key)
									if len(v) == 0 {
										delete(configRoot, substring)
									} else {
										configRoot[substring] = v
									}
								case []interface{}:
									cc.Logger.V(10).Info("type []interface{}", "key", kv.Key)
									if len(v) == 0 {
										delete(configRoot, substring)
									} else {
										configRoot[substring] = v
									}
								default:
									cc.Logger.V(10).Info("type default", "key", kv.Key)
									configRoot[substring] = v
								}
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

func (c *ChartConfig) formatKey(key, prefix string, keyType KeyType, strategy XPathStrategy) string {
	switch keyType {
	case KeyTypeFile:
		return fmt.Sprintf(".Values.%s.%s", prefix, key)
	case KeyTypeShared, KeyTypeNotFound:
		formattedKey := fmt.Sprintf(".Values.%s", key)
		if keyType == KeyTypeNotFound && strategy != XPathStrategyControlIf && strategy != XPathStrategyControlIfYAML {
			panic(fmt.Sprintf("%s not found", key))
		}
		return formattedKey
	default:
		return key
	}
}

func (c *ChartConfig) GetFormattedCondition(xc *XPathConfig, prefix string) (string, bool) {
	formatCondition := func(condition string) (string, bool) {
		not := strings.HasPrefix(condition, "!")
		conditionKey := strings.TrimPrefix(condition, "!")
		key, keyType := c.determineKeyType(conditionKey)
		if key == "" {
			return "", not
		}
		return c.formatKey(key, prefix, keyType, xc.Strategy), not
	}

	formatMultipleConditions := func(conditions []Condition) (string, bool) {
		keys := make([]string, len(conditions))
		for i, condition := range conditions {
			key, not := formatCondition(condition.Key)
			if key != "" {
				if not {
					key = fmt.Sprintf("(not %s)", key)
				}
				keys[i] = key
			}
		}

		return fmt.Sprintf("%s %s", *xc.ConditionOperator, strings.Join(keys, " ")), false
	}

	if xc.Condition != "" {
		return formatCondition(xc.Condition)
	} else if len(xc.Conditions) > 0 {
		if len(xc.Conditions) == 1 {
			return formatCondition(xc.Conditions[0].Key)
		}
		return formatMultipleConditions(xc.Conditions)
	}
	return "", false
}

func (c *ChartConfig) GetFormattedKeyWithDefaultValue(xc *XPathConfig, prefix string) (string, KeyType) {
	key, keyType := c.determineKeyType(xc.Key)
	if key == "" {
		return key, keyType
	}
	key = c.formatKey(key, prefix, keyType, xc.Strategy)
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
	// - control-if and control-if-yaml with multiple conditions:
	//   - must have conditionOperator property
	//   - conditionOperator must be 'and' or 'or'
	// - other strategies cannot have condition or conditions property
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
				} else if strategy == XPathStrategyControlIf || strategy == XPathStrategyControlIfYAML {
					if len(xpathConfig.Conditions) > 1 {
						if xpathConfig.ConditionOperator == nil {
							return fmt.Errorf("'%s' strategy '%s' must have 'conditionOperator' property", manifest, strategy)
						}
					}
					if xpathConfig.ConditionOperator != nil {
						if *xpathConfig.ConditionOperator != "and" && *xpathConfig.ConditionOperator != "or" {
							return fmt.Errorf("'%s' strategy '%s' conditionOperator must be 'and' or 'or'", manifest, strategy)
						}
					}
				} else {
					if xpathConfig.Condition != "" || len(xpathConfig.Conditions) > 0 {
						return fmt.Errorf("'%s' strategy '%s' cannot have 'condition' or 'conditions' property", manifest, strategy)
					}
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

func (c *ChartConfig) determineKeyType(key string) (string, KeyType) {
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
