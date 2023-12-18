package interpreter

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/davinci-std/kanvas"

	"github.com/hashicorp/go-multierror"
	"github.com/mumoshu/kargo"
)

type WorkflowJob struct {
	ID      string
	Outputs map[string]string
	Ran     bool

	*kanvas.WorkflowJob
}

type Interpreter struct {
	Workflow     *kanvas.Workflow
	WorkflowJobs map[string]*WorkflowJob
	runtime      *kanvas.Runtime

	EnableParallel bool
}

func New(wf *kanvas.Workflow, r *kanvas.Runtime) *Interpreter {
	wjs := map[string]*WorkflowJob{}
	for k, v := range wf.WorkflowJobs {
		v := v
		wjs[k] = &WorkflowJob{
			ID:          k,
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
	for _, phase := range p.Workflow.Plan {
		if err := p.parallel(phase, f); err != nil {
			return err
		}
	}

	return nil
}

func (p *Interpreter) run(name string, f func(job *WorkflowJob) error) error {
	job, ok := p.WorkflowJobs[name]
	if !ok {
		return fmt.Errorf("component %q is not defined", name)
	}

	if job.Skipped != nil {
		job.Outputs = job.Skipped
		return nil
	}

	if err := f(job); err != nil {
		return fmt.Errorf("component %q: %w", name, err)
	}

	return nil
}

func (p *Interpreter) parallel(names []string, f func(job *WorkflowJob) error) error {
	var (
		errs  error
		errCh = make(chan error, len(names))
	)

	for _, n := range names {
		n := n
		if p.EnableParallel {
			go func() {
				errCh <- p.run(n, f)
			}()
		} else {
			errCh <- p.run(n, f)
		}
	}

	for i := 0; i < len(names); i++ {
		if err := <-errCh; err != nil {
			errs = multierror.Append(errs, err)
		}
	}

	if errs != nil {
		return fmt.Errorf("failed running component group %v: %s", names, errs)
	}

	return nil
}

func (p *Interpreter) runWithExtraArgs(j *WorkflowJob, op kanvas.Op, steps []kanvas.Step) error {
	outputs := map[string]string{}
	for _, step := range steps {
		if step.IfOutputEq.Key != "" {
			if step.IfOutputEq.Value != outputs[step.IfOutputEq.Key] {
				continue
			}
		}

		for _, c := range step.Run {
			if err := p.runCmd(j, c); err != nil {
				return fmt.Errorf("command %s: %w", c, err)
			}
		}

		if step.OutputFunc != nil {
			if err := step.OutputFunc(p.runtime, outputs); err != nil {
				return err
			}
		}
	}

	if j.Driver.OutputFunc != nil {
		if err := j.Driver.OutputFunc(p.runtime, op, outputs); err != nil {
			return err
		}
	}

	j.Outputs = outputs

	return nil
}

func (p *Interpreter) runCmd(j *WorkflowJob, cmd kargo.Cmd) error {
	args, err := cmd.Args.Collect(func(out string) (string, error) {
		jobOutput := strings.SplitN(out, ".", 2)
		if len(jobOutput) != 2 {
			return "", fmt.Errorf("could not find dot(.) within %q", out)
		}
		jobName := jobOutput[0]
		outName := jobOutput[1]

		fullJobName := kanvas.SiblingID(j.ID, jobName)

		job, ok := p.WorkflowJobs[fullJobName]
		if !ok {
			return "", fmt.Errorf("job %q does not exist", jobName)
		}

		val, ok := job.Outputs[outName]
		if !ok {
			var debug string
			if os.Getenv("DEBUG") == "1" {
				debug = fmt.Sprintf(". Available outputs: %v", job.Outputs)
			} else {
				debug = ". Set DEBUG=1 to see all the outputs"
			}
			return "", fmt.Errorf(`output "%s.%s" does not exist. Ensure that %q outputs %q%s`, jobName, outName, jobName, outName, debug)
		}

		return val, nil
	})
	if err != nil {
		return fmt.Errorf("while collecting args for command %q: %w", cmd.Name, err)
	}

	c := []string{cmd.Name}
	c = append(c, args...)

	dir := cmd.Dir
	if dir == "" {
		dir = j.Dir
	}

	var opts []kanvas.ExecOption
	if len(cmd.AddEnv) > 0 {
		opts = append(opts, kanvas.ExecAddEnv(cmd.AddEnv))
	}

	if err := p.runtime.Exec(dir, c, opts...); err != nil {
		return fmt.Errorf("command %q: %w", cmd.Name, err)
	}

	return nil
}

func (p *Interpreter) Apply() error {
	if err := p.Run(func(job *WorkflowJob) error {
		if err := p.applyJob(job); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return err
	}

	// An example of the whole kanvas standard output would look like:
	// {
	//   "app": {
	//     "pullRequest.head": "kanvas-20231216082910",
	//     "pullRequest.htmlURL": "https://github.com/myorg/mygitopsconfig/pull/123",
	//     "pullRequest.id": "1234567890",
	//     "pullRequest.number": "123"
	//   },
	//   "git": {
	//     "sha": "123d3c1a9a669f45c17a25cf5222c0cc0b630738",
	//     "tag": ""
	//   },
	//   "image": {
	//     "id": "sha256:12356935acbd6a67d3fed10512d89450330047f3ae6fb3a62e9bf4f229529387",
	//     "kanvas.buildx": "true"
	//   }
	// }

	out := map[string]map[string]string{}

	for _, job := range p.WorkflowJobs {
		out[job.ID] = job.Outputs
	}

	res, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling outputs: %w", err)
	}

	fmt.Fprintf(os.Stdout, "%s\n", res)

	return nil
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

	if err := p.runWithExtraArgs(j, kanvas.Diff, j.Driver.Diff); err != nil {
		return err
	}

	j.Ran = true

	return nil
}

func (p *Interpreter) applyJob(j *WorkflowJob) error {
	if j.Ran {
		return nil
	}

	if err := p.runWithExtraArgs(j, kanvas.Apply, j.Driver.Apply); err != nil {
		return err
	}

	j.Ran = true

	return nil
}
