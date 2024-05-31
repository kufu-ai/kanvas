//go:generate docgen kanvas.go kanvas_doc.go Component

package kanvas

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

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

	// AWS is an AWS-specific configuration
	// This is currently used to ensure that you have the right AWS credentials
	// that are required to access resources such as ECR and EKS.
	AWS *AWS `yaml:"aws,omitempty"`

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
	// Noop is a noop configuration that does nothing
	// This is mainly for template components that are only used as dependencies.
	// You override or replaces this with a real component in the environment.
	Noop *Noop `yaml:"noop,omitempty"`
}

func (c *Component) Validate() error {
	if c.Docker == nil && c.Terraform == nil && c.Kubernetes == nil && c.AWS == nil && c.Externals == nil &&
		c.Noop == nil {
		return errors.New("component does not have any of docker, terraform, kubernetes, aws, externals, or noop fields")
	}
	return nil
}

// Environment is a set of sub-components to replace the defaults
type Environment struct {
	// Defaults is the environment-specific defaults
	Defaults Component `yaml:"defaults,omitempty"`
	// Uses is a set of sub-components to replace the defaults
	Uses map[string]Component `yaml:"uses,omitempty"`
	// Overrides is a set of sub-components to override the env and component defaults
	Overrides map[string]Component `yaml:"overrides,omitempty"`
}

// Docker is a docker-specific configuration
type Docker struct {
	// Image is the name of the image to be built
	Image string `yaml:"image"`
	// File is the path to the Dockerfile
	File string `yaml:"file"`
	// Args is a map of build args
	Args map[string]string `yaml:"args"`
	// ArgsFrom is a map of dynamic build args from the outputs of other components
	ArgsFrom map[string]string `yaml:"argsFrom"`
	// TagsFrom is a list of tags to be added to the image, derived from the outputs of other components
	TagsFrom []string `yaml:"tagsFrom"`
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
func LoadConfig(path string, file []byte) (*Component, error) {
	var (
		config Component
	)

	if err := yaml.Unmarshal(file, &config); err != nil {
		return nil, err
	}

	if config.Dir == "" {
		config.Dir = filepath.Dir(path)
	}

	return &config, nil
}

func RenderOrReadFile(path string) ([]byte, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	if filepath.Ext(path) == ".jsonnet" {
		vm := jsonnet.MakeVM()

		if strings.Contains(filepath.Base(path), ".template.") {
			vm.ExtVar("template", "true")

			if v := os.Getenv("GITHUB_REPOSITORY"); v != "" {
				splits := strings.Split(v, "/")
				vm.ExtVar("github_repo_owner", splits[0])
				vm.ExtVar("github_repo_name", splits[1])
			} else {
				return nil, errors.New(".template.jsonnet requires GITHUB_REPOSITORY to be set to OWNER/REPO_NAME for the template to access `std.extVar(\"github_repo_name\")` and `std.extVar(\"github_repo_owner\")`")
			}
		}

		json, err := vm.EvaluateAnonymousSnippet(path, string(file))
		if err != nil {
			return nil, err
		}

		file = []byte(json)
	}

	return file, nil
}
