package app

import (
	"fmt"
	"kanvas"
	"kanvas/interpreter"
	"kanvas/plugin"
)

type App struct {
	Config  kanvas.Component
	Runtime *kanvas.Runtime
	Options kanvas.Options
}

func New(opts kanvas.Options) (*App, error) {
	c, err := kanvas.LoadConfig(opts)
	if err != nil {
		return nil, err
	}

	r := kanvas.NewRuntime()

	return &App{
		Config:  *c,
		Runtime: r,
		Options: opts,
	}, nil

}

func (a *App) newWorkflow() (*kanvas.Workflow, error) {
	return kanvas.NewWorkflow(a.Config, a.Options)
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
