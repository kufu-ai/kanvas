package configai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/davinci-std/kanvas/openaichat"
)

var APIKey = os.Getenv("OPENAI_API_KEY")

const (
	TEMPLATE = `You are who recommends a configuration for a tool called kanvas. Kanvas is the tool that abstracts the usages of various Infra-as-Code toolks like Terraform, Docker, ArgoCD. Kanvas's configuration file is named kanvas.yaml. A reference kanvas.yaml looks like the below.

--start of kanvas.yaml--

components:
  appimage:
    # The repo is the GitHub repository where files for the component is located.
    # This is usually omitted if the Dockerfile is located in the same repository as the kanvas.yaml.
    #repo: davinci-std/exampleapp
	# The dir is the directory where Dockerfile and the docker build context is located.
	# If the dockerfile is at container/images/app/Dockerfile, the dir is container/images/app.
    dir: containerimages/app
    docker:
	  # The image is the name and the tag prefix of the container image to be built and pushed by kanvas.
      image: "davinci-std/example:myownprefix-"
  # base is a common component that is used to represent the cloud infrastructure.
  # It often refers to a Terraform module that creates a Kubernetes cluster and its dependencies.
  base:
    # The repo is the GitHub repository where files for the component is located.
	# For the base component, the repo usually contains a Terraform module that creates a Kubernetes cluster and its dependencies.
	# It must be in ORG/REPO format.
	# If ORG is not present, you can omit it.
    repo: davinci-std/myinfra
	# The dir is the directory where files for the component is located.
	# This is the directory where the Terraform module for the specific environment (like development) is located.
	# If the *.tf files are located in the path/to/exampleapp/terraform/module/*.tf, the dir is path/to/exampleapp/terraform/module.
    dir: path/to/exampleapp/terraform/module
    needs:
    - appimage
    terraform:
      target: null_resource.eks_cluster
      vars:
      - name: containerimage_name
        valueFrom: appimage.id
  # argocd component exists only when you deploy your apps to Kubernetes clusters
  # using ArgoCD, and you want to manage ArgoCD resources using kanvas.
  # If you don't use ArgoCD, you can omit this component.
  argocd:
    # This is the directory where the Terraform module for ArgoCD is located.
    dir: /tf2
    needs:
    - base
    terraform:
      target: aws_alb.argocd_api
      vars:
      - name: cluster_endpoint
        valueFrom: base.cluster_endpoint
      - name: cluster_token
        valueFrom: base.cluster_token
  argocd_resources:
    # only path relative to where the command has run is supported
    # maybe the top-project-level "dir" might be supported in the future
    # which in combination with the relative path support for sub-project dir might be handy for DRY
    dir: /tf2
    needs:
    - argocd
    terraform:
      target: argocd_application.kanvas

--end of kanvas.yaml--

That said, please suggest a kanvas.yaml that fits my use-case.
%s

Here are relevant information:

We have repositories with following contents in the *nix tree command style:

Repositories:

%s

Contents:
%s
`
)

type ConfigRecommender struct {
	APIKey string

	once   sync.Once
	client *openaichat.Client
}

type SuggestOptions struct {
	// If true, the recommender will ask questions to the user.
	DoAsk bool
	// If true, the recommender will use this API key instead of the default one.
	APIKey string
	// If true, the recommender will use functions.
	UseFun bool
	// If true, the recommender will use SSE.
	SSE bool
	// If true, the recommender will use this writer to log.
	Log io.Writer
}

type SuggestOption func(*SuggestOptions)

func WithDoAsk(doAsk bool) SuggestOption {
	return func(o *SuggestOptions) {
		o.DoAsk = doAsk
	}
}

func WithAPIKey(apiKey string) SuggestOption {
	return func(o *SuggestOptions) {
		o.APIKey = apiKey
	}
}

func WithUseFun(useFun bool) SuggestOption {
	return func(o *SuggestOptions) {
		o.UseFun = useFun
	}
}

func WithSSE(sse bool) SuggestOption {
	return func(o *SuggestOptions) {
		o.SSE = sse
	}
}

