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

func (p *YAMLProcessor) walk(v reflect.Value, nindent int, root config.XPath) {
	for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		v = v.Elem()
	}
	switch v.Kind() {
	case reflect.Array, reflect.Slice:
		p.logger.Debugf("Array/Slice: %s", root)
		if !root.IsRoot() {
			fmt.Fprintln(p.out)
		}
		for i := 0; i < v.Len(); i++ {
			p.walk(v.Index(i), nindent+1, root)
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

			xpathConfig := p.currentFileConfig[xpath]
			switch xpathConfig.Strategy {
			case config.XPathStrategyInline:
				// type: {{ .Values.service.type }}
				key := fmt.Sprintf(singleLineKeyFormat, k)
				fmt.Fprint(p.out, indent(key, nindent))
				value := fmt.Sprintf(singleLineValueFormat, xpathConfig.Value)
				fmt.Fprintln(p.out, value)
			case config.XPathStrategyNewline:
				// selector:
				//   {{- include "mychart.selectorLabels" . | nindent 4 }}
				key := fmt.Sprintf(multilineKeyFormat, k)
				fmt.Fprintln(p.out, indent(key, nindent))
				value := fmt.Sprintf(multilineValueFormat, xpathConfig.Value, (nindent+1)*2)
				fmt.Fprintln(p.out, indent(value, nindent+1))
			case config.XPathStrategyControlWith:
				mixed := fmt.Sprintf(withMixedFormat, xpathConfig.Value, k, (nindent+1)*2)
				fmt.Fprintln(p.out, indent(mixed, nindent))
			case config.XPathStrategyControlIf:
				p.logger.Debug("ControlIf not implemented")
			case config.XPathStrategyControlRange:
				p.logger.Debug("ControlRange not implemented")
			default:
				key := fmt.Sprintf(singleLineKeyFormat, k)
				fmt.Fprint(p.out, indent(key, nindent))
				p.walk(v.MapIndex(k), nindent+1, xpath)
			}
		}
	default:
		p.logger.Debugf("Default: %s", root)
		fmt.Fprintln(p.out, v)
		// handle other types
	}
}
