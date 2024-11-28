package cmd

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/go-logr/logr"
	"github.com/spf13/cobra"

	"github.com/yeahdongcn/kustohelmize/internal/third_party/dep/fs"
	cfg "github.com/yeahdongcn/kustohelmize/pkg/config"
	"github.com/yeahdongcn/kustohelmize/pkg/template"
	"github.com/yeahdongcn/kustohelmize/pkg/value"
	"gopkg.in/yaml.v1"
	"helm.sh/helm/v3/cmd/helm/require"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/helmpath"
)

const (
	// Helm Chart.yaml defaults
	HELM_DEFAULT_CHART_VERSION = "version: 0.1.0"
	HELM_DEFAULT_APP_VERSION   = "appVersion: \"1.16.0\""
	HELM_DEFAULT_DESCRIPTION   = "description: A Helm chart for Kubernetes"
)

type createOptions struct {
	options

	version     string
	appVersion  string
	description string

	from                       string
	kubernetesSplitYamlCommand string
	suppressNamespace          bool

	// From helm.
	starter    string // --starter
	starterDir string
}

func newCreateCmd(logger logr.Logger, out io.Writer) *cobra.Command {
	o := &createOptions{
		options: options{
			logger: logger.WithName("create"),
		},
	}

	cmd := &cobra.Command{
		Use:   "create NAME",
		Short: "Create a new Helm chart",
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

	cmd.Flags().StringVarP(&o.version, "version", "v", "", "A SemVer 2 conformant version string of the chart")
	cmd.Flags().StringVarP(&o.appVersion, "app-version", "a", "", "The version of the application enclosed inside of this chart")
	cmd.Flags().StringVarP(&o.description, "description", "d", "", "A one-sentence description of the chart")

	cmd.Flags().StringVarP(&o.from, "from", "f", "", "The path to a Kubernetes manifest YAML file")
	cmd.MarkFlagRequired("from")
	cmd.Flags().StringVarP(&o.kubernetesSplitYamlCommand, "kubernetes-split-yaml-command", "k", "kubernetes-split-yaml", "Command to split Kubernetes YAML")
	cmd.Flags().BoolVarP(&o.suppressNamespace, "suppress-namespace", "s", false, "Suppress creation of namespace resource, which Kustomize will emit. RBAC bindings for SAs will be to {{ .Release.Namespace }}")
	cmd.Flags().StringVarP(&o.intermediateDir, "intermediate-dir", "i", "", "The path to a intermediate directory")
	cmd.Flags().MarkHidden("intermediate-dir")
	cmd.Flags().BoolVarP(&o.enableIntermediateDirCleanup, "cleanup", "", false, "Whether to cleanup the intermediate directory")
	cmd.Flags().MarkHidden("cleanup")
	cmd.Flags().StringVarP(&o.config, "config", "c", "", "The path to a config file")
	cmd.Flags().MarkHidden("config")

	cmd.Flags().StringVarP(&o.starter, "starter", "p", "", "The name or absolute path to Helm starter scaffold")
	return cmd
}

func (o *createOptions) updateConfig(config *cfg.ChartConfig, forceSave bool) error {
	shouldSave := false

	c, _ := os.ReadDir(o.intermediateDir)
	for _, entry := range c {
		if _, ok := config.FileConfig[filepath.Join(o.intermediateDir, entry.Name())]; !ok {
			config.FileConfig[filepath.Join(o.intermediateDir, entry.Name())] = cfg.Config{}
			if !forceSave {
				shouldSave = true
			}
		}
	}

	if shouldSave || forceSave {
		output, err := yaml.Marshal(config)
		if err != nil {
			o.logger.Error(err, "Error marshalling config file")
			return err
		}
		return os.WriteFile(o.configPath(), output, 0644)
	}

	return nil
}

func (o *createOptions) getConfig() (*cfg.ChartConfig, error) {
	path := o.configPath()
	logger := o.logger.WithName("config")
	_, err := os.Stat(path)
	if err == nil {
		o.logger.Info("Config file already exists", "path", path)

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
		err = o.updateConfig(config, false)
		if err != nil {
			o.logger.Error(err, "Error updating config file", "path", path)
			return nil, err
		}
		err = config.Validate()
		if err != nil {
			o.logger.Error(err, "Error validating config file", "path", path)
			return nil, err
		}
		return config, nil
	}

	chartname := o.chartname()
	config := cfg.NewChartConfig(logger, chartname)
	err = o.updateConfig(config, true)
	return config, err
}

func (o *createOptions) prepare() error {
	var path string

	if fs.IsAbsolutePath(o.kubernetesSplitYamlCommand) {
		// Path from -k is absolute
		path = o.kubernetesSplitYamlCommand
	} else {
		// Path is relative to location of the running exe
		e, err := os.Executable()
		if err != nil {
			o.logger.Error(err, "Error getting executable path")
			return err
		}

		path = filepath.Join(filepath.Dir(e), o.kubernetesSplitYamlCommand)
	}
	_, err := exec.Command(path, "--outdir", o.intermediateDir, o.from).CombinedOutput()
	if err != nil {
		o.logger.Error(err, fmt.Sprintf("Error running %s", path))
		return err
	}
	return nil
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

	chartdir := filepath.Join(chartroot, chartname)

	// If _helpers.tpl exists, back it up
	var helpers []byte = nil
	helperFile := filepath.Join(chartdir, chartutil.HelpersName)
	f, err := os.Open(helperFile)
	if err == nil {
		helpers, _ = io.ReadAll(f)
		f.Close()

		defer func() {
			err := os.WriteFile(helperFile, helpers, 0664)
			if err != nil {
				o.logger.Error(err, "Error restoring _helpers.tpl")
			}
		}()
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
	ins, err := os.ReadFile(chartfile)
	if err != nil {
		o.logger.Error(err, "Error reading chartfile")
		return err
	}
	scanner := bufio.NewScanner(bytes.NewReader(ins))
	var sb strings.Builder

	for scanner.Scan() {
		text := scanner.Text()
		if o.version != "" && text == HELM_DEFAULT_CHART_VERSION {
			fmt.Fprintf(&sb, "version: %s\n", o.version)
		} else if o.appVersion != "" && text == HELM_DEFAULT_APP_VERSION {
			fmt.Fprintf(&sb, "appVersion: \"%s\"\n", o.appVersion)
		} else if o.description != "" && text == HELM_DEFAULT_DESCRIPTION {
			fmt.Fprintf(&sb, "description: %s\n", o.description)
		} else {
			fmt.Fprintln(&sb, text)
		}
	}
	err = os.WriteFile(chartfile, []byte(sb.String()), 0644)
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

	p := template.NewProcessor().
		WithLogger(o.logger.WithName("template")).
		WithChartConfig(config).
		WithTemplatesDir(filepath.Join(chartdir, chartutil.TemplatesDir)).
		WithCrdsDir(filepath.Join(chartdir, "crds")).
		WithSuppressNamespace(o.suppressNamespace)

	err = p.Process()
	if err != nil {
		o.logger.Error(err, "Error processing templates")
		return err
	}

	return nil
}
