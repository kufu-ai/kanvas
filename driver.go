package kanvas

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"dario.cat/mergo"
	"github.com/hashicorp/hcl/v2"
	"github.com/helmfile/vals"
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
	Env        string
	ConfigFile string
	LogsFollow bool
	// TempDir is the directory to store temporary files such as the generated Kubernetes manifests
	// or modified copy of a Git repository to be deployed via GitOps.
	// Although this is under the "Options", it is not optional as of today.
	TempDir string
	// UseAI enables AI to suggest a kanvas.yaml file content based on your environment.
	UseAI bool
	// Skip is a list of components to skip.
	Skip []string
	// SkippedJobsOutputs is a map of outputs for skipped jobs.
	// The keys must be found in the Skip list.
	// For example, if Skip is ["foo"], then SkippedJobsOutputs must have a key "foo".
	SkippedJobsOutputs map[string]map[string]string
}

func (o Options) GetConfigFilePath() string {
	return o.ConfigFile
}

func concat(a ...[]interface{}) []interface{} {
	var r []interface{}
	for _, s := range a {
		r = append(r, s...)
	}
	return r
}

func kanvasOutputCommandForID(id string) func(format string) []string {
	return func(format string) []string {
		return append([]string{
			"kanvas", "output", "-t", id, "-f",
		},
			format,
		)
	}
}

func newDriver(id, dir string, c Component, opts Options) (*Driver, error) {
	output := kanvasOutputCommandForID(id)

	if c.AWS != nil {
		return &Driver{
			Diff:   Seq(),
			Apply:  Seq(),
			Output: output,
			OutputFunc: func(r *Runtime, op Op, o map[string]string) error {
				// Get a get-caller-identity result and compare
				// returned Account with the one in the config
				var buf bytes.Buffer
				if err := r.Exec(dir, []string{"aws", "sts", "get-caller-identity"}, ExecStdout(&buf)); err != nil {
					return fmt.Errorf("aws-sts-get-caller-identity failed: %w", err)
				}

				type CallerIdentity struct {
					Account string `json:"Account"`
				}

				var ci CallerIdentity
				if err := json.NewDecoder(&buf).Decode(&ci); err != nil {
					return fmt.Errorf("unable to decode aws-sts-get-caller-identity output: %w", err)
				}

				if ci.Account != c.AWS.Account {
					return fmt.Errorf("aws-sts-get-caller-identity returned Account %q, which is different from the one in the config %q", ci.Account, c.AWS.Account)
				}

				return nil
			},
		}, nil
	} else if c.Docker != nil {
		// TODO Append some unique-ish ID of the to-be-produced image
		image := c.Docker.Image
		dockerfile := c.Docker.File
		if dockerfile == "" {
			dockerfile = "Dockerfile"
		}
		buildArgs := []interface{}{"build"}
		for name, value := range c.Docker.Args {
			buildArgs = append(buildArgs, "--build-arg", fmt.Sprintf("%s=%s", name, value))
		}
		dynBuildArgs := &kargo.Args{}
		for name, valueFrom := range c.Docker.ArgsFrom {
			dynBuildArgs = dynBuildArgs.Append("--build-arg")
			dynBuildArgs = dynBuildArgs.AppendValueFromOutputWithPrefix(
				fmt.Sprintf("%s=", name),
				valueFrom,
			)
		}

		dynTags := &kargo.Args{}
		for _, tagFrom := range c.Docker.TagsFrom {
			dynTags = dynTags.Append("-t")
			dynTags = dynTags.AppendValueFromOutput(tagFrom)
		}

		dockerBuild := cmd.New(
			"docker-build",
			"docker",
			cmd.Args(concat(buildArgs, []interface{}{"-t", image, "-f", dockerfile})...),
			cmd.Args(dynBuildArgs),
			cmd.Args(dynTags),
			cmd.Args("."),
			cmd.Dir(dir),
		)
		dockerBuildxLoad := cmd.New(
			"docker-buildx-push",
			"docker",
			cmd.Args(concat(buildArgs, []interface{}{"--load", "--platform", "linux/amd64", "-t", image, "-f", dockerfile})...),
			cmd.Args(dynBuildArgs),
			cmd.Args(dynTags),
			cmd.Args("."),
		)
		dockerPush := cmd.New(
			"docker-push",
			"docker",
			cmd.Args("push", image),
		)
		dockerBuildxPush := cmd.New(
			"docker-buildx-push",
			"docker",
			cmd.Args(concat(buildArgs, []interface{}{"--push", "--platform", "linux/amd64", "-t", image, "-f", dockerfile, "."})...),
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
		dockerBuildXBuildLoadIfAvailable := Step(Task{
			IfOutputEq: IfOutputEq{
				Key:   "kanvas.buildx",
				Value: "true",
			},
			Run: []kargo.Cmd{
				dockerBuildxLoad,
			},
		})
		dockerBuildIfBuildxNotAvailable := Step(Task{
			IfOutputEq: IfOutputEq{
				Key:   "kanvas.buildx",
				Value: "false",
			},
			Run: []kargo.Cmd{
				dockerBuild,
			},
		})
		return &Driver{
			Diff: Seq(
				dockerBuildXCheckAvailability,
				dockerBuildXBuildLoadIfAvailable,
				dockerBuildIfBuildxNotAvailable,
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
				o["id"] = strings.TrimSpace(buf.String())
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
					fmt.Fprintf(os.Stderr, "Decoded HCL config: %v\n", config)
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
		if opts.TempDir == "" {
			return nil, fmt.Errorf("invalid kubernetes component: TempDir is not set")
		}

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
			TailLogs:     opts.LogsFollow,
			ToolsCommand: []string{"kanvas", "tools"},
			ToolName:     "kanvas",
			TempDir:      opts.TempDir,
		}

		var kc kargo.Config

		if err := mergo.Merge(&kc, c.Kubernetes.Config, mergo.WithOverride); err != nil {
			return nil, fmt.Errorf("unable to merge kubernetes component: %w", err)
		}

		if kc.Path != "" {
			kc.Path = filepath.Join(absdir, kc.Path)
		} else {
			kc.Path = absdir
		}

		diff, err := g.ExecCmds(&kc, kargo.Plan)
		if err != nil {
			return nil, fmt.Errorf("generating plan commands: %w", err)
		}

		apply, err := g.ExecCmds(&kc, kargo.Apply)
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
	} else if c.Externals != nil {
		return &Driver{
			Diff:   Seq(),
			Apply:  Seq(),
			Output: output,
			OutputFunc: func(r *Runtime, op Op, o map[string]string) error {
				rt, err := vals.New(vals.Options{CacheSize: 512})
				if err != nil {
					return fmt.Errorf("unable to init vals: %w", err)
				}

				m, err := c.Externals.NewValsTemplate()
				if err != nil {
					return fmt.Errorf("unable to create vals template: %w", err)
				}

				out, err := rt.Eval(m)
				if err != nil {
					return fmt.Errorf("unable to run vals eval: %w", err)
				}

				for k, v := range out {
					o[k] = fmt.Sprintf("%s", v)
				}

				return nil
			},
		}, nil
	} else if c.Noop != nil {
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
	return &Driver{
		Diff:   Seq(),
		Apply:  Seq(),
		Output: output,
		OutputFunc: func(r *Runtime, op Op, o map[string]string) error {
			return nil
		},
	}, nil
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
