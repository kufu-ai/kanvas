package interpreter

import (
	"context"
	"fmt"
	"kanvas"
	"os/exec"
)

type WorkflowJob struct {
	Outputs map[string]string
	Ran     bool

	*kanvas.WorkflowJob
}

type Interpreter struct {
	Workflow     *kanvas.Workflow
	WorkflowJobs map[string]*WorkflowJob
}

func New(wf *kanvas.Workflow) *Interpreter {
	wjs := map[string]*WorkflowJob{}
	for k, v := range wf.WorkflowJobs {
		v := v
		wjs[k] = &WorkflowJob{
			Outputs:     make(map[string]string),
			WorkflowJob: v,
		}
	}

	return &Interpreter{
		Workflow:     wf,
		WorkflowJobs: wjs,
	}
}

func (p *Interpreter) Run(f func(job *WorkflowJob) error) error {
	return p.parallel(p.Workflow.Entry, f)
}

func (p *Interpreter) run(name string, f func(job *WorkflowJob) error) error {
	job, ok := p.WorkflowJobs[name]
	if !ok {
		return fmt.Errorf("component %q is not defined", name)
	}

	if err := p.parallel(job.Needs, f); err != nil {
		return err
	}

	return f(job)
}

func (p *Interpreter) parallel(names []string, f func(job *WorkflowJob) error) error {
	var (
		errs  []error
		errCh = make(chan error)
	)

	for _, n := range names {
		n := n
		go func() {
			errCh <- p.run(n, f)
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

func (p *Interpreter) exec(dir string, cmd []string) error {
	c := exec.CommandContext(context.TODO(), cmd[0], cmd[1:]...)
	c.Dir = dir
	return c.Run()
}

func (p *Interpreter) runWithExtraArgs(j *WorkflowJob, cmd []string) error {
	var c []string

	c = append(c, cmd...)

	j.Driver.Args.Visit(func(str string) {
		c = append(c, str)
	}, func(out string) {
		// TODO Add the referenced output as an arg
	})

	return p.exec(j.Dir, c)
}

func (p *Interpreter) Apply() error {
	return p.Run(func(job *WorkflowJob) error {
		return p.applyJob(job)
	})
}

func (p *Interpreter) Diff() error {
	return p.Run(func(job *WorkflowJob) error {
		return p.diffJob(job)
	})
}

func (p *Interpreter) diffJob(j *WorkflowJob) error {
	if j.Ran {
		return nil
	}

	if err := p.runWithExtraArgs(j, j.Driver.Diff); err != nil {
		return err
	}

	j.Ran = true

	return nil
}

func (p *Interpreter) applyJob(j *WorkflowJob) error {
	if j.Ran {
		return nil
	}

	if err := p.runWithExtraArgs(j, j.Driver.Apply); err != nil {
		return err
	}

	j.Ran = true

	return nil
}
