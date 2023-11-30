package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"kanvas"
	"kanvas/app"
	"kanvas/build"
	"kanvas/plugin"

	kargotools "github.com/mumoshu/kargo/tools"

	"github.com/spf13/cobra"
)

func Root() *cobra.Command {
	var (
		opts = kanvas.Options{
			SkippedJobsOutputs: map[string]map[string]string{},
		}
	)

	cmd := &cobra.Command{
		Use:     "kanvas",
		Short:   "A container-based application deployer",
		Version: build.Version(),
	}
	cmd.PersistentFlags().StringVarP(&opts.Env, "env", "e", "", "The environment to deploy to")
	cmd.PersistentFlags().StringVarP(&opts.ConfigFile, "config", "c", kanvas.DefaultConfigFileYAML, "The path to the config file that declares the deployment workflow")

	new := &cobra.Command{
		Use:   "new",
		Short: "Creates a new kanvas.yaml file",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			return app.NewConfig(&opts)
		},
	}
	new.Flags().BoolVarP(&opts.UseAI, "use-ai", "a", false, "Use AI to suggest a kanvas.yaml file content based on your environment")
	cmd.AddCommand(new)

	diff := &cobra.Command{
		Use:   "diff",
		Short: "Shows the diff between the desired state and the current state",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			return run(cmd, opts, func(a *app.App) error {
				return a.Diff()
			})
		},
	}
	diff.Flags().StringSliceVar(&opts.Skip, "skip", nil, "Skip the specified component(s) when diffing changes")
	diff.Flags().Var(&JSONFlag{&opts.SkippedJobsOutputs}, "skipped-jobs-outputs", "The outputs from the skipped jobs. Needed for the jobs that depend on the skipped jobs")
	cmd.AddCommand(diff)

	apply := &cobra.Command{
		Use:   "apply",
		Short: "Build the container image(s) if any and runs terraform-apply command(s) to deploy changes",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			return run(cmd, opts, func(a *app.App) error {
				return a.Apply()
			})
		},
	}
	apply.Flags().BoolVar(&opts.LogsFollow, "logs-follow", false, "Follow log output from the components")
	apply.Flags().StringSliceVar(&opts.Skip, "skip", nil, "Skip the specified component(s) when applying changes")
	apply.Flags().Var(&JSONFlag{&opts.SkippedJobsOutputs}, "skipped-jobs-outputs", "The outputs from the skipped jobs. Needed for the jobs that depend on the skipped jobs")
	cmd.AddCommand(apply)

	{
		var (
			exportDir            string
			format               string
			kanvasContainerImage string
		)
		export := &cobra.Command{
			Use:   "export",
			Short: "Export the apply and the diff workflows to GitHub Actions",
			RunE: func(cmd *cobra.Command, args []string) error {
				return run(cmd, opts, func(a *app.App) error {
					cmd.SilenceUsage = true
					return a.Export(format, exportDir, kanvasContainerImage)
				})
			},
		}
		export.Flags().StringVarP(&format, "format", "f", plugin.FormatDefault, fmt.Sprintf("Export workflows in this format. The only supported value is %q", plugin.FormatGitHubActions))
		export.Flags().StringVarP(&exportDir, "dir", "d", "", "Writes the exported workflow definitions to this directory")
		export.Flags().StringVarP(&kanvasContainerImage, "kanvas-container-image", "i", "kanvas:example", "Use this image for running kanvas-related commands within GitHub Actions workflow job(s)")
		cmd.AddCommand(export)
	}

	{
		var (
			target string
			op     string
			format string
		)
		output := &cobra.Command{
			Use:   "output",
			Short: "Writes or saves the outputs from the specified job",
			RunE: func(cmd *cobra.Command, args []string) error {
				return run(cmd, opts, func(a *app.App) error {
					return a.Output(format, op, target)
				})
			},
		}
		output.Flags().StringVarP(&op, "op", "o", "", "Either diff or apply")
		output.Flags().StringVarP(&format, "format", "f", plugin.FormatDefault, fmt.Sprintf("Write outputs in this format. The only supported value is %q", plugin.FormatGitHubActions))
		output.Flags().StringVarP(&target, "target", "t", "", "Targeted job's name for collecting and writings outputs")
		cmd.AddCommand(output)
	}

	{
		tools := &cobra.Command{
			Use:    "tools",
			Hidden: true,
		}

		var opts kargotools.CreatePullRequestOptions
		createPullRequest := &cobra.Command{
			Use: kargotools.CommandCreatePullRequest,
			RunE: func(cmd *cobra.Command, args []string) error {
				cmd.SilenceUsage = true
				r, err := kargotools.CreatePullRequest(
					context.Background(),
					opts,
				)
				if err != nil {
					return err
				}
				j, err := json.MarshalIndent(r, "", "  ")
				if err != nil {
					return err
				}
				fmt.Println(string(j))
				return nil
			},
		}
		createPullRequest.Flags().StringVar(&opts.TokenEnv, kargotools.FlagCreatePullRequestTokenEnv, "", "The env var from which the GitHub token to use for creating the pull request is obtained")
		createPullRequest.Flags().StringVar(&opts.Base, kargotools.FlagCreatePullRequestBase, "", "The base branch to create the pull request against")
		createPullRequest.Flags().StringVar(&opts.Head, kargotools.FlagCreatePullRequestHead, "", "The head branch to create the pull request from")
		createPullRequest.Flags().StringVar(&opts.Title, kargotools.FlagCreatePullRequestTitle, "", "The title of the pull request")
		createPullRequest.Flags().StringVar(&opts.Body, kargotools.FlagCreatePullRequestBody, "", "The body of the pull request")
		createPullRequest.Flags().StringVar(&opts.Dir, kargotools.FlagCreatePullRequestDir, "", "The directory to run git commands in")
		createPullRequest.Flags().BoolVar(&opts.DryRun, kargotools.FlagCreatePullRequestDryRun, false, "Whether to run the command in dry-run mode")

		tools.AddCommand(createPullRequest)

		cmd.AddCommand(tools)
	}

	return cmd
}

func run(cmd *cobra.Command, opts kanvas.Options, do func(*app.App) error) error {
	app, err := app.New(opts)
	if err != nil {
		cmd.SilenceUsage = true
		return err
	}

	return do(app)
}
