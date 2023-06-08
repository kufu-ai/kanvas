package kanvas

import (
	"path/filepath"
	"strings"
)

type Workflow struct {
	Plan         [][]string
	WorkflowJobs map[string]*WorkflowJob
	Dir          string
	Options      Options

	deps map[string][]string
}

type WorkflowJob struct {
	Dir    string
	Needs  []string
	Driver *Driver
}

func NewWorkflow(workDir string, config Component, opts Options) (*Workflow, error) {
	wf := &Workflow{
		WorkflowJobs: map[string]*WorkflowJob{},
		Dir:          workDir,
		deps:         make(map[string][]string),
		Options:      opts,
	}

	if err := wf.Load("", workDir, config); err != nil {
		return nil, err
	}

	return wf, nil
}

func (wf *Workflow) Load(path, baseDir string, config Component) error {
	if err := wf.load(path, baseDir, config); err != nil {
		return err
	}

	plan, err := topologicalSort(wf.deps)
	if err != nil {
		return err
	}

	// Remove top-level components from the plan.
	// They are for logically grouping sub-components so
	// not needed to be executed.
	var midLevels []string
	for _, level := range plan[0] {
		if strings.Count(level, "/") < 2 {
			continue
		}
		midLevels = append(midLevels, level)
	}

	// Replace the top-level components with the mid-level ones.
	// The mid-level components are the ones that actually contain
	// "run" fields to be executed.
	// Top-level components are just for grouping hence they don't
	// contain "run" fields.
	//
	// For the test/e2e/testdata/kanvas.yaml example, the plan looks like the below:
	//
	// Before: plan = [][]string len: 4, cap: 4, [["/product1/appimage","product1"],["/product1/base"],["/product1/argocd"],["/product1/argocd_resources"]]
	// After : plan = [][]string len: 4, cap: 4, [["/product1/appimage"],["/product1/base"],["/product1/argocd"],["/product1/argocd_resources"]]
	//
	// Notice that the top-level component "product1" is removed.
	plan[0] = midLevels

	wf.Plan = plan

	return nil
}

func (wf *Workflow) load(path, baseDir string, config Component) error {
	for name, c := range config.Components {
		subPath := ID(path, name)

		var needs []string
		for _, n := range c.Needs {
			needs = append(needs, ID(path, n))
		}

		dir := c.Dir
		if dir == "" {
			dir = baseDir
		} else {
			if dir[0] == '/' {
				dir = filepath.Join(wf.Dir, dir)
			} else {
				dir = filepath.Join(baseDir, dir)
			}
		}

		driver, err := newDriver(subPath, dir, c, wf.Options)
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
