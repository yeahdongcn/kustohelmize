package value

import (
	"fmt"
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

	out, err := yaml.Marshal(p.config.SharedValues)
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("%s\n", string(out)))
	if err != nil {
		return err
	}

	for filename, fileConfig := range p.config.FileConfig {
		z := filepath.Base(filename)
		p.logger.Info("Processing file", "filename", z, "config", fileConfig)

		x := make(map[string]interface{})
		x[util.FilenameWithoutExt(z)] = make(map[string]interface{})
		root := x[util.FilenameWithoutExt(z)]

		for _, v := range fileConfig {
			newroot := root.(map[string]interface{})
			substrings := strings.Split(v.Key, ".")
			for i, substring := range substrings {
				if i == 0 && substring == "sharedValues" {
					break
				}
				if newroot[substring] == nil {
					newroot[substring] = make(map[string]interface{})
				}
				if i < len(substrings)-1 {
					newroot = newroot[substring].(map[string]interface{})
				} else {
					p.logger.Info("Setting value", "key", v.Key, "substring", substring, "value", v.Value)
					newroot[substring] = v.Value
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
