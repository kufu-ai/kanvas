package kanvas

import (
	"os"

	"github.com/goccy/go-yaml"
)

type Component struct {
	Dir        string               `yaml:"dir"`
	Components map[string]Component `yaml:"components"`
	Needs      []string             `yaml:"needs"`
	Docker     *Docker              `yaml:"docker,omitempty"`
	Terraform  *Terraform           `yaml:"terraform,omitempty"`
}

type Docker struct {
	Image string `yaml:"image"`
	File  string `yaml:"file"`
}

type Terraform struct {
	Target string `yaml:"target"`
	Vars   []Var  `yaml:"var"`
}

type Var struct {
	Name      string `yaml:"name"`
	ValueFrom string `yaml:"valueFrom"`
}

func New(path string) (*Component, error) {
	var (
		config Component
	)

	file, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	if err := yaml.Unmarshal(file, &config); err != nil {
		return nil, err
	}

	return &config, nil
}
