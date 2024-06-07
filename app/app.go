package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/goccy/go-yaml"

	"github.com/davinci-std/kanvas/plugin"

	"github.com/davinci-std/kanvas/interpreter"

	"github.com/davinci-std/kanvas"
)

type App struct {
	Config  Config
	Runtime *kanvas.Runtime
	Options kanvas.Options
}

type Config struct {
	Raw  []byte
	Path string
	kanvas.Component
}

func New(opts kanvas.Options) (*App, error) {
	// ts is a timestamp in the format of YYYYMMDDHHMMSS
	now := time.Now()
	ts := now.Format("20060102150405")
	tempDir, err := os.MkdirTemp("", fmt.Sprintf("kanvas_%s_*", ts))
	if err != nil {
		return nil, fmt.Errorf("unable to create temp dir: %w", err)
	}
	opts.TempDir = tempDir

	path, err := kanvas.DiscoverConfigFile(opts)
	if err != nil {
		return nil, err
	}

	file, err := kanvas.RenderOrReadFile(path)
	if err != nil {
		return nil, err
	}

	c, err := kanvas.LoadConfig(path, file)
	if err != nil {
		return nil, err
	}

	r := kanvas.NewRuntime()

	return &App{
		Config: Config{
			Raw:       file,
			Path:      path,
			Component: *c,
		},
		Runtime: r,
		Options: opts,
	}, nil
}

func (a *App) newWorkflow() (*kanvas.Workflow, error) {
	return kanvas.NewWorkflow(a.Config.Component, a.Options)
}

// Diff shows the diff between the desired state and the current state
func (a *App) Diff() error {
	wf, err := a.newWorkflow()
	if err != nil {
		return err
	}

	p := interpreter.New(wf, a.Runtime)

	return p.Diff()
}

// Apply builds the container image(s) if any and runs terraform-apply command(s) to deploy changes
func (a *App) Apply() error {
	wf, err := a.newWorkflow()
	if err != nil {
		return err
	}

	p := interpreter.New(wf, a.Runtime)

	return p.Apply()
}

func (a *App) Export(format, dir, kanvasContainerImage string) error {
	wf, err := a.newWorkflow()
	if err != nil {
		return err
	}

	e := plugin.New(wf, a.Runtime)

	return e.Export(format, dir, kanvasContainerImage)
}

// RenderConfig contains the configuration for rendering the kanvas.yaml file
type RenderConfig struct {
	// Push is a flag to push the rendered kanvasa.yaml to the git repository
	// If true, the rendered kanvas.yaml is pushed to the `origin` remote repository.
	// In case you want to push to a different branch than the current branch,
	// you must run `git checkout -b <branch>` before running `kanvas render`.
	Push bool
}

type RenderOption func(*RenderConfig)

// Render renders the kanvas.template.jsonnet to kanvas.yaml under the specified
// directory.
// If dir is empty, the rendered file is written to the current directory.
func (a *App) Render(dir string, opts ...RenderOption) error {
	path := filepath.Base(a.Config.Path)

	const ext = ".template.jsonnet"
	if strings.HasSuffix(path, ext) {
		path = path[:len(path)-len(ext)]
	} else {
		path = path[:len(path)-len(filepath.Ext(path))]
	}
	path = filepath.Join(dir, path+".yaml")

	yamlData, err := yaml.JSONToYAML(a.Config.Raw)
	if err != nil {
		return fmt.Errorf("unable to convert json to yaml: %w", err)
	}

	if err := os.WriteFile(path, yamlData, 0644); err != nil {
		return fmt.Errorf("unable to write to %s: %w", path, err)
	}

	var cfg RenderConfig
	for _, o := range opts {
		o(&cfg)
	}

	if cfg.Push {
		gitAdd := fmt.Sprintf("git add %s", path)
		gitCommit := fmt.Sprintf("git commit -m 'Render %s'", path)
		gitPush := "git push origin HEAD"

		if err := a.runCommands(gitAdd, gitCommit, gitPush); err != nil {
			return err
		}

		fmt.Printf("Rendered %s and pushed to the git repository\n", path)
	} else {
		fmt.Printf("Rendered %s\n", path)
	}

	return nil
}

func (a *App) runCommands(commands ...string) error {
	for _, c := range commands {
		fmt.Printf("⌛️ Running command: %s\n", c)

		bashCmd := exec.Command("bash", "-c", c)
		bashCmd.Stdout = os.Stdout
		bashCmd.Stderr = os.Stderr

		if err := bashCmd.Run(); err != nil {
			return fmt.Errorf("unable to run command %q: %w", c, err)
		}
	}

	return nil
}

func (a *App) Output(format, op, target string) error {
	wf, err := a.newWorkflow()
	if err != nil {
		return err
	}

	e := plugin.New(wf, a.Runtime)

	var o kanvas.Op
	switch op {
	case "diff":
		o = kanvas.Diff
	case "apply":
		o = kanvas.Apply
	default:
		return fmt.Errorf("unsupported op %q", op)
	}

	return e.Output(o, format, target)
}
