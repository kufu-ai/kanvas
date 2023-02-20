package kanvas

import (
	"fmt"
	"os"
	"path/filepath"
)

type Workflow struct {
	Entry        []string
	WorkflowJobs map[string]*WorkflowJob
	Dir          string
}

func newWorkflow(config Component) (*Workflow, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	wf := &Workflow{
		WorkflowJobs: map[string]*WorkflowJob{},
		Dir:          dir,
	}

	if err := wf.load("", dir, config); err != nil {
		return nil, err
	}

	return wf, nil
}

func (wf *Workflow) load(path, baseDir string, config Component) error {
	for name, c := range config.Components {
		subPath := id(path, name)

		var needs []string
		for _, n := range c.Needs {
			needs = append(needs, id(path, n))
		}

		dir := c.Dir
		if dir[0] != '/' {
			dir = filepath.Join(baseDir, dir)
		}

		wf.WorkflowJobs[subPath] = &WorkflowJob{
			Dir:   dir,
			Needs: needs,
		}

		if err := wf.load(subPath, dir, c); err != nil {
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
