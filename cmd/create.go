package cmd

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/go-logr/logr"
	"github.com/spf13/cobra"
	cfg "github.com/yeahdongcn/kustohelmize/pkg/config"
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

	// TODO: 1. Config file path 2. Source YAML file directory
	from                         string
	kubernetesSplitYamlCommand   string
	intermediateDir              string
	enableIntermediateDirCleanup bool

	logger logr.Logger
}

func newCreateCmd(logger logr.Logger, out io.Writer) *cobra.Command {
	o := &createOptions{
		logger: logger,
	}

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

			if o.intermediateDir == "" {
				intermediateDir, err := ioutil.TempDir("", "tmp-")
				if err != nil {
					logger.Error(err, "Error creating temporary directory")
					return err
				}
				logger.V(10).Info("Creating temporary directory", "path", intermediateDir)
				o.intermediateDir = intermediateDir
			}
			if o.enableIntermediateDirCleanup {
				defer os.RemoveAll(o.intermediateDir)
			}
			if err := o.prepare(); err != nil {
				return err
			}

			return o.run(out)
		},
	}

	cmd.Flags().StringVarP(&o.from, "from", "f", "", "TODO")
	cmd.MarkFlagRequired("from")
	cmd.Flags().StringVarP(&o.kubernetesSplitYamlCommand, "kubernetes-split-yaml-command", "k", "kubernetes-split-yaml", "kubernetes-split-yaml command (path to executable)")

	cmd.Flags().BoolVarP(&o.enableIntermediateDirCleanup, "cleanup", "c", false, "TODO")
	cmd.Flags().StringVarP(&o.intermediateDir, "intermediate-dir", "i", "", "TODO")
	cmd.Flags().StringVarP(&o.starter, "starter", "p", "", "the name or absolute path to Helm starter scaffold")
	return cmd
}

func (o *createOptions) prepare() error {
	e, err := os.Executable()
	if err != nil {
		o.logger.Error(err, "Error getting executable path")
		return err
	}

	path := filepath.Join(filepath.Dir(e), o.kubernetesSplitYamlCommand)
	_, err = exec.Command(path, "--outdir", o.intermediateDir, o.from).CombinedOutput()
	if err != nil {
		o.logger.Error(err, fmt.Sprintf("Error running %s", path))
		return err
	}
	return nil
}

func (o *createOptions) configPath() string {
	return filepath.Join(filepath.Dir(o.name), "kustohelmize.config")
}

func (o *createOptions) getConfig() (*cfg.Config, error) {
	chartname := filepath.Base(o.name)
	path := o.configPath()

	config := &cfg.Config{
		Chartname:     chartname,
		GlobalConfig:  *cfg.NewGlobalConfig(chartname),
		FileConfigMap: map[string]cfg.FileConfig{},
	}

	c, err := os.ReadDir(o.intermediateDir)
	for _, entry := range c {
		config.FileConfigMap[filepath.Join(o.intermediateDir, entry.Name())] = cfg.FileConfig{}
		o.logger.V(10).Info("Split YAML file", "name", entry.Name())
	}

	output, err := goyaml.Marshal(config)
	if err != nil {
		o.logger.Error(err, "Error marshalling config file")
		return nil, err
	}
	return config, ioutil.WriteFile(path, output, 0644)
}

func (o *createOptions) run(out io.Writer) error {
	o.logger.Info("Creating chart", "name", o.name)

	config, err := o.getConfig()
	if err != nil {
		o.logger.Error(err, "Error getting config")
		return err
	}

	chartname := filepath.Base(o.name)
	cdir := filepath.Dir(o.name)
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
		return chartutil.CreateFrom(cfile, cdir, lstarter)
	}

	chartutil.Stderr = out
	_, err = chartutil.Create(chartname, cdir)
	if err != nil {
		o.logger.Error(err, "Error creating chart", "name", o.name)
		return err
	}

	cdir = filepath.Join(cdir, chartname)
	files := []string{
		filepath.Join(cdir, chartutil.IngressFileName),
		filepath.Join(cdir, chartutil.DeploymentName),
		filepath.Join(cdir, chartutil.ServiceName),
		filepath.Join(cdir, chartutil.ServiceAccountName),
		filepath.Join(cdir, chartutil.HorizontalPodAutoscalerName),
		filepath.Join(cdir, chartutil.NotesName),
	}
	for _, file := range files {
		o.logger.V(10).Info("Removing file", "name", file)
		// Explicitly ignore errors here.
		os.Remove(file)
	}

	p := yaml.NewYAMLProcessor(o.logger, filepath.Join(cdir, chartutil.TemplatesDir), config)
	err = p.Process()
	if err != nil {
		o.logger.Error(err, "Error processing YAML")
		return err
	}

	// config := &config.Config{
	// 	Chartname:    chartname,
	// 	GlobalConfig: *config.NewGlobalConfig(chartname),
	// 	FileConfigMap: map[string]config.FileConfig{
	// 		filepath.Join(".", "test", "testdata", "generated", "mt-controller-manager-deployment.yaml"): {
	// 			config.XPath("spec.type"): {
	// 				Strategy: config.XPathStrategyInline,
	// 				Value:    "Values.xxx",
	// 			},
	// 			config.XPath("spec.selector"): {
	// 				Strategy: config.XPathStrategyNewline,
	// 				Value:    "Values.yyy",
	// 			},
	// 			config.XPath("spec.ports"): {
	// 				Strategy: config.XPathStrategyControlWith,
	// 				Value:    "Values.zzz",
	// 			},
	// 		},
	// 	},
	// }

	// p := yaml.NewYAMLProcessor(logger, file, config)
	// err = p.Process()
	// if err != nil {
	// 	logger.Errorf("Error processing YAML: %s", err)
	// 	return err
	// }
	return nil

	// TODO:

}
