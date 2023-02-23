package app

import (
	"kanvas"
	"kanvas/exporter"
	"kanvas/interpreter"
)

type App struct {
	Config  kanvas.Component
	Runtime *kanvas.Runtime
}

func New(path string) (*App, error) {
	c, err := kanvas.New(path)
	if err != nil {
		return nil, err
	}

	r := kanvas.NewRuntime()

	return &App{
		Config:  *c,
		Runtime: r,
	}, nil

}

// Diff shows the diff between the desired state and the current state
func (a *App) Diff() error {
	wf, err := kanvas.NewWorkflow(a.Config)
	if err != nil {
		return err
	}

	p := interpreter.New(wf, a.Runtime)

	return p.Diff()
}

// Apply builds the container image(s) if any and runs terraform-apply command(s) to deploy changes
func (a *App) Apply() error {
	wf, err := kanvas.NewWorkflow(a.Config)
	if err != nil {
		return err
	}

	p := interpreter.New(wf, a.Runtime)

	return p.Apply()
}

func (a *App) Export(format, dir string) error {
	wf, err := kanvas.NewWorkflow(a.Config)
	if err != nil {
		return err
	}

	e := exporter.New(wf, a.Runtime)

	return e.Export(format, dir)
}

func (a *App) Output(format, target string) error {
	wf, err := kanvas.NewWorkflow(a.Config)
	if err != nil {
		return err
	}

	e := exporter.New(wf, a.Runtime)

	return e.Output(format, target)
}
