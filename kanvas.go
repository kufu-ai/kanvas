package kanvas

import (
	"os"

	"github.com/goccy/go-yaml"
)

type Component struct {
	Dir        string               `yaml:"dir"`
	Components map[string]Component `yaml:"components"`
	Needs      []string             `yaml:"needs"`
	Docker     *Docker              `yaml:"docker,omitempty"`
	Terraform  *Terraform           `yaml:"terraform,omitempty"`
}

type Docker struct {
	Image string `yaml:"image"`
	File  string `yaml:"file"`
}

type Terraform struct {
	Target string `yaml:"target"`
	Vars   []Var  `yaml:"var"`
}

type Var struct {
	Name      string `yaml:"name"`
	ValueFrom string `yaml:"valueFrom"`
}

type App struct {
	Config Component
}

func New(path string) (*App, error) {
	var (
		config Component
	)

	file, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	if err := yaml.Unmarshal(file, &config); err != nil {
		return nil, err
	}

	return &App{
		Config: config,
	}, nil
}

// Diff shows the diff between the desired state and the current state
func (a *App) Diff() error {
	wf, err := newWorkflow(a.Config)
	if err != nil {
		return err
	}

	return wf.Run(func(job *WorkflowJob) error {
		return job.Diff(wf)
	})
}

// Apply builds the container image(s) if any and runs terraform-apply command(s) to deploy changes
func (a *App) Apply() error {
	wf, err := newWorkflow(a.Config)
	if err != nil {
		return err
	}

	return wf.Run(func(job *WorkflowJob) error {
		return job.Apply(wf)
	})
}
