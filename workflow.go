package kanvas

import (
	"os"
	"path/filepath"
)

type Workflow struct {
	Entry        []string
	WorkflowJobs map[string]*WorkflowJob
	Dir          string

	deps map[string][]string
}

type WorkflowJob struct {
	Dir    string
	Needs  []string
	Driver *Driver
}

func NewWorkflow(config Component) (*Workflow, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	wf := &Workflow{
		WorkflowJobs: map[string]*WorkflowJob{},
		Dir:          dir,
		deps:         make(map[string][]string),
	}

	if err := wf.Load("", dir, config); err != nil {
		return nil, err
	}

	return wf, nil
}

func (wf *Workflow) Load(path, baseDir string, config Component) error {
	if err := wf.load(path, baseDir, config); err != nil {
		return err
	}

	levels, err := topologicalSort(wf.deps)
	if err != nil {
		return err
	}
	wf.Entry = levels[0]

	return nil
}

func (wf *Workflow) load(path, baseDir string, config Component) error {
	for name, c := range config.Components {
		subPath := id(path, name)

		var needs []string
		for _, n := range c.Needs {
			needs = append(needs, id(path, n))
		}

		dir := c.Dir
		if dir == "" {
			dir = name
		}
		if dir[0] != '/' {
			dir = filepath.Join(baseDir, dir)
		}

		driver, err := newDriver(subPath, dir, c)
		if err != nil {
			return err
		}

		wf.WorkflowJobs[subPath] = &WorkflowJob{
			Dir:    dir,
			Needs:  needs,
			Driver: driver,
		}

		if err := wf.load(subPath, dir, c); err != nil {
			return err
		}

		wf.deps[subPath] = needs
	}

	return nil
}
