package interpreter

import (
	"fmt"
	"kanvas"
	"strings"

	"github.com/mumoshu/kargo"
)

type WorkflowJob struct {
	Outputs map[string]string
	Ran     bool

	*kanvas.WorkflowJob
}

type Interpreter struct {
	Workflow     *kanvas.Workflow
	WorkflowJobs map[string]*WorkflowJob
	runtime      *kanvas.Runtime
}

func New(wf *kanvas.Workflow, r *kanvas.Runtime) *Interpreter {
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
		runtime:      r,
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
		return fmt.Errorf("%q's dependencies %v: %w", name, job.Needs, err)
	}

	if err := f(job); err != nil {
		return fmt.Errorf("%q: %w", name, err)
	}

	return nil
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

func (p *Interpreter) runWithExtraArgs(j *WorkflowJob, steps []kanvas.Step) error {
	outputs := map[string]string{}
	for _, step := range steps {
		if step.IfOutputEq.Key != "" {
			if step.IfOutputEq.Value != outputs[step.IfOutputEq.Key] {
				continue
			}
		}

		for _, c := range step.Run {
			if err := p.runCmd(j, c); err != nil {
				return err
			}
		}

		if step.OutputFunc != nil {
			if err := step.OutputFunc(p.runtime, outputs); err != nil {
				return err
			}
		}
	}
	return nil
}

func (p *Interpreter) runCmd(j *WorkflowJob, cmd kargo.Cmd) error {
	args, err := cmd.Args.Collect(func(out string) (string, error) {
		jobOutput := strings.SplitN(out, ".", 1)
		jobName := jobOutput[0]
		outName := jobOutput[1]

		job, ok := p.WorkflowJobs[jobName]
		if !ok {
			return "", fmt.Errorf("job %q does not exist", jobName)
		}

		val, ok := job.Outputs[outName]
		if !ok {
			return "", fmt.Errorf("output %q does not exist", outName)
		}

		return val, nil
	})
	if err != nil {
		return fmt.Errorf("collecting args for command %q: %w", cmd.Name, err)
	}

	c := []string{cmd.Name}
	c = append(c, args...)

	dir := cmd.Dir
	if dir == "" {
		dir = j.Dir
	}

	if err := p.runtime.Exec(dir, c); err != nil {
		return fmt.Errorf("executing command %q in %s: %w", cmd.Name, dir, err)
	}

	return nil
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
