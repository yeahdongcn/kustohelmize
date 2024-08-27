package cmd

import (
	"io"

	"github.com/go-logr/logr"
	"github.com/spf13/cobra"
)

func NewRootCmd(logger logr.Logger, out io.Writer, args []string) (*cobra.Command, error) {
	cmd := &cobra.Command{
		Use:          "kustohelmize",
		Short:        "Automate Helm chart creation from any existing Kubernetes manifests",
		Long:         ``,
		SilenceUsage: true,
	}

	// Add subcommands
	cmd.AddCommand(
		newCreateCmd(logger, out),
		newVersionCmd(out),
	)

	return cmd, nil
}
