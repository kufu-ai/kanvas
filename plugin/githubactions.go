package plugin

import (
	"fmt"
	"kanvas"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/mumoshu/kargo"
)

const (
	FormatGitHubActions = "githubactions"

	FormatDefault = FormatGitHubActions
)

func (e *Plugin) outputActionsWorkflows(op kanvas.Op, target string) error {
	outputs := map[string]string{}
	if err := e.wf.WorkflowJobs[target].Driver.OutputFunc(e.r, op, outputs); err != nil {
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

	id := func(raw string) string {
		if raw[0] == '/' {
			raw = raw[1:]
		}
		return strings.ReplaceAll(raw, "/", "-")
	}

	const (
		OutputStepID = "out"
	)

	// Traverse the DAG of jobs
	for name, job := range e.wf.WorkflowJobs {
		if job.Driver == nil {
			continue
		}

		name = id(name)

		for _, step := range job.Driver.Diff {
			for _, c := range step.Run {
				c.Args.Visit(func(str string) {
				}, func(a kargo.DynArg) {
					jobAndOutput := strings.SplitN(a.FromOutput, ".", 2)
					if len(jobAndOutput) != 2 {
						// TODO make this error instead
						panic(fmt.Errorf("Could not find dot(.) within %q", a.FromOutput))
					}
					var jobName string
					if j := jobAndOutput[0]; j[0] == '/' {
						jobName = id(j)
					} else {
						fqn := filepath.Join(strings.ReplaceAll(name, "-", "/"), "..", j)
						jobName = id(fqn)
					}
					if _, ok := outputs[jobName]; !ok {
						outputs[jobName] = map[string]string{}
					}
					outputs[jobName][jobAndOutput[1]] = fmt.Sprintf("${{ steps.%s.outputs.%s }}", OutputStepID, jobAndOutput[1])
				}, func(in kargo.KargoValueProvider) {
				})
			}
		}
	}

	for name, job := range e.wf.WorkflowJobs {
		if job.Driver == nil {
			continue
		}

		name = id(name)

		var needs []string
		for _, n := range job.Needs {
			needs = append(needs, id(n))
		}

		steps := []actionsStep{
			stepCheckout(),
		}

		for i, s := range job.Driver.Diff {
			for j, cmd := range s.Run {
				stepID := cmd.ID
				if stepID == "" {
					if len(job.Driver.Diff) == 1 {
						stepID = "run"
					} else {
						stepID = fmt.Sprintf("run%d%d", i, j)
					}
				}
				steps = append(steps, stepRun(
					stepID,
					cmd,
					func(out string) (string, error) {
						jobAndOutput := strings.SplitN(out, ".", 2)
						if len(jobAndOutput) != 2 {
							// TODO make this error instead
							panic(fmt.Errorf("could not find dot(.) within %q", out))
						}
						var jobName string
						if j := jobAndOutput[0]; j[0] == '/' {
							jobName = id(j)
						} else {
							fqn := filepath.Join(strings.ReplaceAll(name, "-", "/"), "..", j)
							jobName = id(fqn)
						}
						return fmt.Sprintf("${{ needs.%s.outputs.%s }}", jobName, jobAndOutput[1]), nil
					},
				))
			}
		}

		steps = append(steps, actionsStep{
			ID:  OutputStepID,
			Run: strings.Join(job.Driver.Output(FormatGitHubActions), " "),
		})

		o := outputs[name]
		j := &actionsJob{
			RunsOn: "ubuntu-latest",
			Container: container{
				Image: kanvasContainerImage,
			},
			Outputs: o,
			Needs:   needs,
			Steps:   steps,
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
	// See https://docs.github.com/en/actions/using-workflows/workflow-syntax-for-github-actions#jobsjob_idstepsrun
	WorkingDirectory string `yaml:"working-directory,omitempty"`
}

func stepCheckout() actionsStep {
	return actionsStep{
		Uses: "actions/checkout@v3",
		With: map[string]interface{}{
			"fetch-depth": 0,
		},
	}
}

func stepRun(id string, cmd kargo.Cmd, get func(string) (string, error)) actionsStep {
	run := fmt.Sprintf("%s %s", cmd.Name, strings.Join(cmd.Args.MustCollect(get), " "))

	return actionsStep{
		ID:               id,
		Run:              run,
		WorkingDirectory: cmd.Dir,
	}
}
