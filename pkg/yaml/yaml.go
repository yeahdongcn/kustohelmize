package yaml

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/yeahdongcn/kustohelmize/pkg/config"
	"gopkg.in/yaml.v1"
)

type YAMLProcessor struct {
	logger  *logrus.Logger
	out     io.Writer
	config  *config.Config
	current config.XPaths
}

func NewYAMLProcessor(logger *logrus.Logger, out io.Writer, config *config.Config) *YAMLProcessor {
	return &YAMLProcessor{
		logger: logger,
		out:    out,
		config: config,
	}
}

func (p *YAMLProcessor) Process() error {
	for filename, xpaths := range p.config.Rules {
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

		p.current = xpaths
		d := reflect.ValueOf(data)
		p.walk(d, 0, "")
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

func (p *YAMLProcessor) walk(v reflect.Value, nindent int, parent string) {
	for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		v = v.Elem()
	}
	switch v.Kind() {
	case reflect.Array, reflect.Slice:
		p.logger.Debugf("Array/Slice: %s", parent)
		fmt.Fprintln(p.out)
		for i := 0; i < v.Len(); i++ {
			p.walk(v.Index(i), nindent+1, parent)
		}
	case reflect.Map:
		p.logger.Debugf("Map: %s", parent)
		fmt.Fprintln(p.out)
		for _, k := range v.MapKeys() {
			xpath := k.String()
			if k.Kind() == reflect.Ptr || k.Kind() == reflect.Interface {
				xpath = k.Elem().String()
			}
			xpath = filepath.Join(parent, xpath)

			zz := p.current[xpath]
			p.logger.Debugf("zz: %v", zz)
			switch zz.FieldStrategy {
			case config.FieldStrategyPlain:
				// type: {{ .Values.service.type }}
				key := fmt.Sprintf(singleLineKeyFormat, k)
				fmt.Fprint(p.out, indent(key, nindent))
				value := fmt.Sprintf(singleLineValueFormat, zz.Value)
				fmt.Fprintln(p.out, value)
			case config.FieldStrategyNewline:
				// selector:
				//   {{- include "mychart.selectorLabels" . | nindent 4 }}
				key := fmt.Sprintf(multilineKeyFormat, k)
				fmt.Fprintln(p.out, indent(key, nindent))
				value := fmt.Sprintf(multilineValueFormat, zz.Value, (nindent+1)*2)
				fmt.Fprintln(p.out, indent(value, nindent+1))
			case config.FieldStrategyIf:
				mixed := fmt.Sprintf(ifMixedFormat, zz.Value, k, (nindent+1)*2)
				fmt.Fprintln(p.out, indent(mixed, nindent))
			default:
				key := fmt.Sprintf(singleLineKeyFormat, k)
				fmt.Fprint(p.out, indent(key, nindent))
				p.walk(v.MapIndex(k), nindent+1, fmt.Sprintf("%s/%s", parent, k))
			}
		}
	default:
		p.logger.Debugf("Default: %s", parent)
		fmt.Fprintln(p.out, v)
		// handle other types
	}
}
