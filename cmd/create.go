package cmd

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/yeahdongcn/kustohelmize/pkg/config"
	"github.com/yeahdongcn/kustohelmize/pkg/yaml"
	goyaml "gopkg.in/yaml.v1"
	"helm.sh/helm/v3/cmd/helm/require"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/helmpath"
)

type createOptions struct {
	starter    string // --starter
	name       string
	starterDir string
}

func newCreateCmd(logger *logrus.Logger, out io.Writer) *cobra.Command {
	o := &createOptions{}

	cmd := &cobra.Command{
		Use:   "create NAME",
		Short: "",
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
			o.starterDir = helmpath.DataPath("starters")
			return o.run(logger, out)
		},
	}

	cmd.Flags().StringVarP(&o.starter, "starter", "p", "", "the name or absolute path to Helm starter scaffold")
	return cmd
}

func (o *createOptions) run(logger *logrus.Logger, out io.Writer) error {
	fmt.Fprintf(out, "Creating %s\n", o.name)
	fmt.Fprintf(out, "Creating %s\n", filepath.Dir(o.name))

	file, err := os.Create(filepath.Join(filepath.Dir(o.name), o.name))
	if err != nil {
		logger.Errorf("Error creating file: %s", err)
		return err
	}
	defer file.Close()

	chartname := filepath.Base(o.name)
	config := &config.Config{
		Chartname: chartname,
		FileConfigMap: map[string]config.FileConfig{
			filepath.Join(".", "test", "testdata", "service.yaml"): {
				config.XPath("/spec/type"): {
					Strategy: config.XPathStrategyInline,
					Value:    "Values.xxx",
				},
				config.XPath("/spec/selector"): {
					Strategy: config.XPathStrategyNewline,
					Value:    "Values.yyy",
				},
				config.XPath("/spec/ports"): {
					Strategy: config.XPathStrategyControlWith,
					Value:    "Values.zzz",
				},
			},
		},
	}
	// TODO: Remove this
	output, err := goyaml.Marshal(config)
	if err != nil {
		return err
	}
	ioutil.WriteFile(filepath.Join(filepath.Dir(o.name), "config.yaml"), output, 0644)
	p := yaml.NewYAMLProcessor(logger, file, config)
	err = p.Process()
	if err != nil {
		logger.Errorf("Error processing YAML: %s", err)
		return err
	}
	return nil

	// TODO:
	cfile := &chart.Metadata{
		Name:        chartname,
		Description: "A Helm chart for Kubernetes",
		Type:        "application",
		Version:     "0.1.0",
		AppVersion:  "0.1.0",
		APIVersion:  chart.APIVersionV2,
	}

	if o.starter != "" {
		// Create from the starter
		lstarter := filepath.Join(o.starterDir, o.starter)
		// If path is absolute, we don't want to prefix it with helm starters folder
		if filepath.IsAbs(o.starter) {
			lstarter = o.starter
		}
		return chartutil.CreateFrom(cfile, filepath.Dir(o.name), lstarter)
	}

	chartutil.Stderr = out
	_, err = chartutil.Create(chartname, filepath.Dir(o.name))
	return err
}
