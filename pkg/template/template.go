package template

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"

	"github.com/dlclark/regexp2"
	"github.com/go-logr/logr"
	"github.com/yeahdongcn/kustohelmize/internal/third_party/dep/fs"
	"github.com/yeahdongcn/kustohelmize/pkg/chart"
	"github.com/yeahdongcn/kustohelmize/pkg/config"
	"github.com/yeahdongcn/kustohelmize/pkg/util"
	"gopkg.in/yaml.v2"
)

type context struct {
	out              io.Writer
	prefix           string
	fileConfig       config.Config
	setRoleNamespace bool
}

type Processor struct {
	logger            logr.Logger
	config            *config.ChartConfig
	templatesDir      string
	crdsDir           string
	suppressNamespace bool

	context context
}

var roleSubjectRegex = regexp.MustCompile(`subjects\[\d+\].kind`)

func NewProcessor() *Processor {
	return &Processor{}
}

func (p *Processor) WithLogger(logger logr.Logger) *Processor {
	p.logger = logger
	return p
}

func (p *Processor) WithChartConfig(config *config.ChartConfig) *Processor {
	p.config = config
	return p

}

func (p *Processor) WithTemplatesDir(templatesDir string) *Processor {
	p.templatesDir = templatesDir
	return p
}

func (p *Processor) WithCrdsDir(crdsDir string) *Processor {
	p.crdsDir = crdsDir
	return p
}

func (p *Processor) WithSuppressNamespace(suppress bool) *Processor {
	p.suppressNamespace = suppress
	return p
}

func (p *Processor) Process() error {
	for source, fileConfig := range p.config.FileConfig {
		filename := filepath.Base(source)

		if p.suppressNamespace && util.IsNamespaceDefinition(source) {
			// Don't emit namespaces
			continue
		}

		if util.IsCustomResourceDefinition(source) {
			if err := os.MkdirAll(p.crdsDir, 0755); err != nil {
				p.logger.Error(err, "Failed to create CRD directory")
				return err
			}
			dest := filepath.Join(p.crdsDir, filename)
			if err := fs.CopyFile(source, dest); err != nil {
				p.logger.Error(err, "Error copying file", "source", source, "dest", dest)
				return err
			}
			continue
		}

		dest := filepath.Join(p.templatesDir, filename)
		file, err := os.Create(dest)
		if err != nil {
			p.logger.Error(err, "Error creating dest file", "dest", dest)
			return err
		}
		defer file.Close()

		_, err = file.WriteString(chart.Header)
		if err != nil {
			p.logger.Error(err, "Error writing dest file header", "dest", dest)
		}

		bs, err := os.ReadFile(source)
		if err != nil {
			p.logger.Error(err, "Error reading source YAML", "source", source)
			return err
		}
		data := config.GenericMap{}
		err = yaml.Unmarshal(bs, &data)
		if err != nil {
			p.logger.Error(err, "Error unmarshalling source YAML", "source", source)
			return err
		}

		p.context = context{
			out:              file,
			prefix:           util.LowerCamelFilenameWithoutExt(source),
			fileConfig:       fileConfig,
			setRoleNamespace: false,
		}
		d := reflect.ValueOf(data)
		p.walk(d, 0, config.XPathRoot, config.XPathSliceIndexNone)
	}

	return nil
}

func indent(s string, n int) string {
	return indentsFromSlice(s, n, false)
}

func indentsFromSlice(s string, n int, fromSlice bool) string {
	indents := strings.Repeat(defaultIndent, n)

	indented := ""
	scanner := bufio.NewScanner(strings.NewReader(s))
	for scanner.Scan() {
		if fromSlice && indented == "" {
			indented += scanner.Text() + "\n"
		} else {
			indented += indents + scanner.Text() + "\n"
		}
	}
	indented = strings.Trim(indented, "\n")

	return indented
}

// Emit a scalar slice value
func (p *Processor) printSliceScalar(str string, nindent int) {
	if str == "" {
		fmt.Fprintln(p.context.out, indent(fmt.Sprintf("- \"%s\"", str), nindent))
	} else if str == "*" {
		fmt.Fprintln(p.context.out, indent(fmt.Sprintf("- '%s'", str), nindent))
	} else {
		fmt.Fprintln(p.context.out, indent(fmt.Sprintf("- %s", str), nindent))
	}

}

// Perform regex substitution. Die if the regex system errors
func mustReplace(rx *regexp2.Regexp, str string, replacement string) string {
	replaced, err := rx.ReplaceFunc(str, func(m regexp2.Match) string {
		// No additional checking here as existence of only one capture group already asserted.
		groups := m.Groups()
		str0 := groups[0].String()    // The whole matched string
		cap1 := groups[1].Captures[0] // The capture group

		// Replace 'replacement' into str0 at the indices indicated by cap1
		prefix := str0[0:cap1.Index]
		suffix := str0[cap1.Index+cap1.Length:]

		return prefix + replacement + suffix
	}, -1, -1)

	if err != nil {
		panic(err)
	}

	return replaced
}

