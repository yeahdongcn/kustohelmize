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
	for filename, fileConfig := range p.config.PerFileConfig {
		z := filepath.Base(filename)
		// TODO: Remove this
		if !strings.Contains(z, "mt-controller-manager-deployment.yaml") {
			continue
		}
		dest := filepath.Join(p.destDir, z)
		if util.IsCustomResourceDefinition(z) {
			err := exec.Command("cp", "-f", filename, dest).Run()
			if err != nil {
				p.logger.Error(err, "Failed to copy file", "source", filename, "dest", dest)
				return err
			}
			continue
		}

		file, err := os.Create(dest)
		if err != nil {
			p.logger.Error(err, "Error creating file", "dest", dest)
			return err
		}
		defer file.Close()

		b, err := ioutil.ReadFile(filename)
		if err != nil {
			p.logger.Error(err, "Error reading YAML", "filename", filename)
			return err
		}
		data := make(map[string]interface{})
		err = yaml.Unmarshal(b, &data)
		if err != nil {
			p.logger.Error(err, "Error unmarshalling YAML", "filename", filename)
			return err
		}

		p.context = context{
			out:        file,
			prefix:     util.FilenameWithoutExt(z),
			fileConfig: fileConfig,
		}
		d := reflect.ValueOf(data)
		p.walk(d, 0, config.XPathRoot, config.XPathSliceIndexNone)
	}

	return nil
}

func indent(s string, n int) string {
	return xindent(s, n, false)
}

func xindent(s string, n int, fromSlice bool) string {
	if n < 0 {
		n = 0
	}
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

func (p *Processor) handleSlice() {

}

func (p *Processor) processMapOrDie(v reflect.Value, nindent int, xpathConfig config.XPathConfig, isGlobalConfig bool) bool {
	if (xpathConfig == config.XPathConfig{}) {
		return false
	}
	switch xpathConfig.Strategy {
	case config.XPathStrategyInline:
		// name: {{ include "mychart.fullname" . }}
		key := fmt.Sprintf(singleLineKeyFormat, v)
		fmt.Fprint(p.context.out, indent(key, nindent))

		var value string
		if isGlobalConfig {
			value = fmt.Sprintf(globalSingleLineValueFormat, xpathConfig.Key)
		} else {
			value = fmt.Sprintf(perFileSingleLineValueFormat, p.context.prefix, xpathConfig.Key)
		}
		fmt.Fprintln(p.context.out, value)
		return true
	case config.XPathStrategyNewline:
		// selector:
		//   {{- include "mychart.selectorLabels" . | nindent 4 }}
		key := fmt.Sprintf(multilineKeyFormat, v)
		fmt.Fprintln(p.context.out, indent(key, nindent))

		var value string
		if isGlobalConfig {
			value = fmt.Sprintf(globalMultilineValueFormat, xpathConfig.Key, (nindent+1)*2)
		} else {
			value = fmt.Sprintf(perFileMultilineValueFormat, p.context.prefix, xpathConfig.Key, (nindent+1)*2)
		}

		fmt.Fprintln(p.context.out, indent(value, nindent+1))
		return true
	case config.XPathStrategyControlWith:
		// {{- with .Values.tolerations }}
		// tolerations:
		//   {{- toYaml . | nindent 8 }}
		// {{- end }}
		mixed := fmt.Sprintf(withMixedFormat, xpathConfig.Key, v, (nindent+1)*2)
		fmt.Fprintln(p.context.out, indent(mixed, nindent))
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

func (p *Processor) processMap(v reflect.Value, nindent int, xpath config.XPath) bool {
	// XXX: The priority of file config is greater than global config.
	if p.processMapOrDie(v, nindent, p.context.fileConfig[xpath], false) {
		return true
	}
	if p.processMapOrDie(v, nindent, p.config.GlobalConfig[xpath], true) {
		return true
	}

	return false
}

func (p *Processor) walk(v reflect.Value, nindent int, root config.XPath, sliceIndex int) {
	for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		v = v.Elem()
	}
	switch v.Kind() {
	case reflect.Array, reflect.Slice:
		p.logger.V(10).Info("Handling slice", "root", root)

		continueProcess := false
		if v.Len() > 0 {
			first := v.Index(0)
			for first.Kind() == reflect.Ptr || first.Kind() == reflect.Interface {
				first = first.Elem()
			}
			if first.Kind() == reflect.Map {
				continueProcess = true
			}
			if first.Kind() == reflect.Slice {
				panic(10)
			}
		}
		if !root.IsRoot() && !continueProcess {
			fmt.Fprintln(p.context.out)
		}
		if continueProcess {
			for i := 0; i < v.Len(); i++ {
				if i == 0 {
					fmt.Fprint(p.context.out, indent(slicePrefixFirstLineFormat, nindent))
				} else {
					fmt.Fprint(p.context.out, indent(slicePrefixOtherLineFormat, nindent))
				}
				p.walk(v.Index(i), nindent+1, root, i)
			}
		} else {
			for i := 0; i < v.Len(); i++ {
				x := v.Index(i)
				if x.Kind() == reflect.Ptr || x.Kind() == reflect.Interface {
					x = x.Elem()
				}
				str := x.String()
				p.logger.Info(str)
				if str == "" {
					fmt.Fprintln(p.context.out, indent(fmt.Sprintf("- \"%s\"", x), nindent))
				} else if str == "*" {
					fmt.Fprintln(p.context.out, indent(fmt.Sprintf("- '%s'", x), nindent))
				} else {
					fmt.Fprintln(p.context.out, indent(fmt.Sprintf("- %s", x), nindent))
				}
			}
		}
	case reflect.Map:
		p.logger.V(10).Info("Handling map", "root", root)

		if !root.IsRoot() && sliceIndex == config.XPathSliceIndexNone {
			fmt.Fprintln(p.context.out)
		}
		xyz := sliceIndex != config.XPathSliceIndexNone
		for _, k := range v.MapKeys() {
			mapKey := k.String()
			if k.Kind() == reflect.Ptr || k.Kind() == reflect.Interface {
				mapKey = k.Elem().String()
			}
			xpath := root.NewChild(mapKey, sliceIndex)
			if !p.processMap(k, nindent, xpath) {
				key := fmt.Sprintf(singleLineKeyFormat, k)
				if xyz {
					fmt.Fprint(p.context.out, xindent(key, nindent, true))
					xyz = false
				} else {
					fmt.Fprint(p.context.out, xindent(key, nindent, false))
				}
				p.walk(v.MapIndex(k), nindent+1, xpath, -1)
			}
		}
	default:
		// spec.template.spec.nodeSelector: Invalid type. Expected: [string,null], given: boolean
		if v.Kind() == reflect.Invalid {
			fmt.Fprintln(p.context.out, "null")
			return
		}
		str := v.String()
		p.logger.V(10).Info("Handling default", "root", root, "str", str)
		if str == "true" || str == "false" {
			fmt.Fprintln(p.context.out, fmt.Sprintf("\"%s\"", v))
		} else if strings.Contains(str, "\n") {
			fmt.Fprintln(p.context.out, fmt.Sprintf("|\n%s", v))
		} else {
			fmt.Fprintln(p.context.out, v)
		}
	}
}
