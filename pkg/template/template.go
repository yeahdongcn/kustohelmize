package template

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/go-logr/logr"
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
	logger logr.Logger
	// out               io.Writer
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
			err := exec.Command("cp", "-f", source, dest).Run()
			if err != nil {
				p.logger.Error(err, "Failed to copy file", "source", source, "dest", dest)
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

func (p *Processor) processMapOrDie(v reflect.Value, nindent int, xpathConfigs config.XPathConfigs, isGlobalConfig bool, hasSliceIndex bool) bool {
	if len(xpathConfigs) == 0 {
		return false
	}
	xpathConfig := xpathConfigs[0]
	switch xpathConfig.Strategy {
	case config.XPathStrategyInline:
		key := fmt.Sprintf(singleLineKeyFormat, v)
		fmt.Fprint(p.context.out, indentsFromSlice(key, nindent, hasSliceIndex))

		var value string
		key, shared := p.config.GetKeyFromSharedValues(&xpathConfig)
		if isGlobalConfig {
			// name: {{ include "mychart.fullname" . }}
			value = fmt.Sprintf(globalSingleLineValueFormat, key)
		} else {
			if shared {
				value = fmt.Sprintf(sharedSingleLineValueFormat, key)
			} else {
				for _, xpc := range xpathConfigs {
					value += fmt.Sprintf(fileSingleLineValueFormat, p.context.prefix, xpc.Key)
					value += ":"
				}
				value = fmt.Sprintf("\"%s\"", strings.TrimRight(value, ":"))
			}
		}
		fmt.Fprintln(p.context.out, value)
		return true
	case config.XPathStrategyNewline:
		key := fmt.Sprintf(multilineKeyFormat, v)
		fmt.Fprintln(p.context.out, indentsFromSlice(key, nindent, hasSliceIndex))

		var value string
		key, shared := p.config.GetKeyFromSharedValues(&xpathConfig)
		if isGlobalConfig {
			// selector:
			//   {{- include "mychart.selectorLabels" . | nindent 4 }}
			value = fmt.Sprintf(globalMultilineValueFormat, key, (nindent+1)*2)
		} else {
			if shared {
				value = fmt.Sprintf(sharedMultilineValueFormat, key, (nindent+1)*2)
			} else {
				value = fmt.Sprintf(fileMultilineValueFormat, p.context.prefix, key, (nindent+1)*2)
			}
		}
		fmt.Fprintln(p.context.out, indent(value, nindent+1))
		return true
	case config.XPathStrategyControlWith:
		var mixed string
		key, shared := p.config.GetKeyFromSharedValues(&xpathConfig)
		if isGlobalConfig {
			mixed = fmt.Sprintf(globalWithMixedFormat, key, v, (nindent+1)*2)
		} else {
			if shared {
				// {{- with .Values.tolerations }}
				// tolerations:
				//   {{- toYaml . | nindent 8 }}
				// {{- end }}
				mixed = fmt.Sprintf(sharedWithMixedFormat, key, v, (nindent+1)*2)
			} else {
				mixed = fmt.Sprintf(fileWithMixedFormat, p.context.prefix, key, v, (nindent+1)*2)
			}
		}
		fmt.Fprintln(p.context.out, indentsFromSlice(mixed, nindent, hasSliceIndex))
		return true
	case config.XPathStrategyControlIf:
		p.logger.Info("ControlIf not implemented")
	case config.XPathStrategyControlRange:
		p.logger.Info("ControlRange not implemented")
	default:
		panic(fmt.Sprintf("Unknown XPath strategy: %s", xpathConfig.Strategy))
	}

	return false
}

func (p *Processor) processMap(v reflect.Value, nindent int, xpath config.XPath, hasSliceIndex *bool) bool {
	// XXX: The priority of file config is greater than global config.
	if p.processMapOrDie(v, nindent, p.context.fileConfig[xpath], false, *hasSliceIndex) {
		// XXX: For the first element only.
		if *hasSliceIndex {
			*hasSliceIndex = false
		}
		return true
	}
	if p.processMapOrDie(v, nindent, p.config.GlobalConfig[xpath], true, *hasSliceIndex) {
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
			fmt.Fprintln(p.context.out, fmt.Sprintf("\"%s\"", v))
		} else if strings.Contains(s, "\n") {
			fmt.Fprintln(p.context.out, fmt.Sprintf("|\n%s", indent(s, nindent+1)))
		} else {
			fmt.Fprintln(p.context.out, v)
		}
	}
}
