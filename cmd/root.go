package cmd

import (
	"io"

	"github.com/go-logr/logr"
	"github.com/spf13/cobra"
)

func NewRootCmd(logger logr.Logger, out io.Writer, args []string) (*cobra.Command, error) {
	cmd := &cobra.Command{
		Use:          "xxx",
		Short:        "The Helm package manager for Kubernetes.",
		Long:         ``,
		SilenceUsage: true,
	}

	// Add subcommands
	cmd.AddCommand(
		newCreateCmd(logger, out),
	)

	return cmd, nil
}
