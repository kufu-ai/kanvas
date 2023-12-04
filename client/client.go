package client

import (
	"encoding/json"
	"strings"

	kargotools "github.com/mumoshu/kargo/tools"
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

type Output struct {
	// PullRequest is the pull request that was created by the apply command, if any.
	PullRequest *kargotools.PullRequest `json:"pullRequest"`
}

func (r *ApplyResult) UnmarshalJSON(b []byte) error {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}

	r.Outputs = map[string]Output{}

	for k, v := range m {
		var o Output
		if err := json.Unmarshal(v, &o); err != nil {
			return err
		}
		r.Outputs[k] = o
	}

	return nil
}

type DiffResult struct {
}
