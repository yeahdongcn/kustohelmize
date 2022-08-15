package template

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/go-logr/logr"
	"github.com/yeahdongcn/kustohelmize/internal/third_party/dep/fs"
	"github.com/yeahdongcn/kustohelmize/pkg/chart"
	"github.com/yeahdongcn/kustohelmize/pkg/config"
	"github.com/yeahdongcn/kustohelmize/pkg/util"
	"gopkg.in/yaml.v1"
)

type context struct {
	out        io.Writer
	prefix     string
	fileConfig config.Config
}

type Processor struct {
	logger  logr.Logger
	config  *config.ChartConfig
	destDir string

	context context
}

func NewProcessor(logger logr.Logger, config *config.ChartConfig, destDir string) *Processor {
	return &Processor{
		logger:  logger,
		destDir: destDir,
		config:  config,
	}
}

func (p *Processor) Process() error {
	for source, fileConfig := range p.config.FileConfig {
		filename := filepath.Base(source)
		dest := filepath.Join(p.destDir, filename)
		if util.IsCustomResourceDefinition(filename) {
			err := fs.CopyFile(source, dest)
			if err != nil {
				p.logger.Error(err, "Error copying file", "source", source, "dest", dest)
				return err
			}
			continue
		}

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

		bs, err := ioutil.ReadFile(source)
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
			out:        file,
			prefix:     util.LowerCamelFilenameWithoutExt(filename),
			fileConfig: fileConfig,
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

func (p *Processor) processSlice(v reflect.Value, nindent int) {
	for i := 0; i < v.Len(); i++ {
		item := util.ReflectValue(v.Index(i))
		str := item.String()
		if str == "" {
			fmt.Fprintln(p.context.out, indent(fmt.Sprintf("- \"%s\"", item), nindent))
		} else if str == "*" {
			fmt.Fprintln(p.context.out, indent(fmt.Sprintf("- '%s'", item), nindent))
		} else {
			fmt.Fprintln(p.context.out, indent(fmt.Sprintf("- %s", item), nindent))
		}
	}
}

func (p *Processor) processMapOrDie(v reflect.Value, nindent int, xpathConfigs config.XPathConfigs, hasSliceIndex bool) bool {
	if len(xpathConfigs) == 0 {
		return false
	}
	xpathConfig := xpathConfigs[0]
	switch xpathConfig.Strategy {
	case config.XPathStrategyInline:
		fallthrough
	case config.XPathStrategyInlineYAML:
		key := fmt.Sprintf(singleLineKeyFormat, v)
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
		} else {
			// imagePullPolicy: {{ toYaml .Values.image.pullPolicy }}
			value = fmt.Sprintf(singleValueFormat, key)
		}
		fmt.Fprintln(p.context.out, value)
		return true
	case config.XPathStrategyNewline:
		fallthrough
	case config.XPathStrategyNewlineYAML:
		key := fmt.Sprintf(newlineKeyFormat, v)
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
		value := fmt.Sprintf(withFormat, key, v, (nindent+1)*2)
		fmt.Fprintln(p.context.out, indentsFromSlice(value, nindent, hasSliceIndex))
		return true
	case config.XPathStrategyControlIf:
		fallthrough
	case config.XPathStrategyControlIfYAML:
		key, _ := p.config.GetFormattedKeyWithDefaultValue(&xpathConfig, p.context.prefix)

		var value string
		if xpathConfig.Strategy == config.XPathStrategyControlIf {
			value = fmt.Sprintf(ifFormat, key, v, key, (nindent+1)*2)
		} else {
			value = fmt.Sprintf(ifYAMLFormat, key, v, key, (nindent+1)*2)
		}
		fmt.Fprintln(p.context.out, indentsFromSlice(value, nindent, hasSliceIndex))
		return true
	case config.XPathStrategyControlRange:
		p.logger.Info("ControlRange not implemented")
	default:
		panic(fmt.Sprintf("Unknown XPath strategy: %s", xpathConfig.Strategy))
	}

	return false
}

func (p *Processor) processMap(v reflect.Value, nindent int, xpath config.XPath, hasSliceIndex *bool) bool {
	// XXX: The priority of file config is greater than global config.
	if p.processMapOrDie(v, nindent, p.context.fileConfig[xpath], *hasSliceIndex) {
		p.logger.V(10).Info("Processed map for file config")
		// XXX: For the first element only.
		if *hasSliceIndex {
			*hasSliceIndex = false
		}
		return true
	}
	if p.processMapOrDie(v, nindent, p.config.GlobalConfig[xpath], *hasSliceIndex) {
		p.logger.V(10).Info("Processed map for global config")
		// XXX: For the first element only.
		if *hasSliceIndex {
			*hasSliceIndex = false
		}
		return true
	}

	return false
}

func (p *Processor) walk(v reflect.Value, nindent int, root config.XPath, sliceIndex int) {
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
			p.processSlice(v, nindent)
		} else {
			for i := 0; i < v.Len(); i++ {
				if i == 0 {
					fmt.Fprint(p.context.out, indent(slicePrefixFirstLineFormat, nindent))
				} else {
					fmt.Fprint(p.context.out, indent(slicePrefixOtherLineFormat, nindent))
				}
				p.walk(v.Index(i), nindent+1, root, i)
			}
		}
	case reflect.Map:
		p.logger.V(10).Info("Processing map", "root", root)

		if !root.IsRoot() && sliceIndex == config.XPathSliceIndexNone {
			fmt.Fprintln(p.context.out)
		}
		hasSliceIndex := sliceIndex != config.XPathSliceIndexNone
		for _, k := range v.MapKeys() {
			mapKey := util.ReflectValue(k).String()
			xpath := root.NewChild(mapKey, sliceIndex)
			if !p.processMap(k, nindent, xpath, &hasSliceIndex) {
				key := fmt.Sprintf(singleLineKeyFormat, k)
				if hasSliceIndex {
					fmt.Fprint(p.context.out, indentsFromSlice(key, nindent, true))
					// XXX: For the first element only.
					hasSliceIndex = false
				} else {
					fmt.Fprint(p.context.out, indent(key, nindent))
				}
				p.walk(v.MapIndex(k), nindent+1, xpath, config.XPathSliceIndexNone)
			}
		}
	default:
		// spec.template.spec.nodeSelector: Invalid type. Expected: [string,null], given: boolean
		if v.Kind() == reflect.Invalid {
			fmt.Fprintln(p.context.out, "null")
			return
		}
		s := v.String()
		p.logger.V(10).Info("Processing others", "root", root, "s", s)
		if s == "true" || s == "false" {
			fmt.Fprintf(p.context.out, "\"%s\"\n", v)
		} else if strings.Contains(s, "\n") {
			fmt.Fprintf(p.context.out, "|\n%s\n", indent(s, nindent+1))
		} else {
			fmt.Fprintln(p.context.out, v)
		}
	}
}
