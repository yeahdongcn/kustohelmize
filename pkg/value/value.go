package value

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/go-logr/logr"
	"github.com/yeahdongcn/kustohelmize/pkg/config"
	"github.com/yeahdongcn/kustohelmize/pkg/util"
	"gopkg.in/yaml.v1"
	"k8s.io/helm/pkg/chartutil"
)

type Processor struct {
	logger  logr.Logger
	config  *config.ChartConfig
	destDir string
}

func NewProcessor(logger logr.Logger, config *config.ChartConfig, destDir string) *Processor {
	return &Processor{
		logger:  logger,
		config:  config,
		destDir: destDir,
	}
}

func (p *Processor) Process() error {
	path := filepath.Join(p.destDir, chartutil.ValuesfileName)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		p.logger.Error(err, "Error opening file", "path", path)
		return err
	}
	defer file.Close()
	for filename, fileConfig := range p.config.PerFileConfig {
		z := filepath.Base(filename)
		p.logger.Info("Processing file", "filename", z, "config", fileConfig)

		x := make(map[string]interface{})
		x[util.FilenameWithoutExt(z)] = make(map[string]interface{})
		root := x[util.FilenameWithoutExt(z)]

		for _, v := range fileConfig {
			newroot := root.(map[string]interface{})
			components := strings.Split(v.Key, ".")
			for i, com := range components {
				if newroot[com] == nil {
					newroot[com] = make(map[string]interface{})
				}
				if i < len(components)-1 {
					newroot = newroot[com].(map[string]interface{})
				} else {
					newroot[com] = v.Value
				}
			}
		}

		xx := root.(map[string]interface{})
		if len(xx) == 0 {
			continue
		}
		out, err := yaml.Marshal(x)
		if err != nil {
			p.logger.Error(err, "Error marshalling file", "filename", z)
			return err
		}
		_, err = file.Write(out)
		if err != nil {
			p.logger.Error(err, "Error writing file", "filename", z)
			return err
		}
	}
	return nil
}

func (p *Processor) walk(x string, root interface{}) {
	switch x := root.(type) {
	case map[string]interface{}:
		for k, v := range x {
			p.walk(k, v)
		}
	case []interface{}:
		for _, v := range x {
			p.walk("", v)
		}
	case string:
		p.logger.Info("Found string", "x", x)
	case int:
		p.logger.Info("Found int", "x", x)
	case bool:
		p.logger.Info("Found bool", "x", x)
	default:
		p.logger.Info("Found unknown", "x", x)
	}
}
