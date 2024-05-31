package kanvas

import (
	"fmt"
	"os"
	"strings"
)

const (
	defaultConfigFileBase            = "kanvas"
	DefaultConfigFileYAML            = defaultConfigFileBase + ".yaml"
	DefaultConfigFileJsonnet         = defaultConfigFileBase + ".jsonnet"
	DefaultConfigFileTemplateJsonnet = defaultConfigFileBase + ".template.jsonnet"
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

	var (
		found []string
	)

	if _, err := os.Stat(DefaultConfigFileYAML); err == nil {
		found = append(found, DefaultConfigFileYAML)
	}

	if _, err := os.Stat(DefaultConfigFileJsonnet); err == nil {
		found = append(found, DefaultConfigFileJsonnet)
	}

	if _, err := os.Stat(DefaultConfigFileTemplateJsonnet); err == nil {
		found = append(found, DefaultConfigFileTemplateJsonnet)
	}

	if len(found) > 1 {
		return "", fmt.Errorf("%d config files %s found. Please specify the config file with --config-file", len(found), strings.Join(found, ", "))
	} else if len(found) == 0 {
		return "", fmt.Errorf("unable to find config file")
	}

	return found[0], nil
}
