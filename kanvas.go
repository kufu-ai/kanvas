//go:generate docgen kanvas.go kanvas_doc.go Component

package kanvas

import (
	"os"
	"path/filepath"

	"github.com/goccy/go-yaml"
	"github.com/google/go-jsonnet"
	"github.com/mumoshu/kargo"
)

// Component is a component of the application
type Component struct {
	// Dir is the directory to be chdir'ed before running the commands
	// If empty, this defaults to the base dir, which is where kanvas.yaml is located.
	Dir string `yaml:"dir,omitempty"`
	// Components is a map of sub-components
	Components map[string]Component `yaml:"components"`
	// Needs is a list of components that this component depends on
	Needs []string `yaml:"needs,omitempty"`
	// Docker is a docker-specific configuration
	Docker *Docker `yaml:"docker,omitempty"`
	// Terraform is a terraform-specific configuration
	Terraform *Terraform `yaml:"terraform,omitempty"`
	// Kubernetes is a kubernetes-specific configuration
	Kubernetes *Kubernetes `yaml:"kubernetes,omitempty"`
	// Environments is a map of environments
	Environments map[string]Environment `yaml:"environments,omitempty"`
	// Externals exposes external parameters and secrets as the component's outputs
	Externals *Externals `yaml:"externals,omitempty"`
}

// Environment is a set of sub-components to replace the defaults
type Environment struct {
	// Defaults is the environment-specific defaults
	Defaults Component `yaml:"defaults,omitempty"`
	// Uses is a set of sub-components to replace the defaults
	Uses map[string]Component `yaml:"uses,omitempty"`
}

// Docker is a docker-specific configuration
type Docker struct {
	// Image is the name of the image to be built
	Image string `yaml:"image"`
	// File is the path to the Dockerfile
	File string            `yaml:"file"`
	Args map[string]string `yaml:"args"`
}

// Terraform is a terraform-specific configuration
type Terraform struct {
	// Target is the target resource to be deployed
	Target string `yaml:"target"`
	// Vars is a list of variables to be passed to terraform
	Vars []Var `yaml:"vars"`
}

// Var is a variable to be passed to terraform
type Var struct {
	// Name is the name of the variable
	Name string `yaml:"name"`
	// ValueFrom is the source of the value of the variable
	ValueFrom string `yaml:"valueFrom"`
	// Value is the value of the variable
	Value string `yaml:"value"`
}

// Kubernetes is a kubernetes-specific configuration
type Kubernetes struct {
	// Config contains all the Kubernetes-specific configuration
	kargo.Config `yaml:",inline"`
}

// LoadConfig loads the configuration file
// The configuration file is either a yaml or jsonnet file.
//
// If the file is a yaml file, it is unmarshalled into the Component struct as-is.
//
// The jsonnet file can be used to generate the yaml file.
// If the file is a jsonnet file, it is compiled to json first.
// The compiled json is then unmarshalled into the Component struct.
func LoadConfig(opts Options) (*Component, error) {
	var (
		config Component
	)

	path, err := DiscoverConfigFile(opts)
	if err != nil {
		return nil, err
	}

	file, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	if filepath.Ext(path) == ".jsonnet" {
		vm := jsonnet.MakeVM()

		json, err := vm.EvaluateAnonymousSnippet(path, string(file))
		if err != nil {
			return nil, err
		}

		file = []byte(json)
	}

	if err := yaml.Unmarshal(file, &config); err != nil {
		return nil, err
	}

	if config.Dir == "" {
		config.Dir = filepath.Dir(path)
	}

	return &config, nil
}
