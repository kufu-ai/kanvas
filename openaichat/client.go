package openaichat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/r3labs/sse/v2"
	"golang.org/x/sync/errgroup"
)

// Client is a client for the OpenAI Chat API.
type Client struct {
	APIKey string
}

type Result struct {
	Choice Choice
}

type SSEResult struct {
	Choices []Choice
}

type Config struct {
	Log io.Writer
}

type Option func(*Config)

func WithLog(log io.Writer) Option {
	return func(c *Config) {
		c.Log = log
	}
}

func (c *Client) Complete(messages []Message, funcs []Function, opts ...Option) (*Result, error) {
	client := c.newHTTPClient(false)

	var config Config
	for _, opt := range opts {
		opt(&config)
	}

	logOut := config.Log

	reqBody := ChatCompletionRequest{
		Messages:  messages,
		Model:     "gpt-3.5-turbo-16k",
		Stream:    false,
		Functions: funcs,
	}
	reqBodyJSON, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request body: %w", err)
	}

	req, err := http.NewRequestWithContext(context.Background(), "POST", APIEndpoint, bytes.NewBuffer([]byte(reqBodyJSON)))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	resp, err := client.Do(req)

	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}

	var chatCompletionRes ChatCompletionResponse
	var buf bytes.Buffer

	err = json.NewDecoder(io.TeeReader(resp.Body, &buf)).Decode(&chatCompletionRes)
	if err != nil {
		return nil, fmt.Errorf("error decoding response body: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code: %d: %s", resp.StatusCode, *chatCompletionRes.Error)
	}

	if logOut != nil {
		fmt.Fprintf(logOut, "%s\n", buf.String())
	}

	if l := len(chatCompletionRes.Choices); l != 1 {
		panic(fmt.Errorf("unexpected number of choices: %d", l))
	}

	choice := chatCompletionRes.Choices[0]

	if choice.FinishReason == "" {
		return nil, fmt.Errorf("finish reason: %s", choice.FinishReason)
	}

	return &Result{
		Choice: choice,
	}, nil
}

func (c *Client) SSE(messages []Message, funcs []Function, opts ...Option) (*SSEResult, error) {
	client := c.newHTTPClient(true)

	var config Config
	for _, opt := range opts {
		opt(&config)
	}

	logOut := config.Log

	reqBody := ChatCompletionRequest{
		Messages:  messages,
		Model:     "gpt-3.5-turbo",
		Stream:    true,
		Functions: funcs,
	}
	reqBodyJSON, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request body: %w", err)
	}

	sseClient := sse.NewClient(APIEndpoint)
	sseClient.Connection = client
	sseClient.Method = "POST"
	sseClient.Body = bytes.NewBuffer([]byte(reqBodyJSON))

	var eg errgroup.Group

	var errCh = make(chan error, 1)

	ctx, cancel := context.WithCancel(context.Background())

	var choices []Choice

	fmt.Fprintf(logOut, "Submitted the prompt to the AI...\n")

	eg.Go(func() error {
		defer close(errCh)
		return sseClient.SubscribeRawWithContext(ctx, func(msg *sse.Event) {
			var chatCompletionRes ChatCompletionResponse
			err := json.Unmarshal([]byte(msg.Data), &chatCompletionRes)
			if err != nil {
				if string(msg.Data) == "[DONE]" {
					return
				}

				fmt.Println(err)

				errCh <- err
				return
			}

			if logOut != nil {
				fmt.Fprintf(logOut, "AI is thinking: %s\n", string(msg.Data))
			}

			if l := len(chatCompletionRes.Choices); l != 1 {
				panic(fmt.Errorf("unexpected number of choices: %d", l))
			}

			choice := chatCompletionRes.Choices[0]

			choices = append(choices, choice)

			if choice.FinishReason != "" {
				cancel()
				return
			}
		})
	})

	eg.Go(func() error {
		err := <-errCh
		return err
	})

	if err := eg.Wait(); err != nil {
		return nil, err
	}

	return &SSEResult{
		Choices: choices,
	}, nil
}

func (c *Client) newHTTPClient(treatNon200Error bool) *http.Client {
	return &http.Client{
		Transport: &customTransport{
			RoundTripper:     http.DefaultTransport,
			BearerToken:      c.APIKey,
			TreatNon200Error: treatNon200Error,
		},
	}
}
