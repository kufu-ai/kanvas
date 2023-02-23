package kanvas

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
)

type Driver struct {
	Diff       []string
	Apply      []string
	Output     func(format string) []string
	OutputFunc func(*Runtime, map[string]string) error
	Dir        string

	Args *Args
}

type Args struct {
	underlying []interface{}
}

func (a *Args) AddString(v string) {
	a.underlying = append(a.underlying, v)
}

func (a *Args) AddValueFromOutput(ref string) {
	a.underlying = append(a.underlying, DynArg{FromOutput: ref})
}

func (a *Args) Visit(str func(string), out func(string)) {
	for _, x := range a.underlying {
		switch a := x.(type) {
		case string:
			str(a)
		case DynArg:
			out(a.FromOutput)
		default:
			panic(fmt.Sprintf("unexpected type(%T) of item: %q", a, a))
		}
	}
}

type DynArg struct {
	FromOutput string
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
		return &Driver{
			Dir:    dir,
			Diff:   []string{"docker", "build", "-t", image, "-f", c.Docker.File, dir},
			Apply:  []string{"docker", "push", image},
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

		dynArgs := &Args{}
		for _, v := range c.Terraform.Vars {
			dynArgs.AddString("-var")
			dynArgs.AddValueFromOutput(v.ValueFrom)
		}

		return &Driver{
			Dir:    dir,
			Diff:   append([]string{"terraform", "plan"}, args...),
			Apply:  append([]string{"terraform", "apply"}, args...),
			Output: output,
			Args:   dynArgs,
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
	}

	return nil, fmt.Errorf("no driver specified for this component")
}
