package ghrepos

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/google/go-github/v54/github"
	"golang.org/x/oauth2"
)

var (
	GitHubToken = os.Getenv("GITHUB_TOKEN")
	ProjectRoot = os.Getenv("KANVAS_WORKSPACE")
)

// Summarizer summarizes GitHub repositories
type Summarizer struct {
	GitHubToken string

	client *github.Client
	sync.Once
}

func (c *Summarizer) getGitHubClient() *github.Client {
	c.Once.Do(func() {
		if c.GitHubToken == "" {
			c.GitHubToken = GitHubToken
		}
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: c.GitHubToken},
		)
		ctx := context.Background()
		tc := oauth2.NewClient(ctx, ts)

		c.client = github.NewClient(tc)
	})

	return c.client
}

// Summary is a summary of GitHub repositories possibly related to the target repository.
// It's used as a context to let the AI suggest a kanvas.yaml file content
// based on your environment.
type Summary struct {
	// Repos is a list of GitHub repositories possibly related to the target repository.
	Repos []string
	// Contents is a list of summaries of the repositories.
	// Each summary starts with the repository name followed by a list of paths to files.
	Contents []string
}

func (c *Summarizer) Summarize(workspace string) (*Summary, error) {
	if workspace == "" {
		workspace = ProjectRoot
	}

	contents, err := c.getPossiblyRelatedRepoContents(workspace)
	if err != nil {
		return nil, err
	}

	var summary Summary

	for _, content := range contents {
		summary.Repos = append(summary.Repos, content.Repo.GetName())

		var contentStr string
		contentStr += fmt.Sprintf("%s\n", content.Repo.GetName())
		for _, file := range content.Files {
			contentStr += fmt.Sprintf("  %s\n", file)
		}

		summary.Contents = append(summary.Contents, contentStr)
	}

	return &summary, nil
}

func (c *Summarizer) getPossiblyRelatedRepoContents(workspace string) ([]RepoContent, error) {
	ctx := context.Background()

	client := c.getGitHubClient()

	r, err := git.PlainOpen(workspace)
	if err != nil {
		return nil, err
	}

	remotes, err := r.Remotes()
	if err != nil {
		return nil, err
	}

	var origin *config.RemoteConfig
	for _, remote := range remotes {
		if remote.Config().Name == "origin" {
			origin = remote.Config()
			break
		}
	}

	url := origin.URLs[0]
	// git@github.com:davinci-std/kajero.git"
	urlParts := strings.Split(url, ":")
	if len(urlParts) != 2 {
		return nil, fmt.Errorf("unexpected url: %s", url)
	}

	userRepoStr := urlParts[1]
	userRepo := strings.Split(userRepoStr, "/")
	if len(urlParts) != 2 {
		return nil, fmt.Errorf("unexpected url: %s", url)
	}

	org := userRepo[0]

	var (
		allRepos []*github.Repository
		nextPage *int
	)

	for {
		opts := github.RepositoryListByOrgOptions{}

		if nextPage != nil {
			opts.ListOptions.Page = *nextPage
		}

		// List all repositories by the organization of the target repository
		// to collect possibly related repositories.
		//
		// Note that List doesn't work as it's tied to the user endnpoint.
		// We need to use ListByOrg assuming everyone uses organization repositories...
		//
		// Do also note that GitHub classic tokens doesn't work probably when you aren&t an admin of
		// the organization.
		// GitHub fine-grained tokens work fine after an admin of the organization approved your token.
		// Note that the approval needs manual operation.
		//
		// See https://github.com/google/go-github/issues/2396#issuecomment-1181176636
		repos, res, err := client.Repositories.ListByOrg(ctx, org, &opts)
		if err != nil {
			return nil, err
		}

		allRepos = append(allRepos, repos...)

		if res.NextPage == 0 {
			break
		}

		if res.NextPage != 0 {
			nextPage = &res.NextPage
		}

		time.Sleep(1 * time.Second)
	}

	for _, r := range allRepos {
		fmt.Fprintf(os.Stderr, "Summarizing repository: %s\n", r.GetName())
	}

	var possibilyRelevantRepos []*github.Repository
	for _, repo := range allRepos {
		// Skip the repository if it's the same as the target repository
		if repo.GetName() == userRepo[1] {
			continue
		}

		// Skip the repository if it's archived
		if repo.GetArchived() {
			continue
		}

		// Skip the repository if it's a fork
		if repo.GetFork() {
			continue
		}

		// Skip the repository if it's a template
		if repo.GetTemplateRepository() != nil {
			continue
		}

		// Skip the repository if it's a mirror
		if repo.GetMirrorURL() != "" {
			continue
		}

		// Skip the repository if it's a disabled
		if repo.GetDisabled() {
			continue
		}

		possibilyRelevantRepos = append(possibilyRelevantRepos, repo)
	}

	contents, err := c.getRepoContents(possibilyRelevantRepos)
	if err != nil {
		return nil, err
	}

	return contents, nil
}

type RepoContent struct {
	Repo  *github.Repository
	Files []string
}

func (c *Summarizer) getRepoContents(repos []*github.Repository) ([]RepoContent, error) {
	client := c.getGitHubClient()

	numRepos := len(repos)

	var repoContents []RepoContent
	for i, repo := range repos {
		// Get the tree for the master branch
		tree, res, err := client.Git.GetTree(context.Background(), repo.GetOwner().GetLogin(), repo.GetName(), *repo.DefaultBranch, true)

		// Maybe "409 Repopsitory is empty" which means the repository is literally empty
		// and have no files yet.
		// This needs to be earlier than the error check because
		// it is also an error.
		if res != nil && res.StatusCode == 409 {
			fmt.Fprintf(os.Stderr, "  Skipping empty repository: %s\n", repo.GetName())
			continue
		}

		fmt.Fprintf(os.Stderr, "  Summarizing repository: %s (%d/%d)\n", repo.GetName(), i+1, numRepos)

		if err != nil {
			return nil, err
		}

		numEntries := len(tree.Entries)
		var files []string
		for j, entry := range tree.Entries {
			fmt.Fprintf(os.Stderr, "    Processing tree entry: %s (%d/%d)\n", entry.GetPath(), j+1, numEntries)

			if entry.GetType() == "blob" {
				files = append(files, entry.GetPath())
			}
		}

		repoContents = append(repoContents, RepoContent{
			Repo:  repo,
			Files: files,
		})
	}

	return repoContents, nil
}
