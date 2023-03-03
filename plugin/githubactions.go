package plugin

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/goccy/go-yaml"
)

const (
	FormatGitHubActions = "githubactions"
)

func (e *Plugin) outputActionsWorkflows(target string) error {
	outputs := map[string]string{}
	if err := e.wf.WorkflowJobs[target].OutputFunc(e.r, outputs); err != nil {
		return fmt.Errorf("unable to process outputs for target %q: %w", target, err)
	}
	return nil
}

func (e *Plugin) exportActionsWorkflows(dir string) error {
	w := &actionsWorkflow{
		name: "Plan deployment",
		on: map[string]interface{}{
			"pull_request": map[string][]string{
				"branches":     {"main"},
				"paths-ignore": {"**.md", "**/docs/**"},
			},
		},
		jobs: make(map[string]actionsJob, len(e.wf.WorkflowJobs)),
	}

	outputs := map[string]map[string]string{}

	const step = "out"

	// Traverse the DAG of jobs
	for _, job := range e.wf.WorkflowJobs {
		job.Args.Visit(func(str string) {
		}, func(out string) {
			jobAndOutput := strings.SplitN(out, ".", 1)
			jobName := strings.ReplaceAll(jobAndOutput[0], "/", "-")
			if _, ok := outputs[jobName]; !ok {
				outputs[jobName] = map[string]string{}
			}
			outputs[jobName][jobAndOutput[1]] = fmt.Sprintf("${{ steps.%s.outputs.%s }}", step, jobAndOutput[1])
		})
	}

	for name, job := range e.wf.WorkflowJobs {
		name = strings.ReplaceAll(name, "/", "-")

		var needs []string
		for _, n := range job.Needs {
			needs = append(needs, strings.ReplaceAll(n, "/", "-"))
		}

		var cmd []string
		cmd = append(cmd, job.Diff...)
		job.Args.Visit(func(str string) {
		}, func(out string) {
			jobAndOutput := strings.SplitN(out, ".", 1)
			jobName := strings.ReplaceAll(jobAndOutput[0], "/", "-")
			cmd = append(cmd, fmt.Sprintf("${{ needs.%s.outputs.%s }}", jobName, jobAndOutput[1]))
		})

		j := &actionsJob{
			runsOn:  "ubuntu-latest",
			outputs: outputs[name],
			needs:   needs,
			steps: []actionsStep{
				stepCheckout(),
				stepRun("run", cmd),
				stepRun(name, job.Output(FormatGitHubActions)),
			},
		}

		w.AddJob(name, *j)
	}

	planYamlData, err := yaml.Marshal(w)
	if err != nil {
		return fmt.Errorf("unable to marshal plan workflow definition: %w", err)
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("unable to create directory %q: %w", dir, err)
	}

	if err := os.WriteFile(filepath.Join(dir, "plan_deployment.yaml"), planYamlData, 0644); err != nil {
		return fmt.Errorf("unable to write the plan workflow definition: %w", err)
	}

	return nil
}

type actionsWorkflow struct {
	name string                 `yaml:"name"`
	on   map[string]interface{} `yaml:"on"`
	jobs map[string]actionsJob  `yaml:"jobs"`
}

func (w *actionsWorkflow) AddJob(name string, def actionsJob) {
	w.jobs[name] = def
}

type actionsJob struct {
	needs   []string          `yaml:"needs,omitempty"`
	runsOn  string            `yaml:"runs_on"`
	outputs map[string]string `yaml:"outputs,omitempty"`
	steps   []actionsStep     `yaml:"steps"`
}

type actionsStep struct {
	id   string                 `yaml:"id,omitempty"`
	run  string                 `yaml:"run,omitempty"`
	uses string                 `yaml:"uses,omitempty"`
	with map[string]interface{} `yaml:"with,omitempty"`
}

func stepCheckout() actionsStep {
	return actionsStep{
		uses: "actions/checkout@v3",
		with: map[string]interface{}{
			"fetch-depth": 0,
		},
	}
}

func stepRun(id string, cmd []string) actionsStep {
	return actionsStep{
		id:  id,
		run: strings.Join(cmd, " "),
	}
}
