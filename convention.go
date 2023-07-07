package kanvas

import (
	"fmt"
	"os"
)

const (
	defaultConfigFileBase    = "kanvas"
	DefaultConfigFileYAML    = defaultConfigFileBase + ".yaml"
	DefaultConfigFileJsonnet = defaultConfigFileBase + ".jsonnet"
)

// DiscoverConfigFile returns the path to the config file to use.
// If the config file is not specified, it will try to find the config file in the current directory.
// The config file can be either a kanvas.yaml or kanvas.jsonnet file.
//
// If both the yaml and jsonnet config files exist, it will return an error.
// If the config file is not specified and no config file is found, it will return an error.
func DiscoverConfigFile(opts Options) (string, error) {
	if opts.ConfigFile != "" {
		return opts.ConfigFile, nil
	}

	var yamlFound, jsonnetFound bool

	if _, err := os.Stat(DefaultConfigFileYAML); err == nil {
		yamlFound = true
	}

	if _, err := os.Stat(DefaultConfigFileJsonnet); err == nil {
		jsonnetFound = true
	}

	if yamlFound && jsonnetFound {
		return "", fmt.Errorf("both %q and %q exist. Please specify the config file with --config-file", DefaultConfigFileYAML, DefaultConfigFileJsonnet)
	}

	if yamlFound {
		return DefaultConfigFileYAML, nil
	}

	if jsonnetFound {
		return DefaultConfigFileJsonnet, nil
	}

	return "", fmt.Errorf("unable to find config file")
}
