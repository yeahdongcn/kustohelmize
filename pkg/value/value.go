package value

import (
	"os"
	"path/filepath"

	"github.com/go-logr/logr"
	"github.com/yeahdongcn/kustohelmize/pkg/chart"
	"github.com/yeahdongcn/kustohelmize/pkg/config"
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

	values, err := p.config.Values()
	if err != nil {
		return err
	}
	for _, str := range []string{chart.Header, values} {
		_, err = file.WriteString(str)
		if err != nil {
			return err
		}
	}

	return nil
}
