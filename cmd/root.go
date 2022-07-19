package cmd

import (
	"io"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func NewRootCmd(logger *logrus.Logger, out io.Writer, args []string) (*cobra.Command, error) {
	cmd := &cobra.Command{
		Use:          "helm",
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
