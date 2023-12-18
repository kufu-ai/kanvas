package client

import (
	"encoding/json"
	"strings"
)

const (
	EnvVarPullRequestAssigneeIDs = "KANVAS_PULLREQUEST_ASSIGNEE_IDS"
	EnvVarPullRequestHead        = "KANVAS_PULLREQUEST_HEAD"
)

type ApplyOptions struct {
	// SkippedComponents is a map of component name to its output.
	// You need the output of the component to be skipped for the components that depend on it.
	SkippedComponents map[string]map[string]string `json:"skippedComponents"`
	// PullRequestAssigneeIDs is the list of GitHub user IDs to assign to the pull request that will be created by the apply command.
	// Each ID can be either an integer or a string.
	PullRequestAssigneeIDs []string `json:"pullRequestAssigneeIDs"`
	// PullRequestHead is the branch to create the pull request from.
	PullRequestHead string `json:"pullRequestHead"`
	// EnvVars is the list of environment variables to set for the apply command.
	EnvVars map[string]string `json:"envVars"`
}

func (o *ApplyOptions) GetSkip() []string {
	var skipped []string
	for k := range o.SkippedComponents {
		skipped = append(skipped, k)
	}
	return skipped
}

func (o *ApplyOptions) GetSkippedComponents() map[string]map[string]string {
	return o.SkippedComponents
}

func (o *ApplyOptions) GetEnvVars() map[string]string {
	m := map[string]string{}
	for k, v := range o.EnvVars {
		m[k] = v
	}

	if o.PullRequestAssigneeIDs != nil {
		m[EnvVarPullRequestAssigneeIDs] = strings.Join(o.PullRequestAssigneeIDs, ",")
	}

	if o.PullRequestHead != "" {
		m[EnvVarPullRequestHead] = o.PullRequestHead
	}

	return m
}

type DiffOptions struct {
	// SkippedComponents is a map of component name to its output.
	// You need the output of the component to be skipped for the components that depend on it.
	SkippedComponents map[string]map[string]string `json:"skippedComponents"`
	// PullRequestAssigneeIDs is the list of GitHub user IDs to assign to the pull request that will be created by the apply command.
	// Each ID can be either an integer or a string.
	PullRequestAssigneeIDs []string `json:"pullRequestAssigneeIDs"`
	// PullRequestHead is the branch to create the pull request from.
	PullRequestHead string `json:"pullRequestHead"`
	// EnvVars is the list of environment variables to set for the apply command.
	EnvVars map[string]string `json:"envVars"`
}

func (o *DiffOptions) GetSkip() []string {
	var skipped []string
	for k := range o.SkippedComponents {
		skipped = append(skipped, k)
	}
	return skipped
}

func (o *DiffOptions) GetSkippedComponents() map[string]map[string]string {
	return o.SkippedComponents
}

func (o *DiffOptions) GetEnvVars() map[string]string {
	m := map[string]string{}
	for k, v := range o.EnvVars {
		m[k] = v
	}

	if o.PullRequestAssigneeIDs != nil {
		m[EnvVarPullRequestAssigneeIDs] = strings.Join(o.PullRequestAssigneeIDs, ",")
	}

	if o.PullRequestHead != "" {
		m[EnvVarPullRequestHead] = o.PullRequestHead
	}

	return m
}

type ApplyResult struct {
	Outputs map[string]Output `json:"-"`
}

// Any kanvas component and the provider used for the component
// may define its own output.
type Output struct {
	// PullRequest is the pull request that was created by the apply command.
	//
	// You can expect any current and future kanvas provider
	// that works with pull requests to produce the outputs for this field.
	PullRequest *PullRequest `json:"-"`
}

type PullRequest struct {
	// ID is the pull request ID.
	//
	// It is typed as "number" in the API response,
	// but due to that kanvas output is typed as string,
	// it is typed as string here.
	ID string `json:"pullRequest.id"`
	// NodeID is the pull request node ID.
	//
	// This is often used to identify the pull request when
	// using the GitHub GraphQL API.
	NodeID string `json:"pullRequest.nodeID"`
	// Number is the pull request number.
	//
	// It is typed as "number" in the API response,
	// but due to that kanvas output is typed as string,
	// it is typed as string here.
	Number  string `json:"pullRequest.number"`
	Head    string `json:"pullRequest.head"`
	HTMLURL string `json:"pullRequest.htmlURL"`
}

func (p PullRequest) IsEmpty() bool {
	return p.ID == "" && p.Number == "" && p.Head == "" && p.HTMLURL == ""
}

// GetPullRequests returns the list of pull requests that were created by the apply command.
//
// By the nature of the apply command, it may or may not create a pull request.
// It may even create multiple pull requests.
//
// It's up to the consumer of the kanvas output to decide what to do with the pull requests.
func (r *ApplyResult) GetPullRequests() []*PullRequest {
	var prs []*PullRequest
	for _, o := range r.Outputs {
		if o.PullRequest != nil {
			prs = append(prs, o.PullRequest)
		}
	}
	return prs
}

func (r *ApplyResult) UnmarshalJSON(b []byte) error {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}

	r.Outputs = map[string]Output{}

	for k, v := range m {
		var o Output

		var pr PullRequest
		if err := json.Unmarshal(v, &pr); err != nil {
			return err
		}
		if !pr.IsEmpty() {
			o.PullRequest = &pr
		}

		r.Outputs[k] = o
	}

	return nil
}

type DiffResult struct {
}
