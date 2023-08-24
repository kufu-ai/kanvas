package configai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"kanvas/openaichat"
	"os"
	"sync"
)

var APIKey = os.Getenv("OPENAI_API_KEY")

const (
	TEMPLATE = `You are who recommends a configuration for a tool called kanvas. Kanvas is the tool that abstracts the usages of various Infra-as-Code toolks like Terraform, Docker, ArgoCD. Kanvas's configuration file is named kanvas.yaml. A reference kanvas.yaml looks like the below.

--start of kanvas.yaml--

components:
	product1:
	components:
		appimage:
		dir: /containerimages/app
		docker:
			image: "davinci-std/example:myownprefix-"
		base:
		dir: /tf2
		needs:
		- appimage
		terraform:
			target: null_resource.eks_cluster
			vars:
			- name: containerimage_name
			valueFrom: appimage.id
		argocd:
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
		r, err := c.client.Complete(messages, funcs, openaichat.WithLog(out))
		if err != nil {
			return nil, err
		}

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
