package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

func newVersionCmd(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			version := "v0.0.1"
			fmt.Fprintf(out, "version: %s\n", version)
			return nil
		},
	}

	return cmd
}
