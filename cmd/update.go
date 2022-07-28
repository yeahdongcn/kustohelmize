package cmd

import (
	"io"

	"github.com/go-logr/logr"
	"github.com/spf13/cobra"
)

type updateOptions struct {
	logger logr.Logger

	name string
}

func newUpdateCmd(logger logr.Logger, out io.Writer) *cobra.Command {
	o := &updateOptions{
		logger: logger.WithName("update"),
	}

	cmd := &cobra.Command{
		Use: "",
		RunE: func(cmd *cobra.Command, args []string) error {
			o.name = args[0]
			return nil
		},
	}
	return cmd
}
