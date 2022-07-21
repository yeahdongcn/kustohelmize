package yaml

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"reflect"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/yeahdongcn/kustohelmize/pkg/config"
	"gopkg.in/yaml.v1"
)

type YAMLProcessor struct {
	logger            *logrus.Logger
	out               io.Writer
	config            *config.Config
	currentFileConfig config.FileConfig
}

func NewYAMLProcessor(logger *logrus.Logger, out io.Writer, config *config.Config) *YAMLProcessor {
	return &YAMLProcessor{
		logger: logger,
		out:    out,
		config: config,
	}
}

func (p *YAMLProcessor) Process() error {
	for filename, fileConfig := range p.config.FileConfigMap {
		b, err := ioutil.ReadFile(filename)
		if err != nil {
			p.logger.Errorf("Error reading file %s: %v", filename, err)
			return err
		}
		data := make(map[string]interface{})
		err = yaml.Unmarshal(b, &data)
		if err != nil {
			p.logger.Errorf("Error unmarshalling file %s: %v", filename, err)
			return err
		}

		p.currentFileConfig = fileConfig
		d := reflect.ValueOf(data)
		p.walk(d, 0, config.XPathRoot)
	}

	return nil
}

func indent(s string, n int) string {
	indents := strings.Repeat(defaultIndent, n)

	indented := ""
	scanner := bufio.NewScanner(strings.NewReader(s))
	for scanner.Scan() {
		indented += indents + scanner.Text() + "\n"
	}
	indented = strings.Trim(indented, "\n")

	return indented
}

func (p *YAMLProcessor) handleSlice() {

}

func (p *YAMLProcessor) handleMap(v reflect.Value, nindent int, k reflect.Value, xpath config.XPath) bool {
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
			p.logger.Debug("ControlIf not implemented")
		case config.XPathStrategyControlRange:
			p.logger.Debug("ControlRange not implemented")
		default:
			// p.logger.Debugf("Unknown strategy: %s", xpathConfig.Strategy)
		}
	}

	return false
}

func (p *YAMLProcessor) walk(v reflect.Value, nindent int, root config.XPath) {
	for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		v = v.Elem()
	}
	switch v.Kind() {
	case reflect.Array, reflect.Slice:
		p.logger.Debugf("Array/Slice: %s", root)

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
				p.walk(v.Index(i), nindent+1, root)
			}
		} else {
			fmt.Fprintln(p.out, indent(fmt.Sprintf("%s", v), nindent))
		}
	case reflect.Map:
		p.logger.Debugf("Map: %s", root)
		if !root.IsRoot() {
			fmt.Fprintln(p.out)
		}
		for _, k := range v.MapKeys() {
			mapKey := k.String()
			if k.Kind() == reflect.Ptr || k.Kind() == reflect.Interface {
				mapKey = k.Elem().String()
			}
			xpath := root.NewChild(mapKey)
			if !p.handleMap(v, nindent, k, xpath) {
				key := fmt.Sprintf(singleLineKeyFormat, k)
				fmt.Fprint(p.out, indent(key, nindent))
				p.walk(v.MapIndex(k), nindent+1, xpath)
			}
		}
	default:
		p.logger.Debugf("Default: %s", root)
		p.logger.Debugf("%s", v.String())
		fmt.Fprintln(p.out, v)
		// panic(11)
		// p.logger.Debugf("Default: %s", root)
		// p.logger.Debugf("%s", v.String())

		// // handle other types
	}
}
