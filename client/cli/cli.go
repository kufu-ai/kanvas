package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/davinci-std/kanvas/client/internal/clientif"

	"github.com/davinci-std/kanvas/client"
)

var _ clientif.Client = &Client{}

// Client is a command-line client for kanvas.
// It runs the kanvas command with the given options.
//
// If you are building a bot or another tool that uses kanvas, this should be a
// good starting point.
//
// Note that the kanvas command needs to be installed in the environment where
// this client is run.
//
// Also note that the repository containing the kanvas config file needs to be
// cloned in the environment where this client is run.
//
// Usage:
//
//		Let's say you have a configuration file at path/to/kanvas.yaml.
//	    The configuration file looks like:
//
//	     environments:
//	       dev:
//			 uses:
//			   k8sapp: {}
//	     components:
//	       k8sapp: {}
//
//		You want to apply the configuration to the environment dev.
//
//		cli := cli.New()
//		cli.Command = []string{"kanvas"}
//
//		// Diffing
//		res, err := cli.Diff(context.Background(), "path/to/kanvas.yaml", "dev", client.DiffOptions{})
//
//		// Applying
//		res, err := cli.Apply(context.Background(), "path/to/kanvas.yaml", "dev", client.ApplyOptions{})
type Client struct {
	// Command is the path to the kanvas command.
	// Defaults to "kanvas".
	Command []string
}

func New() *Client {
	return &Client{}
}

// Apply applies the configuration in the given config to the environment env.
//
// config is the path to the configuration file, which looks like path/to/kanvas.yaml.
// env is the name of the environment. The specified configuration needs to have the environment whose name is env.
func (c *Client) Apply(ctx context.Context, config, env string, opts client.ApplyOptions) (*client.ApplyResult, error) {
	out, err := c.run(ctx, config, env, &opts, "apply")
	if err != nil {
		return nil, err
	}

	var r client.ApplyResult

	if err := json.Unmarshal(out.Bytes(), &r); err != nil {
		return nil, fmt.Errorf("failed to unmarshal json: %w", err)
	}

	return &r, nil
}

// Diff compares the desired state against the current state for the configuration in the given config.
//
// config is the path to the configuration file, which looks like path/to/kanvas.yaml.
// env is the name of the environment. The specified configuration needs to have the environment whose name is env.
func (c *Client) Diff(ctx context.Context, config, env string, opts client.DiffOptions) (*client.DiffResult, error) {
	out, err := c.run(ctx, config, env, &opts, "diff")
	if err != nil {
		return nil, err
	}

	var r client.DiffResult

	if err := json.Unmarshal(out.Bytes(), &r); err != nil {
		return nil, fmt.Errorf("failed to unmarshal json: %w", err)
	}

	return &r, nil
}

func (c *Client) run(ctx context.Context, config, env string, opts clientif.Options, command string) (*bytes.Buffer, error) {
	return run(ctx, config, env, opts, c.GetCommand(), command)
}

func (c *Client) GetCommand() []string {
	if len(c.Command) > 0 {
		return c.Command
	}
	return []string{"kanvas"}
}

func run(ctx context.Context, configPath, env string, opts clientif.Options, bin []string, command string) (*bytes.Buffer, error) {
	var a []string

	configDir, configName := filepath.Split(configPath)

	a = append(a, "--config", configName, "--env", env, command)

	if opts.GetSkip() != nil {
		a = append(a, "--skip", strings.Join(opts.GetSkip(), ","))
	}

	if opts.GetSkippedComponents() != nil {
		b, err := json.Marshal(opts.GetSkippedComponents())
		if err != nil {
			return nil, fmt.Errorf("failed to marshal skipped components: %w", err)
		}
		a = append(a, "--skipped-jobs-outputs", string(b))
	}

	cmdName := bin[0]
	if len(bin) > 1 {
		a = append(bin[1:], a...)
	}

	cmd := exec.CommandContext(ctx, cmdName, a...)

	cmd.Dir = configDir

	if opts.GetEnvVars() != nil && len(opts.GetEnvVars()) > 0 {
		cmd.Env = append(cmd.Env, os.Environ()...)

		for k, v := range opts.GetEnvVars() {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	var (
		stdout bytes.Buffer
		stderr bytes.Buffer
	)

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("command kanvas %v: %w: %s", a, err, stderr.String())
	}

	return &stdout, nil
}