// Process a slice of scalars
func (p *Processor) processSlice(v reflect.Value, xpath config.XPath, nindent int) {
	xpathConfigs := p.context.fileConfig[xpath]
	for i := 0; i < v.Len(); i++ {
		item := util.ReflectValue(v.Index(i))
		str := item.String()
		if len(xpathConfigs) == 0 {
			p.printSliceScalar(str, nindent)
		} else {
			for _, xpathConfig := range xpathConfigs {
				if xpathConfig.Strategy != config.XPathStrategyInlineRegex {
					continue
				}
				rx := xpathConfig.RegexCompiled
				if matched, _ := rx.MatchString(str); !matched {
					continue
				}
				// Match against xpathConfig.RegexCompiled and do replacement.
				var value string
				key, keyType := p.config.GetFormattedKeyWithDefaultValue(&xpathConfig, p.context.prefix)
				if keyType.IsHelpersType() {
					// name: {{ include "mychart.fullname" . }}
					value = fmt.Sprintf(singleIncludeFormat, key)
				} else {
					// imagePullPolicy: {{ .Values.image.pullPolicy }}
					value = fmt.Sprintf(singleValueFormat, key)
				}
				// Replace now
				str = mustReplace(rx, str, value)
				break
			}
			p.printSliceScalar(str, nindent)
		}
	}
}

