package value

import (
	"github.com/go-logr/logr"
	"github.com/yeahdongcn/kustohelmize/pkg/config"
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
	return nil
}
