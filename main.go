package main

import (
	"os"

	"github.com/sirupsen/logrus"
	"github.com/yeahdongcn/kustohelmize/cmd"
)

func main() {
	logger := logrus.New()
	logger.Level = logrus.DebugLevel

	cmd, err := cmd.NewRootCmd(logger, os.Stdout, os.Args[1:])
	if err != nil {
		logger.Error(err)
		os.Exit(1)
	}

	if err := cmd.Execute(); err != nil {
		logger.Debug(err)
		os.Exit(1)
	}
}
