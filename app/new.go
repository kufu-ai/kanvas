package app

import (
	"fmt"
	"kanvas"
	"os"

	"github.com/projectdiscovery/yamldoc-go/encoder"
)

// New generates a new kanvas.yaml from all the comments and the default settings.
func (a *App) New() error {
	data, err := a.generateConfigData()
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "%s", string(data))

	f := a.Options.GetConfigPath()
	if stat, err := os.Stat(f); err == nil && !stat.IsDir() {
		return fmt.Errorf("file %q already exists", f)
	}

	if err := os.WriteFile(f, data, 0644); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Created %q\n", f)

	return nil
}

func (a *App) generateConfigData() ([]byte, error) {
	encoder := encoder.NewEncoder(&kanvas.Component{
		Components: map[string]kanvas.Component{
			"image": {
				Dir: "docker",
				Docker: &kanvas.Docker{
					Image: "examplecom/myapp",
					File:  "Dockerfile",
				},
			},
		},
	}, encoder.WithComments(encoder.CommentsAll))

	data, err := encoder.Encode()
	if err != nil {
		return nil, err
	}

	return data, nil
}
