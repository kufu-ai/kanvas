package kanvas

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mumoshu/kargo"
)

type Driver struct {
	Diff       []kargo.Cmd
	Apply      []kargo.Cmd
	Output     func(format string) []string
	OutputFunc func(*Runtime, map[string]string) error
	Dir        string
}

func newDriver(id, dir string, c Component) (*Driver, error) {
	output := func(format string) []string {
		return append([]string{
			"kanvas", "output", "-t", id, "-f",
		},
			format,
		)
	}

	if c.Docker != nil {
		// TODO Append some unique-ish ID of the to-be-produced image
		image := c.Docker.Image
		dockerfile := c.Docker.File
		if dockerfile == "" {
			dockerfile = "Dockerfile"
		}
		return &Driver{
			Dir:    dir,
			Diff:   []kargo.Cmd{{Name: "docker", Args: kargo.NewArgs("build", "-t", image, "-f", dockerfile), Dir: dir}},
			Apply:  []kargo.Cmd{{Name: "docker", Args: kargo.NewArgs("push", image)}},
			Output: output,
			OutputFunc: func(r *Runtime, o map[string]string) error {
				var buf bytes.Buffer
				if err := r.Exec(dir, []string{"docker", "inspect", "--format={{index .RepoDigests 0}}"}, ExecStdout(&buf)); err != nil {
					return fmt.Errorf("docker-inspect failed: %w", err)
				}
				o["id"] = buf.String()
				return nil
			},
		}, nil
	} else if c.Terraform != nil {
		var args []string

		if c.Terraform.Target != "" {
			args = []string{"-target", c.Terraform.Target}
		}

		dynArgs := &kargo.Args{}
		for _, v := range c.Terraform.Vars {
			dynArgs.Append("-var")
			dynArgs.AppendValueFromOutput(v.ValueFrom)
		}

		return &Driver{
			Dir:    dir,
			Diff:   []kargo.Cmd{{Name: "terraform", Args: kargo.NewArgs("plan", args, dynArgs)}},
			Apply:  []kargo.Cmd{{Name: "terraform", Args: kargo.NewArgs("apply", args, dynArgs)}},
			Output: output,
			OutputFunc: func(r *Runtime, o map[string]string) error {
				var buf bytes.Buffer
				if err := r.Exec(dir, []string{"terraform", "output", "-json"}, ExecStdout(&buf)); err != nil {
					return fmt.Errorf("terraform-output failed: %w", err)
				}

				d := json.NewDecoder(&buf)

				type terraformOutput struct {
					Sensitive bool        `json:"sensitive"`
					Type      string      `json:"type"`
					Value     interface{} `json:"value"`
				}
				type terraformOutputs struct {
					Outputs map[string]terraformOutput `json:"outputs"`
				}

				m := terraformOutputs{
					Outputs: map[string]terraformOutput{},
				}
				if err := d.Decode(&m); err != nil {
					return fmt.Errorf("unable to decode terraform outputs: %w", err)
				}

				for k, out := range m.Outputs {
					switch tpe := out.Type; tpe {
					case "string":
						o[k] = out.Value.(string)
					case "number":
						o[k] = strconv.Itoa(out.Value.(int))
					default:
						return fmt.Errorf("unable to unmarshal terraform output %q of type %q", k, tpe)
					}
				}

				o["_raw"] = buf.String()
				return nil
			},
		}, nil
	} else if c.Kubernetes != nil {
		var (
			name = c.Kubernetes.Name
		)
		if name == "" {
			name = filepath.Base(c.Dir)
		}

		c.Kubernetes.Name = name
		c.Kubernetes.Path = c.Dir

		g := &kargo.Generator{
			GetValue: func(key string) (string, error) {
				return strings.ToUpper(key), nil
			},
			TailLogs: false,
		}

		diff, err := g.ExecCmds(&c.Kubernetes.Config, kargo.Plan)
		if err != nil {
			return nil, fmt.Errorf("generating plan commands: %w", err)
		}

		apply, err := g.ExecCmds(&c.Kubernetes.Config, kargo.Apply)
		if err != nil {
			return nil, fmt.Errorf("generating apply commands: %w", err)
		}

		return &Driver{
			Dir:    dir,
			Diff:   diff,
			Apply:  apply,
			Output: output,
			OutputFunc: func(r *Runtime, o map[string]string) error {
				return nil
			},
		}, nil
	} else {
		return &Driver{
			Dir:    dir,
			Diff:   []kargo.Cmd{},
			Apply:  []kargo.Cmd{},
			Output: output,
			OutputFunc: func(r *Runtime, o map[string]string) error {
				return nil
			},
		}, nil
	}

	if len(c.Components) == 0 {
		return nil, fmt.Errorf("invalid component: this component has no driver or components")
	}

	return nil, nil
}
