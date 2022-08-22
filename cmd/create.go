package cmd

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/go-logr/logr"
	"github.com/spf13/cobra"
	"github.com/yeahdongcn/kustohelmize/pkg/config"
	cfg "github.com/yeahdongcn/kustohelmize/pkg/config"
	"github.com/yeahdongcn/kustohelmize/pkg/template"
	"github.com/yeahdongcn/kustohelmize/pkg/value"
	"gopkg.in/yaml.v1"
	"helm.sh/helm/v3/cmd/helm/require"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/helmpath"
)

type createOptions struct {
	logger logr.Logger

	version                      string
	from                         string
	kubernetesSplitYamlCommand   string
	intermediateDir              string
	enableIntermediateDirCleanup bool
	config                       string

	// From helm.
	starter    string // --starter
	name       string
	starterDir string
}

func newCreateCmd(logger logr.Logger, out io.Writer) *cobra.Command {
	o := &createOptions{
		logger: logger.WithName("create"),
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
				o.intermediateDir = fmt.Sprintf("%s-%s", o.name, "generated")
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

	cmd.Flags().StringVarP(&o.version, "version", "v", "", "A SemVer 2 conformant version string of the chart.")
	cmd.Flags().StringVarP(&o.from, "from", "f", "", "TODO")
	cmd.MarkFlagRequired("from")
	cmd.Flags().StringVarP(&o.kubernetesSplitYamlCommand, "kubernetes-split-yaml-command", "k", "kubernetes-split-yaml", "kubernetes-split-yaml command (path to executable)")
	cmd.Flags().StringVarP(&o.intermediateDir, "intermediate-dir", "i", "", "TODO")
	cmd.Flags().MarkHidden("intermediate-dir")
	cmd.Flags().BoolVarP(&o.enableIntermediateDirCleanup, "cleanup", "", false, "TODO")
	cmd.Flags().MarkHidden("cleanup")
	cmd.Flags().StringVarP(&o.config, "config", "c", "", "TODO")
	cmd.Flags().MarkHidden("config")

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

func (o *createOptions) chartroot() string {
	return filepath.Dir(o.name)
}

func (o *createOptions) chartname() string {
	return filepath.Base(o.name)
}

func (o *createOptions) configPath() string {
	return filepath.Join(filepath.Dir(o.name), fmt.Sprintf("%s.config", o.chartname()))
}

func (o *createOptions) getConfig() (*cfg.ChartConfig, error) {
	path := o.configPath()
	logger := o.logger.WithName("config")
	_, err := os.Stat(path)
	if err == nil {
		o.logger.Info("Config file already exists", "path", path)

		out, err := ioutil.ReadFile(path)
		if err != nil {
			o.logger.Error(err, "Error reading config file", "path", path)
			return nil, err
		}
		config := &config.ChartConfig{Logger: logger}
		err = yaml.Unmarshal(out, config)
		if err != nil {
			o.logger.Error(err, "Error unmarshalling config file", "path", path)
			return nil, err
		}
		return config, nil
	}

	chartname := o.chartname()
	config := cfg.NewChartConfig(logger, chartname)

	c, err := os.ReadDir(o.intermediateDir)
	for _, entry := range c {
		config.FileConfig[filepath.Join(o.intermediateDir, entry.Name())] = cfg.Config{}
	}

	output, err := yaml.Marshal(config)
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

	chartname := o.chartname()
	chartroot := o.chartroot()
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
		return chartutil.CreateFrom(cfile, chartroot, lstarter)
	}

	chartutil.Stderr = out
	_, err = chartutil.Create(chartname, chartroot)
	if err != nil {
		o.logger.Error(err, "Error creating chart", "name", o.name)
		return err
	}

	chartdir := filepath.Join(chartroot, chartname)
	files := []string{
		filepath.Join(chartdir, chartutil.IngressFileName),
		filepath.Join(chartdir, chartutil.DeploymentName),
		filepath.Join(chartdir, chartutil.ServiceName),
		filepath.Join(chartdir, chartutil.ServiceAccountName),
		filepath.Join(chartdir, chartutil.HorizontalPodAutoscalerName),
		filepath.Join(chartdir, chartutil.NotesName),

		filepath.Join(chartdir, chartutil.TestConnectionName),

		filepath.Join(chartdir, chartutil.ValuesfileName),
	}
	for _, file := range files {
		o.logger.V(10).Info("Removing file", "name", file)
		// Explicitly ignore errors here.
		os.Remove(file)
	}

	chartfile := filepath.Join(chartdir, chartutil.ChartfileName)
	ins, err := ioutil.ReadFile(chartfile)
	if err != nil {
		o.logger.Error(err, "Error reading chartfile")
		return err
	}
	scanner := bufio.NewScanner(bytes.NewReader(ins))
	outs := ""
	for scanner.Scan() {
		if o.version != "" && scanner.Text() == "version: 0.1.0" {
			outs += fmt.Sprintf("version: %s\n", o.version)
		} else {
			outs += scanner.Text() + "\n"
		}
	}
	err = ioutil.WriteFile(chartfile, []byte(outs), 0644)
	if err != nil {
		o.logger.Error(err, "Error writing chartfile")
		return err
	}

	v := value.NewProcessor(o.logger.WithName("value"), config, chartdir)
	err = v.Process()
	if err != nil {
		o.logger.Error(err, "Error processing values")
		return err
	}

	p := template.NewProcessor(o.logger.WithName("template"), config, filepath.Join(chartdir, chartutil.TemplatesDir), filepath.Join(chartdir, "crds"))
	err = p.Process()
	if err != nil {
		o.logger.Error(err, "Error processing templates")
		return err
	}

	return nil
}
