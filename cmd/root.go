package cmd

import (
	"io"

	"github.com/spf13/cobra"
)

func NewRootCmd(out io.Writer, args []string) (*cobra.Command, error) {
	cmd := &cobra.Command{
		Use:          "helm",
		Short:        "The Helm package manager for Kubernetes.",
		Long:         ``,
		SilenceUsage: true,
	}

	// Add subcommands
	cmd.AddCommand(
		newCreateCmd(out),
	)

	return cmd, nil
}
