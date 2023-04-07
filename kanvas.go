package kanvas

import (
	"os"

	"github.com/goccy/go-yaml"
	"github.com/mumoshu/kargo"
)

type Component struct {
	Dir        string               `yaml:"dir"`
	Components map[string]Component `yaml:"components"`
	Needs      []string             `yaml:"needs"`
	Docker     *Docker              `yaml:"docker,omitempty"`
	Terraform  *Terraform           `yaml:"terraform,omitempty"`
	Kubernetes *Kubernetes          `yaml:"kubernetes,omitempty"`
}

type Docker struct {
	Image string `yaml:"image"`
	File  string `yaml:"file"`
}

type Terraform struct {
	Target string `yaml:"target"`
	Vars   []Var  `yaml:"vars"`
}

type Var struct {
	Name      string `yaml:"name"`
	ValueFrom string `yaml:"valueFrom"`
}

type Kubernetes struct {
	kargo.Config `yaml:",inline"`
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
