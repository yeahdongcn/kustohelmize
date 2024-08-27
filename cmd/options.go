package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/go-logr/logr"
)

type options struct {
	logger logr.Logger

	intermediateDir              string
	enableIntermediateDirCleanup bool
	config                       string

	// From helm.
	name string
}

func (o *options) chartroot() string {
	return filepath.Dir(o.name)
}

func (o *options) chartname() string {
	return filepath.Base(o.name)
}

func (o *options) configPath() string {
	return filepath.Join(filepath.Dir(o.name), fmt.Sprintf("%s.config", o.chartname()))
}
