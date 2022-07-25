package yaml

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
	"github.com/yeahdongcn/kustohelmize/pkg/config"
	"gopkg.in/yaml.v1"
)

type YAMLProcessor struct {
	logger            logr.Logger
	out               io.Writer
	config            *config.Config
	destDir           string
	currentFileConfig config.FileConfig
}

func NewYAMLProcessor(logger logr.Logger, destDir string, config *config.Config) *YAMLProcessor {
	return &YAMLProcessor{
		logger:  logger,
		destDir: destDir,
		config:  config,
	}
}

func (p *YAMLProcessor) Process() error {
	for filename, fileConfig := range p.config.FileConfigMap {
		dest := filepath.Join(p.destDir, filepath.Base(filename))
		file, err := os.Create(dest)
		if err != nil {
			p.logger.Error(err, "Error creating file", "dest", dest)
			return err
		}
		defer file.Close()
		p.out = file

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

		p.currentFileConfig = fileConfig
		d := reflect.ValueOf(data)
		p.walk(d, 0, config.XPathRoot, false)
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

func (p *YAMLProcessor) handleSlice() {

}

func (p *YAMLProcessor) handleMap(nindent int, k reflect.Value, xpath config.XPath) bool {
	aa := p.currentFileConfig[xpath]
	xpathConfigs := []config.XPathConfig{aa, p.config.GlobalConfig[xpath]}

	for _, xpathConfig := range xpathConfigs {
		switch xpathConfig.Strategy {
		case config.XPathStrategyInline:
			// name: {{ include "mychart.fullname" . }}
			key := fmt.Sprintf(singleLineKeyFormat, k)
			fmt.Fprint(p.out, indent(key, nindent))
			value := fmt.Sprintf(singleLineValueFormat, xpathConfig.Value)
			fmt.Fprintln(p.out, value)
			return true
		case config.XPathStrategyNewline:
			// selector:
			//   {{- include "mychart.selectorLabels" . | nindent 4 }}
			key := fmt.Sprintf(multilineKeyFormat, k)
			fmt.Fprintln(p.out, indent(key, nindent))
			value := fmt.Sprintf(multilineValueFormat, xpathConfig.Value, (nindent+1)*2)
			fmt.Fprintln(p.out, indent(value, nindent+1))
			return true
		case config.XPathStrategyControlWith:
			// {{- with .Values.tolerations }}
			// tolerations:
			//   {{- toYaml . | nindent 8 }}
			// {{- end }}
			mixed := fmt.Sprintf(withMixedFormat, xpathConfig.Value, k, (nindent+1)*2)
			fmt.Fprintln(p.out, indent(mixed, nindent))
			return true
		case config.XPathStrategyControlIf:
			p.logger.Info("ControlIf not implemented")
		case config.XPathStrategyControlRange:
			p.logger.Info("ControlRange not implemented")
		default:
			// p.logger.Debugf("Unknown strategy: %s", xpathConfig.Strategy)
		}
	}

	return false
}

func (p *YAMLProcessor) walk(v reflect.Value, nindent int, root config.XPath, fromSlice bool) {
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
			fmt.Fprintln(p.out)
		}
		if continueProcess {
			for i := 0; i < v.Len(); i++ {
				if i == 0 {
					fmt.Fprint(p.out, indent(slicePrefixFirstLineFormat, nindent))
				} else {
					fmt.Fprint(p.out, indent(slicePrefixOtherLineFormat, nindent))
				}
				p.walk(v.Index(i), nindent+1, root, true)
			}
		} else {
			for i := 0; i < v.Len(); i++ {
				fmt.Fprintln(p.out, indent(fmt.Sprintf("- %s", v.Index(i)), nindent))
			}
		}
	case reflect.Map:
		p.logger.V(10).Info("Handling map", "root", root)

		if !root.IsRoot() && !fromSlice {
			fmt.Fprintln(p.out)
		}
		for _, k := range v.MapKeys() {
			mapKey := k.String()
			if k.Kind() == reflect.Ptr || k.Kind() == reflect.Interface {
				mapKey = k.Elem().String()
			}
			xpath := root.NewChild(mapKey)
			if !p.handleMap(nindent, k, xpath) {
				key := fmt.Sprintf(singleLineKeyFormat, k)
				if fromSlice {
					fmt.Fprint(p.out, xindent(key, nindent, fromSlice))
					fromSlice = false
				} else {
					fmt.Fprint(p.out, xindent(key, nindent, fromSlice))
				}
				p.walk(v.MapIndex(k), nindent+1, xpath, false)
			}
		}
	default:
		str := v.String()
		p.logger.V(10).Info("Handling default", "root", root, "str", str)
		// spec.template.spec.nodeSelector: Invalid type. Expected: [string,null], given: boolean
		if str == "true" || str == "false" {
			fmt.Fprintln(p.out, fmt.Sprintf("\"%s\"", v))
		} else {
			fmt.Fprintln(p.out, v)
		}

		// panic(11)
		// p.logger.Debugf("Default: %s", root)
		// p.logger.Debugf("%s", v.String())

		// // handle other types
	}
}