func WithLog(log io.Writer) SuggestOption {
	return func(o *SuggestOptions) {
		o.Log = log
	}
}

func (c *ConfigRecommender) Suggest(repos, contents string, opt ...SuggestOption) (*string, error) {
	c.once.Do(func() {
		if c.APIKey == "" {
			c.APIKey = APIKey
		}
		c.client = &openaichat.Client{APIKey: c.APIKey}
	})

	var opts SuggestOptions

	for _, o := range opt {
		o(&opts)
	}

	var (
		doAsk, useFun, sse bool
		out                io.Writer
	)

	doAsk = opts.DoAsk
	useFun = opts.UseFun
	sse = opts.SSE
	out = opts.Log

	var conds string
	if doAsk {
		conds = "Please ask questions to me if needed."
	}
	content := fmt.Sprintf(TEMPLATE, conds, repos, contents)

	messages := []openaichat.Message{
		{Role: "user", Content: content},
	}

	var funcs []openaichat.Function
	if useFun {
		funcs = []openaichat.Function{
			{
				Name:        "this_is_the_kanvas_yaml",
				Description: "Send the kanvas.yaml you generated to the user. You suggest and send the yaml to me by calling function. Don't use this to let me(user) suggest a kanvas.yaml. It's you(AI)'s job to suggest and send the yaml.",
				Parameters: openaichat.FunctionParameters{
					Type: "object",
					Properties: map[string]openaichat.FunctionParameterProperty{
						"kanvas_yaml": {
							Type:        "string",
							Description: "Generated kanvas.yaml",
						},
					},
				},
			},
		}
	}

	if !sse {
		fmt.Fprintf(os.Stderr, "Starting to generate kanvas.yaml...\n")

		// Measure the time to complete the request.

		start := time.Now()

		r, err := c.client.Complete(messages, funcs, openaichat.WithLog(out))
		if err != nil {
			return nil, err
		}

		end := time.Now()

		// Print the time to complete the request.
		duration := end.Sub(start)
		fmt.Fprintf(os.Stderr, "Completed to generate kanvas.yaml in %s\n", duration)

		if useFun {
			if r.Choice.FinishReason != "function_call" {
				return nil, fmt.Errorf("unexpected finish reason: %s", r.Choice.FinishReason)
			}

			if r.Choice.Message.FunctionCall == nil {
				return nil, fmt.Errorf("unexpected missing function call")
			}

			fc := r.Choice.Message.FunctionCall

			if fc.Name != "this_is_the_kanvas_yaml" {
				return nil, fmt.Errorf("unexpected function name: %s", fc.Name)
			}

			funRes, err := parseFunctionResult(bytes.NewBufferString(*fc.Arguments))
			if err != nil {
				return nil, err
			}

			return &funRes.KanvasYAML, nil
		}

		return &r.Choice.Message.Content, nil
	}

	res, err := c.client.SSE(messages, funcs, openaichat.WithLog(out))
	if err != nil {
		return nil, err
	}

	var (
		currentFunName string
		generatedYAML  string
	)
	if useFun {
		var jsonBuf bytes.Buffer
		for _, c := range res.Choices {
			if c.Delta == nil {
				panic("unexpected missing delta")
			}
			d := c.Delta
			fc := d.FunctionCall
			if fc != nil {
				if currentFunName == "" {
					currentFunName = fc.Name
				} else if fc.Name != "" {
					currentFunName = fc.Name
				}

				if currentFunName != "this_is_the_kanvas_yaml" {
					panic("unexpected funname " + currentFunName)
				}
				jsonBuf.WriteString(*fc.Arguments)
			}
		}
		funRes, err := parseFunctionResult(&jsonBuf)
		if err != nil {
			return nil, err
		}
		generatedYAML = funRes.KanvasYAML
	}

	return &generatedYAML, nil
}

type FunRes struct {
	KanvasYAML string `json:"kanvas_yaml"`
}

func parseFunctionResult(buf *bytes.Buffer) (*FunRes, error) {
	var funRes FunRes
	err := json.Unmarshal(buf.Bytes(), &funRes)
	if err != nil {
		return nil, err
	}

	return &funRes, nil
}
