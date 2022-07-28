package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yeahdongcn/kustohelmize/cmd/require"
)

// version must be set by go build's -X main.version= option in the Makefile.
var version = "unknown"

// gitCommit will be the hash that the binary was built from
// and will be populated by the Makefile
var gitCommit = "unknown"

// gitTreeState will be the state of the git tree that the binary was built from
// and will be populated by the Makefile
var gitTreeState = "unknown"

// getVersionParts returns the different version components
func getVersionParts() []string {
	v := []string{"Version: " + version}

	if gitCommit != "" {
		v = append(v, "GitCommit: "+gitCommit)
	}

	if gitTreeState != "" {
		v = append(v, "GitTreeState: "+gitTreeState)
	}

	return v
}

// GetVersionString returns the string representation of the version
func getVersionString(more ...string) string {
	v := append(getVersionParts(), more...)
	return strings.Join(v, ", ")
}

const versionDesc = `
Show the version for Kustohelmize.

This will print a representation the version of Kustohelmize.
The output will look something like this:

Version: 1.0.0, GitCommit: a2864dacb7b21b1efdad9ceb063f69ade4010738, GitTreeState: dirty

- Version is the semantic version of the release.
- GitCommit is the SHA for the commit that this version was built from.
- GitTreeState is "clean" if there are no local code changes when this binary was
  built, and "dirty" if the binary was built from locally modified code.
`

func newVersionCmd(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "print the version information about Kustohelmize",
		Long:  versionDesc,
		Args:  require.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintln(out, getVersionString())
		},
	}

	return cmd
}
