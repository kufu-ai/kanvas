package kanvas

import (
	"fmt"
)

type driver struct {
	diff  []string
	apply []string
	dir   string

	args *Args
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

func newDriver(id, dir string, c Component) (*driver, error) {
	if c.Docker != nil {
		// TODO Append some unique-ish ID of the to-be-produced image
		image := c.Docker.Image
		return &driver{
			dir:   dir,
			diff:  []string{"docker", "build", "-t", image, "-f", c.Docker.File, dir},
			apply: []string{"docker", "push", image},
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

		return &driver{
			dir:   dir,
			diff:  append([]string{"terraform", "plan"}, args...),
			apply: append([]string{"terraform", "apply"}, args...),
			args:  dynArgs,
		}, nil
	}

	return nil, fmt.Errorf("no driver specified for this component")
}