func (p *Processor) processMapOrDie(k reflect.Value, v reflect.Value, nindent int,
	xpath config.XPath, xpathConfigs config.XPathConfigs, hasSliceIndex bool) bool {
	if len(xpathConfigs) == 0 {
		return false
	}
	xpathConfig := xpathConfigs[0]
	switch xpathConfig.Strategy {
	case config.XPathStrategyInline, config.XPathStrategyInlineYAML:
		key := fmt.Sprintf(singleLineKeyFormat, k)
		fmt.Fprint(p.context.out, indentsFromSlice(key, nindent, hasSliceIndex))

		var value string
		key, keyType := p.config.GetFormattedKeyWithDefaultValue(&xpathConfig, p.context.prefix)
		if keyType.IsHelpersType() {
			// name: {{ include "mychart.fullname" . }}
			value = fmt.Sprintf(singleIncludeFormat, key)
		} else if len(xpathConfigs) > 1 {
			// image: "{{ .Values.image.repository }}:{{ .Values.image.tag | default .Chart.AppVersion }}"
			for _, xpc := range xpathConfigs {
				key, _ := p.config.GetFormattedKeyWithDefaultValue(&xpc, p.context.prefix)
				value += fmt.Sprintf(singleValueFormat, key)
				value += config.MultiValueSeparator
			}
			value = fmt.Sprintf("\"%s\"", strings.TrimRight(value, config.MultiValueSeparator))
		} else if xpathConfig.Strategy == config.XPathStrategyInline {
			// imagePullPolicy: {{ .Values.image.pullPolicy }}
			value = fmt.Sprintf(singleValueFormat, key)
			if strings.Contains(string(xpath), ".env") && xpathConfig.ValueRequiresQuote() {
				// env:
				// - name: ENABLE_FEATURE_GATE
				//   value: "{{ .Values.deployment.nginx.env.ENABLE_FEATURE_GATE }}"
				value = fmt.Sprintf("\"%s\"", value)
			}
		} else {
			// imagePullPolicy: {{ toYaml .Values.image.pullPolicy }}
			value = fmt.Sprintf(singleValueFormat, key)
		}
		fmt.Fprintln(p.context.out, value)
		return true
	case config.XPathStrategyNewline, config.XPathStrategyNewlineYAML:
		key := fmt.Sprintf(newlineKeyFormat, k)
		fmt.Fprintln(p.context.out, indentsFromSlice(key, nindent, hasSliceIndex))

		var value string
		key, keyType := p.config.GetFormattedKeyWithDefaultValue(&xpathConfig, p.context.prefix)
		if keyType.IsHelpersType() {
			// selector:
			//   {{- include "mychart.selectorLabels" . | nindent 4 }}
			value = fmt.Sprintf(newlineIncludeFormat, key, (nindent+1)*2)
		} else if xpathConfig.Strategy == config.XPathStrategyNewline {
			// imagePullPolicy:
			//   {{- .Values.image.pullPolicy | nindent 12 }}
			value = fmt.Sprintf(newlineValueFormat, key, (nindent+1)*2)
		} else {
			// selector:
			//   {{- toYaml .Values.resources | nindent 12 }}
			value = fmt.Sprintf(newlineYAMLValueFormat, key, (nindent+1)*2)
		}
		fmt.Fprintln(p.context.out, indent(value, nindent+1))
		return true
	case config.XPathStrategyControlWith:
		key, _ := p.config.GetFormattedKeyWithDefaultValue(&xpathConfig, p.context.prefix)
		// {{- with .Values.tolerations }}
		// tolerations:
		//   {{- toYaml . | nindent 8 }}
		// {{- end }}
		value := fmt.Sprintf(withFormat, key, k, (nindent+1)*2)
		fmt.Fprintln(p.context.out, indentsFromSlice(value, nindent, hasSliceIndex))
		return true
	case config.XPathStrategyControlIf, config.XPathStrategyControlIfYAML:
		key, _ := p.config.GetFormattedKeyWithDefaultValue(&xpathConfig, p.context.prefix)
		condition, isNegative := p.config.GetFormattedCondition(&xpathConfig, p.context.prefix)
		if condition == "" {
			// If no condition is specified, fall back to the key
			condition = key
			isNegative = false
		}

		var value string
		if key == "" && condition != "" {
			vStr := util.ToStringOrDie(v)
			format := ifOriginFormat
			if isNegative {
				format = ifNotOriginFormat
			}
			value = fmt.Sprintf(format, condition, k, vStr)
		} else {
			if xpathConfig.Strategy == config.XPathStrategyControlIf {
				format := ifFormat
				if isNegative {
					format = ifNotFormat
				}
				value = fmt.Sprintf(format, condition, k, key)
			} else { // XPathStrategyControlIfYAML
				format := ifYAMLFormat
				if isNegative {
					format = ifNotYAMLFormat
				}
				value = fmt.Sprintf(format, condition, k, key, (nindent+1)*2)
			}
		}
		fmt.Fprintln(p.context.out, indentsFromSlice(value, nindent, hasSliceIndex))
		return true
	case config.XPathStrategyControlRange:
		key, _ := p.config.GetFormattedKeyWithDefaultValue(&xpathConfig, p.context.prefix)
		// {{- range .Values.imagePullSecrets }}
		//   - name: {{ . }}
		// {{- end }}
		value := fmt.Sprintf(rangeFormat, k, key)
		fmt.Fprintln(p.context.out, indentsFromSlice(value, nindent, hasSliceIndex))
		return true
	case config.XPathStrategyFileIf:
		key, _ := p.config.GetFormattedKeyWithDefaultValue(&xpathConfig, p.context.prefix)
		fmt.Fprintf(p.context.out, fileIfFormat, key)
		return true
	case config.XPathStrategyInlineRegex:
		// Processed by slice
		return false
	case config.XPathStrategyAppendWith:
		vStr := util.ToStringOrDie(v)
		value := fmt.Sprintf("%s:%s", k, vStr)
		fmt.Fprintln(p.context.out, indentsFromSlice(value, nindent, hasSliceIndex))

		key, _ := p.config.GetFormattedKeyWithDefaultValue(&xpathConfig, p.context.prefix)
		value = fmt.Sprintf(appendWithFormat, key, nindent*2)
		fmt.Fprintln(p.context.out, indentsFromSlice(value, nindent, hasSliceIndex))
		return true
	default:
		panic(fmt.Sprintf("Unknown XPath strategy: %s", xpathConfig.Strategy))
	}
}

func (p *Processor) processMap(k reflect.Value, v reflect.Value, nindent int, xpath config.XPath, hasSliceIndex *bool) bool {
	// XXX: The priority of file config is greater than global config.
	if p.processMapOrDie(k, v, nindent, xpath, p.context.fileConfig[xpath], *hasSliceIndex) {
		p.logger.V(10).Info("Processed map for file config", "xpath", xpath)
		// XXX: For the first element only.
		if *hasSliceIndex {
			*hasSliceIndex = false
		}
		return true
	}
	if p.processMapOrDie(k, v, nindent, xpath, p.config.GlobalConfig[xpath], *hasSliceIndex) {
		p.logger.V(10).Info("Processed map for global config", "xpath", xpath)
		// XXX: For the first element only.
		if *hasSliceIndex {
			*hasSliceIndex = false
		}
		return true
	}

	return false
}

