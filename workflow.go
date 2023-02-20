package kanvas

import "fmt"

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
