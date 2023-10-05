package kanvas

import (
	"fmt"
	"path/filepath"
	"strings"

	"dario.cat/mergo"
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

func NewWorkflow(config Component, opts Options) (*Workflow, error) {
	wf := &Workflow{
		WorkflowJobs: map[string]*WorkflowJob{},
		Dir:          config.Dir,
		deps:         make(map[string][]string),
		Options:      opts,
	}

	if err := wf.Load("", config.Dir, config); err != nil {
		return nil, err
	}

	return wf, nil
}

func (wf *Workflow) Load(path, baseDir string, config Component) error {
	components, err := wf.loadEnvironment(config)
	if err != nil {
		return err
	}

	if err := wf.load(path, baseDir, components); err != nil {
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

	// Replace the top-level components with the mid-level ones, if any.
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
	if len(midLevels) > 0 {
		plan[0] = midLevels
	}

	wf.Plan = plan

	return nil
}

func (wf *Workflow) loadEnvironment(config Component) (map[string]Component, error) {
	var env Environment
	if config.Environments != nil && wf.Options.Env != "" {
		var ok bool
		env, ok = config.Environments[wf.Options.Env]
		if !ok {
			return nil, fmt.Errorf("environment %q not found", wf.Options.Env)
		}
	}

	r := map[string]Component{}

	usedEnvs := map[string]struct{}{}

	for name, c := range config.Components {
		if replacement, ok := env.Uses[name]; ok {
			c = replacement

			usedEnvs[name] = struct{}{}
		}

		defaults, err := DeepCopyComponent(env.Defaults)
		if err != nil {
			return nil, err
		}

		if err := mergo.Merge(defaults, c, mergo.WithOverride); err != nil {
			return nil, err
		}

		r[name] = c
	}

	for name, _ := range env.Uses {
		if _, ok := usedEnvs[name]; !ok {
			return nil, fmt.Errorf("environment %q uses %q but it is not defined", wf.Options.Env, name)
		}
	}

	return r, nil
}

func (wf *Workflow) load(path, baseDir string, components map[string]Component) error {
	for name, c := range components {
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

		if err := wf.load(subPath, dir, c.Components); err != nil {
			return err
		}

		wf.deps[subPath] = needs
	}

	return nil
}
