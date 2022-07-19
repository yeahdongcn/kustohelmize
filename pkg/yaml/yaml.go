package yaml

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"reflect"
	"strings"

	"gopkg.in/yaml.v1"
)

type YAMLProcessor struct {
	out  io.Writer
	data map[string]interface{}
}

func NewYAMLProcessor(out io.Writer, filename string) (*YAMLProcessor, error) {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	data := make(map[string]interface{})
	err = yaml.Unmarshal(b, &data)
	if err != nil {
		return nil, err
	}
	return &YAMLProcessor{
		out:  out,
		data: data,
	}, nil
}

func (p *YAMLProcessor) Process() {
	d := reflect.ValueOf(p.data)
	p.walk(d, 0, "$")
}

func (p *YAMLProcessor) walk(v reflect.Value, level int, parent string) {
	for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		v = v.Elem()
	}
	switch v.Kind() {
	case reflect.Array, reflect.Slice:
		// fmt.Fprintln(p.out, parent)
		fmt.Fprintln(p.out)
		for i := 0; i < v.Len(); i++ {
			p.walk(v.Index(i), level+1, parent)
		}
	case reflect.Map:
		// fmt.Fprintln(p.out, parent)
		fmt.Fprintln(p.out)
		for _, k := range v.MapKeys() {
			ident := ""
			for i := 0; i < level; i++ {
				ident += "  "
			}

			xx := k.String()
			if k.Kind() == reflect.Interface {
				xx = k.Elem().String()
			}

			if xx == "selector" {
				fmt.Fprintf(p.out, "%s%s: ", ident, k)
				fmt.Fprintln(p.out, "{{- include \"mychart.selectorLabels\" . | nindent 4 }}")
			} else if xx == "ports" {
				x := `{{- with .Values.ports }}
ports:
  {{- toYaml . | nindent 4 }}
{{- end }}`

				scanner := bufio.NewScanner(strings.NewReader(x))
				for scanner.Scan() {
					fmt.Fprintf(p.out, "%s%s\n", ident, scanner.Text())
				}
			} else {
				fmt.Fprintf(p.out, "%s%s: ", ident, k)
				p.walk(v.MapIndex(k), level+1, fmt.Sprintf("%s/%s", parent, k))
			}

		}
	default:
		// fmt.Fprintln(p.out, parent)
		fmt.Fprintln(p.out, v)
		// handle other types
	}
}
