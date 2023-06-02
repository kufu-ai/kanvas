package kanvas

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
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
	if c.Stdout != nil {
		var stderr bytes.Buffer

		c.Stderr = &stderr
		if err := c.Run(); err != nil {
			return fmt.Errorf("executing %q in %q: %w: %s", cmd, dir, err, stderr.String())
		}
	} else {
		var (
			stdout, stderr bytes.Buffer
		)
		c.Stdout = io.MultiWriter(&stdout, os.Stdout)
		c.Stderr = io.MultiWriter(&stderr, os.Stderr)
		err := c.Run()
		if err != nil {
			return fmt.Errorf("executing %q in %q: %w: %s", cmd, dir, err, stderr.String())
		}
	}

	return nil
}
