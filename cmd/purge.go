package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/go-logr/logr"
	"github.com/spf13/cobra"
	cfg "github.com/yeahdongcn/kustohelmize/pkg/config"
	"gopkg.in/yaml.v2"
	"helm.sh/helm/v3/cmd/helm/require"
)

type purgeOptions struct {
	options
}

func newPurgeCmd(logger logr.Logger, out io.Writer) *cobra.Command {
	o := &purgeOptions{
		options: options{
			logger: logger.WithName("purge"),
		},
	}

	cmd := &cobra.Command{
		Use:   "purge NAME",
		Short: "Purge a generated Helm chart",
		Long:  ``,
		Args:  require.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				// Allow file completion when completing the argument for the name
				// which could be a path
				return nil, cobra.ShellCompDirectiveDefault
			}
			// No more completions, so disable file completion
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			o.name = args[0]
			if o.intermediateDir == "" {
				o.intermediateDir = fmt.Sprintf("%s-%s", o.name, "generated")
			}
			if o.enableIntermediateDirCleanup {
				defer os.RemoveAll(o.intermediateDir)
			}

			return o.run(out)
		},
	}

	cmd.Flags().StringVarP(&o.intermediateDir, "intermediate-dir", "i", "", "The path to a intermediate directory")
	cmd.Flags().MarkHidden("intermediate-dir")
	cmd.Flags().BoolVarP(&o.enableIntermediateDirCleanup, "cleanup", "", false, "Whether to cleanup the intermediate directory")
	cmd.Flags().MarkHidden("cleanup")
	cmd.Flags().StringVarP(&o.config, "config", "c", "", "The path to a config file")
	cmd.Flags().MarkHidden("config")

	return cmd
}

func (o *purgeOptions) purgeConfig() ([]string, error) {
	path := o.configPath()
	logger := o.logger.WithName("config")
	_, err := os.Stat(path)
	if err != nil {
		o.logger.Info("Config file does not exist", "path", path)
		return nil, err
	}

	out, err := os.ReadFile(path)
	if err != nil {
		o.logger.Error(err, "Error reading config file", "path", path)
		return nil, err
	}
	config := &cfg.ChartConfig{Logger: logger}
	err = yaml.Unmarshal(out, config)
	if err != nil {
		o.logger.Error(err, "Error unmarshalling config file", "path", path)
		return nil, err
	}

	c, _ := os.ReadDir(o.intermediateDir)
	names := make(map[string]any)
	for _, entry := range c {
		names[entry.Name()] = struct{}{}
	}
	redundancies := make([]string, 0)
	for path := range config.FileConfig {
		name := filepath.Base(path)
		if _, ok := names[name]; !ok {
			redundancies = append(redundancies, path)
		}
	}

	return redundancies, nil
}

func (o *purgeOptions) run(out io.Writer) error {
	o.logger.Info("Purging chart", "name", o.name)

	redundancies, err := o.purgeConfig()
	if err != nil {
		o.logger.Error(err, "Error purging config")
		return err
	}

	if len(redundancies) == 0 {
		o.logger.Info("No files to purge")
		return nil
	}

	for _, redundancy := range redundancies {
		fmt.Fprintf(out, "Redundant file config found: %s\n", redundancy)
	}

	return nil
}
