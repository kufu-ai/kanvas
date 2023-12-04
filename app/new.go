package app

import (
	"fmt"
	"os"
	"strings"

	"github.com/davinci-std/kanvas/ghrepos"

	"github.com/davinci-std/kanvas/configai"

	"github.com/davinci-std/kanvas"

	"github.com/projectdiscovery/yamldoc-go/encoder"
)

// New generates a new kanvas.yaml from all the comments and the default settings.
func NewConfig(opts *kanvas.Options) error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	args := generateArgs{
		Dir: wd,
	}

	var data []byte

	if opts.UseAI {
		println("Using AI to generate kanvas.yaml...")
		data, err = generateConfigDataUsingAI(args)
		if err != nil {
			return err
		}
	} else {
		data, err = generateConfigData(args)
		if err != nil {
			return err
		}
	}

	fmt.Fprintf(os.Stdout, "%s", string(data))

	f := opts.GetConfigFilePath()
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

func generateConfigData(args generateArgs) ([]byte, error) {
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

func generateConfigDataUsingAI(args generateArgs) ([]byte, error) {
	c := &configai.ConfigRecommender{}

	projectRoot := args.Dir

	s := &ghrepos.Summarizer{}
	summary, err := s.Summarize(projectRoot)
	if err != nil {
		return nil, err
	}

	contents := strings.Join(summary.Contents, "\n")
	repos := strings.Join(summary.Repos, "\n")

	kanvasConfigYAML, err := c.Suggest(string(contents), string(repos), configai.WithUseFun(true), configai.WithLog(os.Stderr))
	if err != nil {
		if os.Getenv("KANVAS_DEBUG") != "" {
			// Dumps the contents and repos to respective files for debugging
			fmt.Fprintf(os.Stderr, "Writing contents.txt and repos.txt for debugging...\n")
			if err := os.WriteFile("contents.txt", []byte(contents), 0644); err != nil {
				return nil, fmt.Errorf("error writing contents.txt: %w", err)
			}
			if err := os.WriteFile("repos.txt", []byte(repos), 0644); err != nil {
				return nil, fmt.Errorf("error writing repos.txt: %w", err)
			}
		}
		return nil, err
	}

	return []byte(*kanvasConfigYAML), nil
}
