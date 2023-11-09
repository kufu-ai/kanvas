package kanvas

import (
	"bytes"
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

	if len(components) == 0 {
		return fmt.Errorf("no components found")
	}

	if err := wf.load(path, baseDir, components); err != nil {
		return fmt.Errorf("loading %q %q: %w", path, baseDir, err)
	}

	plan, err := topologicalSort(wf.deps)
	if err != nil {
		return err
	}

	if len(components) > 0 && len(plan) == 0 {
		return fmt.Errorf("BUG: Unable to produce a valid plan even though there was no error")
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
	overrodeEnvs := map[string]struct{}{}

	for name, c := range config.Components {
		replacement, replaced := env.Uses[name]
		if replaced {
			if err := replacement.Validate(); err != nil {
				return nil, fmt.Errorf("environment %q: override for component %q: %w", wf.Options.Env, name, err)
			}

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

		overrides, overrode := env.Overrides[name]
		if overrode {
			overrodeEnvs[name] = struct{}{}
			if err := mergo.Merge(defaults, overrides, mergo.WithOverride); err != nil {
				return nil, fmt.Errorf("unable to override component %q: %w", name, err)
			}
		}

		if replaced && overrode {
			return nil, fmt.Errorf("component %q is both used and overridden. You can only use or override a component", name)
		}

		r[name] = c
	}

	for name := range env.Uses {
		if _, ok := usedEnvs[name]; !ok {
			return nil, fmt.Errorf("environment %q uses %q but it is not defined", wf.Options.Env, name)
		}
	}

	for name := range env.Overrides {
		if _, ok := overrodeEnvs[name]; !ok {
			return nil, fmt.Errorf("environment %q overrides %q but it is not defined", wf.Options.Env, name)
		}
	}

	return r, nil
}

func (wf *Workflow) load(path, baseDir string, components map[string]Component) error {
	const gitJob = "git"

	// "git" job is a special job that is always added to the workflow
	if _, ok := wf.WorkflowJobs[gitJob]; !ok {
		dir := baseDir
		driver := &Driver{
			Output: kanvasOutputCommandForID(gitJob),
			OutputFunc: func(r *Runtime, op Op, o map[string]string) error {
				var tag bytes.Buffer
				if err := r.Exec(dir, []string{"git", "tag", "--points-at", "HEAD"}, ExecStdout(&tag)); err != nil {
					return fmt.Errorf("unable to get current git tag: %w", err)
				}
				o["tag"] = strings.TrimSpace(tag.String())

				var sha bytes.Buffer
				if err := r.Exec(dir, []string{"git", "rev-parse", "HEAD"}, ExecStdout(&sha)); err != nil {
					return fmt.Errorf("unable to get current git sha: %w", err)
				}
				o["sha"] = strings.TrimSpace(sha.String())

				return nil
			},
		}

		wf.WorkflowJobs[gitJob] = &WorkflowJob{
			Dir:    dir,
			Driver: driver,
		}
		// This is to ensure that the git job is managed by the topological sorter
		wf.deps[gitJob] = []string{}
	}

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

		if len(c.Components) > 0 {
			if err := wf.load(subPath, dir, c.Components); err != nil {
				return err
			}
		}

		wf.deps[subPath] = needs
	}

	return nil
}
