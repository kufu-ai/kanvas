package app

import (
	"fmt"
	"kanvas"
	"os"

	"github.com/projectdiscovery/yamldoc-go/encoder"
)

// New generates a new kanvas.yaml from all the comments and the default settings.
func (a *App) New() error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	args := generateArgs{
		Dir: wd,
	}
	data, err := a.generateConfigData(args)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "%s", string(data))

	f := a.Options.GetConfigFilePath()
	if f == "" {
		f = kanvas.DefaultConfigFileYAML
	}

	if stat, err := os.Stat(f); err == nil && !stat.IsDir() {
		return fmt.Errorf("file %q already exists", f)
	} else if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("unable to stat %q: %w", f, err)
	}

	if err := os.WriteFile(f, data, 0644); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Created %q\n", f)

	return nil
}

type generateArgs struct {
	Dir string
}

func (a *App) generateConfigData(args generateArgs) ([]byte, error) {
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
