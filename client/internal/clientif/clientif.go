package clientif

import (
	"context"

	"github.com/davinci-std/kanvas/client"
)

type Client interface {
	// Apply applies the configuration in the given config to the environment env.
	//
	// config is the path to the configuration file, which looks like path/to/kanvas.yaml.
	// env is the name of the environment. The specified configuration needs to have the environment whose name is env.
	Apply(ctx context.Context, config, env string, opts client.ApplyOptions) (*client.ApplyResult, error)
	// Diff compares the desired state against the current state for the configuration in the given config.
	//
	// config is the path to the configuration file, which looks like path/to/kanvas.yaml.
	// env is the name of the environment. The specified configuration needs to have the environment whose name is env.
	Diff(ctx context.Context, config, env string, opts client.DiffOptions) (*client.DiffResult, error)
}

// Options is the command-line options for kanvas apply and diff commands
type Options interface {
	// GetSkip returns the list of component names to skip,
	// which is passed to --skip after joining the list with comma.
	GetSkip() []string
	// GetSkippedComponents returns the map of component name to its output.
	// It is passed to --skipped-jobs-outputs as JSON.
	GetSkippedComponents() map[string]map[string]string
	// GetEnvVars returns the map of environment variable name to its value.
	// It is set when running kanvas.
	GetEnvVars() map[string]string
}
