package kanvas

import (
	"fmt"
)

type Driver struct {
	Diff  []string
	Apply []string
	Dir   string

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
	if c.Docker != nil {
		// TODO Append some unique-ish ID of the to-be-produced image
		image := c.Docker.Image
		return &Driver{
			Dir:   dir,
			Diff:  []string{"docker", "build", "-t", image, "-f", c.Docker.File, dir},
			Apply: []string{"docker", "push", image},
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
			Dir:   dir,
			Diff:  append([]string{"terraform", "plan"}, args...),
			Apply: append([]string{"terraform", "apply"}, args...),
			Args:  dynArgs,
		}, nil
	}

	return nil, fmt.Errorf("no driver specified for this component")
}
