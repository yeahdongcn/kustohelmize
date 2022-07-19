package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	yaml "gopkg.in/yaml.v3"
)

func main() {
	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	chartName := "mychart"
	filename := filepath.Join(wd, "test", "testdata", "service.yaml")
	out, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatal(err)
	}
	data := make(map[string]interface{})

	err = yaml.Unmarshal(out, &data)
	if err != nil {
		log.Fatal(err)
	}

	d := reflect.ValueOf(data)
	walk(d, 0)

	return

	fmt.Println(data)

	x := data["metadata"].(map[string]interface{})
	x["name"] = fmt.Sprintf("{{ include \"%s.fullname\" . }}", chartName)
	// service.DeletionTimestamp = fmt.Sprintf("{{ include \"%s.fullname\" . }}", chartName)
	x = data["spec"].(map[string]interface{})
	x["ports"] = []map[string]interface{}{
		{
			"port": "{{ .Values.service.port }}",
			"xxx":  "dfsdf",
		},
	}
	// x["ports"] = "".(inter)
	out, err = yaml.Marshal(data)
	if err != nil {
		log.Fatal(err)
	}
	y := removeEmptyFields(string(out))
	fmt.Println(y)
}

func walk(v reflect.Value, level int8) {
	for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		v = v.Elem()
	}
	switch v.Kind() {
	case reflect.Array, reflect.Slice:
		fmt.Println("")
		for i := 0; i < v.Len(); i++ {
			walk(v.Index(i), level+1)
		}
	case reflect.Map:
		fmt.Println("")
		for _, k := range v.MapKeys() {
			ident := ""
			for i := 0; i < int(level); i++ {
				ident += "  "
			}

			if k.String() == "selector" {
				fmt.Printf("%s%s: ", ident, k)
				fmt.Println("{{- include \"mychart.selectorLabels\" . | nindent 4 }}")
			} else if k.String() == "ports" {
				x := `{{- with .Values.ports }}
ports:
  {{- toYaml . | nindent 4 }}
{{- end }}`

				scanner := bufio.NewScanner(strings.NewReader(x))
				for scanner.Scan() {
					fmt.Printf("%s%s\n", ident, scanner.Text())
				}
			} else {
				fmt.Printf("%s%s: ", ident, k)
				walk(v.MapIndex(k), level+1)
			}

		}
	default:
		fmt.Println(v)
		// handle other types
	}
}
