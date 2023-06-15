package kanvas

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/mumoshu/kargo"
	"github.com/mumoshu/kargo/cmd"
)

type Step Task
type Steps []Step

type Task struct {
	IfOutputEq IfOutputEq
	Run        []kargo.Cmd
	OutputFunc func(*Runtime, map[string]string) error
}

type IfOutputEq struct {
	Key   string
	Value string
}

type Driver struct {
	Diff       Steps
	Apply      Steps
	Output     func(format string) []string
	OutputFunc func(*Runtime, Op, map[string]string) error
}

type Op int

const (
	Diff Op = iota
	Apply
)

type Options struct {
	ConfigFile string
	LogsFollow bool
}

func (o Options) GetConfigPath() string {
	if o.ConfigFile != "" {
		return o.ConfigFile
	}
	return "kanvas.yaml"
}

func newDriver(id, dir string, c Component, opts Options) (*Driver, error) {
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
		dockerBuild := cmd.New(
			"docker-build",
			"docker",
			cmd.Args("build", "-t", image, "-f", dockerfile, "."),
			cmd.Dir(dir),
		)
		dockerPush := cmd.New(
			"docker-push",
			"docker",
			cmd.Args("push", image),
		)
		dockerBuildxPush := cmd.New(
			"docker-buildx-push",
			"docker",
			cmd.Args("build", "--load", "--platform", "linux/amd64", "-t", image, "-f", dockerfile, "."),
		)
		dockerBuildXCheckAvailability := Step(Task{
			OutputFunc: func(r *Runtime, o map[string]string) error {
				if err := r.Exec(dir, []string{"docker", "buildx", "inspect"}); err != nil {
					o["kanvas.buildx"] = "false"
				} else {
					o["kanvas.buildx"] = "true"
				}
				return nil
			},
		})

		dockerBuildXPushIfAvailable := Step(Task{
			IfOutputEq: IfOutputEq{
				Key:   "kanvas.buildx",
				Value: "true",
			},
			Run: []kargo.Cmd{
				dockerBuildxPush,
			},
		})
		dockerBuildAndPushIfBuildxNotAvailable := Step(Task{
			IfOutputEq: IfOutputEq{
				Key:   "kanvas.buildx",
				Value: "false",
			},
			Run: []kargo.Cmd{
				dockerBuild,
				dockerPush,
			},
		})
		return &Driver{
			Diff: Seq(
				cmdToStep(dockerBuild),
			),
			Apply: Seq(
				dockerBuildXCheckAvailability,
				dockerBuildXPushIfAvailable,
				dockerBuildAndPushIfBuildxNotAvailable,
			),
			Output: output,
			OutputFunc: func(r *Runtime, op Op, o map[string]string) error {
				var buf bytes.Buffer
				if err := r.Exec(dir, []string{"docker", "inspect", "--format={{.ID}}", image}, ExecStdout(&buf)); err != nil {
					return fmt.Errorf("docker-inspect failed: %w", err)
				}
				o["id"] = buf.String()
				return nil
			},
		}, nil
	} else if c.Terraform != nil {
		var args []string

		// if c.Terraform.Target != "" {
		// 	args = []string{"-target", c.Terraform.Target}
		// }

		dynArgs := &kargo.Args{}
		for _, v := range c.Terraform.Vars {
			dynArgs.Append("-var")
			if v.ValueFrom != "" {
				dynArgs.AppendValueFromOutputWithPrefix(
					fmt.Sprintf("%s=", v.Name),
					v.ValueFrom,
				)
			} else if v.Value != "" {
				dynArgs.Append(fmt.Sprintf("%s=%s", v.Name, v.Value))
			} else {
				return nil, fmt.Errorf("invalid var %v: it must have either Value or ValueFrom", v)
			}
		}

		applyArgs := append([]string{}, args...)
		applyArgs = append(applyArgs, "-auto-approve")

		return &Driver{
			Diff: Seq(
				Cmd("terraform-init", "terraform", cmd.Args("init"), cmd.Dir(dir)),
				Cmd("terraform-plan", "terraform", cmd.Args("plan", args, dynArgs), cmd.Dir(dir)),
			),
			Apply: Seq(
				Cmd("terraform-init", "terraform", cmd.Args("init"), cmd.Dir(dir)),
				Cmd("terraform-apply", "terraform", cmd.Args("apply", applyArgs, dynArgs), cmd.Dir(dir)),
			),
			Output: output,
			OutputFunc: func(r *Runtime, op Op, o map[string]string) error {
				var buf bytes.Buffer
				if err := r.Exec(dir, []string{"terraform", "output", "-json"}, ExecStdout(&buf)); err != nil {
					return fmt.Errorf("terraform-output failed: %w", err)
				}

				out := buf.String()

				d := json.NewDecoder(bytes.NewBufferString(out))

				type terraformOutput struct {
					Sensitive bool        `json:"sensitive"`
					Type      string      `json:"type"`
					Value     interface{} `json:"value"`
				}

				m := map[string]terraformOutput{}
				if err := d.Decode(&m); err != nil {
					return fmt.Errorf("unable to decode terraform outputs: %w", err)
				}

				for k, out := range m {
					switch tpe := out.Type; tpe {
					case "string":
						o[k] = out.Value.(string)
					case "number":
						o[k] = strconv.Itoa(out.Value.(int))
					case "bool":
						o[k] = strconv.FormatBool(out.Value.(bool))
					default:
						return fmt.Errorf("unable to unmarshal terraform output %q of type %q", k, tpe)
					}
				}

				o["_raw"] = buf.String()

				if op != Diff {
					return nil
				}

				type Output struct {
					Name    string   `hcl:"name,label"`
					Options hcl.Body `hcl:",remain"`
				}

				type Config struct {
					Outputs []Output `hcl:"output,block"`
					Options hcl.Body `hcl:",remain"`
				}

				files, err := filepath.Glob(filepath.Join(dir, "*.tf"))
				if err != nil {
					return fmt.Errorf("unable to glob %s/*.tf: %w", dir, err)
				}

				for _, f := range files {
					var config Config
					err := decodeHCLFile(f, nil, &config)
					if err != nil {
						return fmt.Errorf("failed to load configuration %s: %v", f, err)
					}
					fmt.Fprintf(os.Stdout, "%v\n", config)
					for _, out := range config.Outputs {
						if _, ok := o[out.Name]; !ok {
							o[out.Name] = "<computed>"
						}
					}
				}

				return nil
			},
		}, nil
	} else if c.Kubernetes != nil {
		var (
			name = c.Kubernetes.Name
		)

		absdir, err := filepath.Abs(dir)
		if err != nil {
			return nil, fmt.Errorf("invalid dir %q: %w", dir, err)
		}

		if name == "" {
			name = filepath.Base(absdir)
		}

		c.Kubernetes.Name = name

		g := &kargo.Generator{
			GetValue: func(key string) (string, error) {
				return "$" + strings.ToUpper(strings.ReplaceAll(key, ".", "_")), nil
			},
			TailLogs: opts.LogsFollow,
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
			Diff:   cmdsToSeq(diff),
			Apply:  cmdsToSeq(apply),
			Output: output,
			OutputFunc: func(r *Runtime, op Op, o map[string]string) error {
				return nil
			},
		}, nil
	} else {
		return &Driver{
			Diff:   Seq(),
			Apply:  Seq(),
			Output: output,
			OutputFunc: func(r *Runtime, op Op, o map[string]string) error {
				return nil
			},
		}, nil
	}

	if len(c.Components) == 0 {
		return nil, fmt.Errorf("invalid component: this component has no driver or components")
	}

	return nil, nil
}

func Cmd(id, name string, opts ...cmd.Option) Step {
	c := cmd.New(id, name, opts...)

	return cmdToStep(c)
}

func cmdToTask(cmd kargo.Cmd) Task {
	return Task{
		Run: []kargo.Cmd{cmd},
	}
}

func cmdToStep(cmd kargo.Cmd) Step {
	return Step(cmdToTask(cmd))
}

func cmdsToSeq(cmds []kargo.Cmd) Steps {
	var steps []Step
	for _, cmd := range cmds {
		steps = append(steps, cmdToStep(cmd))
	}
	return Seq(steps...)
}

func Seq(c ...Step) Steps {
	return c
}
