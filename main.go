package main

import (
	"flag"
	"os"

	"github.com/yeahdongcn/kustohelmize/cmd"
	"go.uber.org/zap/zapcore"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func main() {
	opts := zap.Options{
		Development: true,
		Level:       zapcore.Level(-10),
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	logger := zap.New(zap.UseFlagOptions(&opts))

	cmd, err := cmd.NewRootCmd(logger, os.Stdout, os.Args[1:])
	if err != nil {
		logger.Error(err, "Error creating root command")
		os.Exit(1)
	}

	if err := cmd.Execute(); err != nil {
		logger.Error(err, "Error executing root command")
		os.Exit(1)
	}
}
