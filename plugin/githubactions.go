package plugin

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/goccy/go-yaml"
)

const (
	FormatGitHubActions = "githubactions"

	FormatDefault = FormatGitHubActions
)

func (e *Plugin) outputActionsWorkflows(target string) error {
	outputs := map[string]string{}
	if err := e.wf.WorkflowJobs[target].Driver.OutputFunc(e.r, outputs); err != nil {
		return fmt.Errorf("unable to process outputs for target %q: %w", target, err)
	}

	// See https://docs.github.com/en/actions/using-jobs/defining-outputs-for-jobs
	githubOutput := os.Getenv("GITHUB_OUTPUT")

	f, err := os.OpenFile(githubOutput, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}

	for k, v := range outputs {
		if _, err := f.WriteString(fmt.Sprintf("%s=%s\n", k, v)); err != nil {
			return fmt.Errorf("unable to write a kv to GITHUB_OUTPUT: %w", err)
		}
	}

	if err := f.Close(); err != nil {
		return fmt.Errorf("unable to close GITHUB_OUTPUT: %w", err)
	}

	return nil
}

func (e *Plugin) exportActionsWorkflows(dir, kanvasContainerImage string) error {
	w := &actionsWorkflow{
		Name: "Plan deployment",
		On: map[string]interface{}{
			"pull_request": map[string][]string{
				"branches":     {"main"},
				"paths-ignore": {"**.md", "**/docs/**"},
			},
		},
		Jobs: make(map[string]actionsJob, len(e.wf.WorkflowJobs)),
	}

	outputs := map[string]map[string]string{}

	const step = "out"

	// Traverse the DAG of jobs
	for _, job := range e.wf.WorkflowJobs {
		if job.Driver == nil {
			continue
		}

		job.Driver.Args.Visit(func(str string) {
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
		if job.Driver == nil {
			continue
		}

		name = strings.ReplaceAll(name, "/", "-")

		var needs []string
		for _, n := range job.Needs {
			needs = append(needs, strings.ReplaceAll(n, "/", "-"))
		}

		var cmd []string
		cmd = append(cmd, job.Driver.Diff...)
		job.Driver.Args.Visit(func(str string) {
		}, func(out string) {
			jobAndOutput := strings.SplitN(out, ".", 1)
			jobName := strings.ReplaceAll(jobAndOutput[0], "/", "-")
			cmd = append(cmd, fmt.Sprintf("${{ needs.%s.outputs.%s }}", jobName, jobAndOutput[1]))
		})

		j := &actionsJob{
			RunsOn: "ubuntu-latest",
			Container: container{
				Image: kanvasContainerImage,
			},
			Outputs: outputs[name],
			Needs:   needs,
			Steps: []actionsStep{
				stepCheckout(),
				stepRun("run", cmd),
				stepRun(name, job.Driver.Output(FormatGitHubActions)),
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
	Name string                 `yaml:"name"`
	On   map[string]interface{} `yaml:"on"`
	Jobs map[string]actionsJob  `yaml:"jobs"`
}

func (w *actionsWorkflow) AddJob(name string, def actionsJob) {
	w.Jobs[name] = def
}

type actionsJob struct {
	Needs     []string          `yaml:"needs,omitempty"`
	RunsOn    string            `yaml:"runs_on"`
	Container container         `yaml:"container"`
	Outputs   map[string]string `yaml:"outputs,omitempty"`
	Steps     []actionsStep     `yaml:"steps"`
}

type container struct {
	Image string `yaml:"image"`
}

type actionsStep struct {
	ID   string                 `yaml:"id,omitempty"`
	Run  string                 `yaml:"run,omitempty"`
	Uses string                 `yaml:"uses,omitempty"`
	With map[string]interface{} `yaml:"with,omitempty"`
}

func stepCheckout() actionsStep {
	return actionsStep{
		Uses: "actions/checkout@v3",
		With: map[string]interface{}{
			"fetch-depth": 0,
		},
	}
}

func stepRun(id string, cmd []string) actionsStep {
	return actionsStep{
		ID:  id,
		Run: strings.Join(cmd, " "),
	}
}
