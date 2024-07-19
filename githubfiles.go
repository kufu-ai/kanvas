package kanvas

import (
	"fmt"
	"log"
	"os"

	"github.com/mumoshu/gitimpart"
)

// GitHubFiles driver is a driver that pushes files and optionally run kustomize-build to/on the target repository/branch.
//
// The driver requires the following environment variables to be set:
// - GITHUB_TOKEN: a GitHub token with write access to the target repository
// - GITHUB_REPOSITORY: the target repository in the OWNER/REPO_NAME format
//
// Note that the changes are always pushed to a new branch and a pull request is created to merge the changes to the target branch,
// instead of pushing the changes directly to the target branch.
type GitHubFiles struct {
	// Path to the gitimpart jsonnet file to use
	Path string
	// Repo is the repository to push the changes to
	Repo string
	// Branch is the branch to push the changes to
	Branch string
}

func newGitHubFilesDriver(conf *GitHubFiles) (*Driver, error) {
	return &Driver{
		Diff: []Task{
			{
				Func: func(j *WorkflowJob, _ map[string]string) error {
					vars := getJsonnetVars()
					if len(vars) == 0 {
						return fmt.Errorf("GitHubFiles driver requires GITHUB_REPOSITORY to be set to OWNER/REPO_NAME for the template to access `std.extVar(\"github_repo_name\")` and `std.extVar(\"github_repo_owner\")`")
					}
					return runGitImpart(conf.Repo, conf.Branch, conf.Path, vars, true)
				},
			},
		},
		Apply: []Task{
			{
				Func: func(j *WorkflowJob, _ map[string]string) error {
					vars := getJsonnetVars()
					if len(vars) == 0 {
						return fmt.Errorf("GitHubFiles driver requires GITHUB_REPOSITORY to be set to OWNER/REPO_NAME for the template to access `std.extVar(\"github_repo_name\")` and `std.extVar(\"github_repo_owner\")`")
					}
					return runGitImpart(conf.Repo, conf.Branch, conf.Path, vars, false)
				},
			},
		},
		Output: nil,
		OutputFunc: func(r *Runtime, op Op, o map[string]string) error {
			return nil
		},
	}, nil
}

func runGitImpart(repo, branch string, configFile string, vars map[string]string, dryRun bool) error {
	const (
		ghTokenEnv = "GITHUB_TOKEN"
		// We currently assume that a pull request is always desired
		// to update the files in the target repository.
		pullRequest = true
	)
	ghtoken := os.Getenv(ghTokenEnv)
	if ghtoken == "" {
		log.Printf("GITHUB_TOKEN is not set. Access to private repositories will be denied unless you configure other means of authentication")
	}

	var loadOpts []gitimpart.LoadOption

	if len(vars) > 0 {
		loadOpts = append(loadOpts, gitimpart.Vars(vars))
	}

	r, err := gitimpart.RenderFile(configFile, loadOpts...)
	if err != nil {
		return fmt.Errorf("failed to render file %s: %v", configFile, err)
	}

	opts := []gitimpart.PushOptions{
		gitimpart.WithGitHubToken(ghtoken),
	}

	if dryRun {
		opts = append(opts, gitimpart.WithDryRun())
	}

	if pullRequest {
		opts = append(opts, gitimpart.WithPullRequest())
	}

	err = gitimpart.Push(
		*r,
		repo,
		branch,
		opts...,
	)
	if err != nil {
		return fmt.Errorf("failed to push the changes: %v", err)
	}

	if !dryRun {
		log.Printf("successfully pushed the changes to %s/%s", repo, branch)
	}

	return nil
}