func (p *Processor) walk(v reflect.Value, nindent int, root config.XPath, sliceIndex int) {

	if root.IsRoot() {
		// Process root level map for existence of file-if
		hasSliceIndex := false
		if p.processMap(reflect.ValueOf(""), reflect.ValueOf(""), 0, root, &hasSliceIndex) {
			defer fmt.Fprintln(p.context.out, endDelimited)
		}
	}

	v = util.ReflectValue(v)
	switch v.Kind() {
	case reflect.Array, reflect.Slice:
		p.logger.V(10).Info("Processing slice", "root", root)

		keepWalking := false
		if v.Len() > 0 {
			first := util.ReflectValue(v.Index(0))
			if first.Kind() == reflect.Map {
				// XXX: If this is a slice of maps, we need to process them separately.
				keepWalking = true
			}
		}
		if !root.IsRoot() && !keepWalking {
			fmt.Fprintln(p.context.out)
		}
		if !keepWalking {
			p.processSlice(v, root, nindent)
		} else {
			for i := 0; i < v.Len(); i++ {
				isConditional := false

				xpathConfigs := p.context.fileConfig[root.NewElement(i)]
				if len(xpathConfigs) > 0 {
					xpathConfig := xpathConfigs[0]
					switch xpathConfig.Strategy {
					// We only support control-if for slice elements
					case config.XPathStrategyControlIf:
						condition, isNegative := p.config.GetFormattedCondition(&xpathConfig, p.context.prefix)
						if condition != "" {
							isConditional = true
							var value string
							if isNegative {
								value = fmt.Sprintf(fileIfFormat, condition)
							} else {
								value = fmt.Sprintf(fileIfNotFormat, condition)
							}
							if i == 0 {
								value = fmt.Sprintf("\n%s", value)
							}
							fmt.Fprintln(p.context.out, indent(value, nindent))
						}
					}
				}

				if i == 0 && !isConditional {
					fmt.Fprint(p.context.out, indent(slicePrefixFirstLineFormat, nindent))
				} else {
					fmt.Fprint(p.context.out, indent(slicePrefixOtherLineFormat, nindent))
				}
				p.walk(v.Index(i), nindent+1, root, i)

				if isConditional {
					fmt.Fprintln(p.context.out, indent(endDelimited, nindent))
				}
			}
		}
	case reflect.Map:
		p.logger.V(10).Info("Processing map", "root", root)

		kvs := util.SortedMapKeys(v, string(root))
		if !root.IsRoot() && sliceIndex == config.XPathSliceIndexNone {
			if len(kvs) > 0 {
				fmt.Fprintln(p.context.out)
			} else {
				// Handle empty map
				fmt.Fprintf(p.context.out, "{}\n")
			}
		}
		hasSliceIndex := sliceIndex != config.XPathSliceIndexNone
		for _, kv := range kvs {
			mapKey := util.ReflectValue(kv).String()
			xpath := root.NewChild(mapKey, sliceIndex)
			if !p.processMap(kv, v.MapIndex(kv), nindent, xpath, &hasSliceIndex) {
				key := fmt.Sprintf(singleLineKeyFormat, kv)
				if hasSliceIndex {
					fmt.Fprint(p.context.out, indentsFromSlice(key, nindent, true))
					// XXX: For the first element only.
					hasSliceIndex = false
				} else {
					fmt.Fprint(p.context.out, indent(key, nindent))
				}
				p.walk(v.MapIndex(kv), nindent+1, xpath, config.XPathSliceIndexNone)
			}
		}
	default:
		if p.suppressNamespace && (strings.HasSuffix(string(root), "metadata.namespace") || (p.context.setRoleNamespace && strings.HasSuffix(string(root), "namespace"))) {
			// Use helm's idea of what the namespace is
			fmt.Fprintf(p.context.out, singleValueFormat, ".Release.Namespace")
			fmt.Fprintln(p.context.out)
			p.context.setRoleNamespace = false
			return
		}
		// spec.template.spec.nodeSelector: Invalid type. Expected: [string,null], given: boolean
		if v.Kind() == reflect.Invalid {
			fmt.Fprintln(p.context.out, "null")
			return
		}
		s := v.String()
		if p.suppressNamespace && roleSubjectRegex.MatchString(string(root)) && s == "ServiceAccount" {
			// This is a bit mucky.
			// Here we are inside a subjects block of a role/customrole binding.
			// It is for a service account.
			// Order of keys is guaranteed.
			// Therefore we know that the next 'namespace' key encountered will be for this subject.
			p.context.setRoleNamespace = true
		}
		p.logger.V(10).Info("Processing others", "root", root, "s", s)
		str := util.String(s)
		if str.IsBool() || str.IsNumeric() || str.IsWhiteSpace() {
			fmt.Fprintf(p.context.out, "\"%s\"\n", v)
		} else if str.HasNewLine() {
			fmt.Fprintf(p.context.out, "|\n%s\n", indent(s, nindent+1))
		} else {
			fmt.Fprintln(p.context.out, v)
		}
	}
}
