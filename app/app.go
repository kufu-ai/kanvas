package app

import (
	"fmt"
	"os"
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

func (a *App) Render(dir string) error {
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
