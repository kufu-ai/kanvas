package app

import (
	"kanvas"
	"kanvas/interpreter"
)

type App struct {
	Config kanvas.Component
}

func New(path string) (*App, error) {
	c, err := kanvas.New(path)
	if err != nil {
		return nil, err
	}

	return &App{
		Config: *c,
	}, nil

}

// Diff shows the diff between the desired state and the current state
func (a *App) Diff() error {
	wf, err := kanvas.NewWorkflow(a.Config)
	if err != nil {
		return err
	}

	p := interpreter.New(wf)

	return p.Diff()
}

// Apply builds the container image(s) if any and runs terraform-apply command(s) to deploy changes
func (a *App) Apply() error {
	wf, err := kanvas.NewWorkflow(a.Config)
	if err != nil {
		return err
	}

	p := interpreter.New(wf)

	return p.Apply()
}
