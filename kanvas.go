package kanvas

import (
	"fmt"
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
		return job.Diff()
	})
}

// Apply builds the container image(s) if any and runs terraform-apply command(s) to deploy changes
func (a *App) Apply() error {
	wf, err := newWorkflow(a.Config)
	if err != nil {
		return err
	}

	return wf.Run(func(job *WorkflowJob) error {
		return job.Apply()
	})
}

type Workflow struct {
	Entry        []string
	WorkflowJobs map[string]*WorkflowJob
}

func newWorkflow(config Component) (*Workflow, error) {
	wf := &Workflow{
		WorkflowJobs: map[string]*WorkflowJob{},
	}

	if err := wf.load("", config); err != nil {
		return nil, err
	}

	return wf, nil
}

func (wf *Workflow) load(path string, config Component) error {
	for name, p := range config.Components {
		subPath := id(path, name)

		var needs []string
		for _, n := range p.Needs {
			needs = append(needs, id(path, n))
		}

		wf.WorkflowJobs[subPath] = &WorkflowJob{
			Dir:   p.Dir,
			Needs: needs,
		}

		if err := wf.load(subPath, p); err != nil {
			return err
		}
	}
	return nil
}

func (wf *Workflow) Run(f func(job *WorkflowJob) error) error {
	return wf.parallel(wf.Entry, f)
}

func (wf *Workflow) run(name string, f func(job *WorkflowJob) error) error {
	job, ok := wf.WorkflowJobs[name]
	if !ok {
		return fmt.Errorf("component %q is not defined", name)
	}

	if err := wf.parallel(job.Needs, f); err != nil {
		return err
	}

	return f(job)
}

func (wf *Workflow) parallel(names []string, f func(job *WorkflowJob) error) error {
	var (
		errs  []error
		errCh = make(chan error)
	)

	for _, n := range names {
		go func() {
			errCh <- wf.run(n, f)
		}()
	}

	for i := 0; i < len(names); i++ {
		errs = append(errs, <-errCh)
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed resolving dependencies: %v", errs)
	}

	return nil
}

type WorkflowJob struct {
	Dir     string
	Needs   []string
	Outputs map[string]string
	Ran     bool
}

func (j *WorkflowJob) Diff() error {
	if j.Ran {
		return nil
	}

	j.Ran = true

	return nil
}

func (j *WorkflowJob) Apply() error {
	if j.Ran {
		return nil
	}

	j.Ran = true

	return nil
}
