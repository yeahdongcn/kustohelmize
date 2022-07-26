package value

import (
	"github.com/go-logr/logr"
	"github.com/yeahdongcn/kustohelmize/pkg/config"
)

type ValueProcessor struct {
	logger logr.Logger
	config *config.Config
}

func NewValueProcessor(logger logr.Logger, config *config.Config) *ValueProcessor {
	return &ValueProcessor{
		logger: logger,
		config: config,
	}
}

func (p *ValueProcessor) Process() error {
	return nil
}
