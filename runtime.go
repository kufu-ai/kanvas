package kanvas

import (
	"context"
	"fmt"
	"io"
	"os/exec"
)

type Runtime struct {
}

func NewRuntime() *Runtime {
	return &Runtime{}
}

type ExecOption func(*exec.Cmd)

func ExecStdout(w io.Writer) ExecOption {
	return func(c *exec.Cmd) {
		c.Stdout = w
	}
}

func (r *Runtime) Exec(dir string, cmd []string, opts ...ExecOption) error {
	c := exec.CommandContext(context.TODO(), cmd[0], cmd[1:]...)
	c.Dir = dir
	for _, o := range opts {
		o(c)
	}
	out, err := c.CombinedOutput()
	if err != nil {
		return fmt.Errorf("executing %q: %w: %s", cmd, err, out)
	}

	return nil
}
