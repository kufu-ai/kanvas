package cmd

import (
	"fmt"
	"kanvas/app"
	"kanvas/build"
	"kanvas/plugin"

	"github.com/spf13/cobra"
)

func Root() *cobra.Command {
	var (
		configFile string
	)

	cmd := &cobra.Command{
		Use:     "kanvas",
		Short:   "A container-based application deployer",
		Version: build.Version(),
	}
	cmd.PersistentFlags().StringVarP(&configFile, "config", "c", "kanvas.yaml", "The path to the config file that declares the deployment workflow")

	diff := &cobra.Command{
		Use:   "diff",
		Short: "Shows the diff between the desired state and the current state",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			return run(cmd, configFile, func(a *app.App) error {
				return a.Diff()
			})
		},
	}
	cmd.AddCommand(diff)

	apply := &cobra.Command{
		Use:   "apply",
		Short: "Build the container image(s) if any and runs terraform-apply command(s) to deploy changes",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			return run(cmd, configFile, func(a *app.App) error {
				return a.Apply()
			})
		},
	}
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
				return run(cmd, configFile, func(a *app.App) error {
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
			format string
		)
		output := &cobra.Command{
			Use:   "output",
			Short: "Writes or saves the outputs from the specified job",
			RunE: func(cmd *cobra.Command, args []string) error {
				return run(cmd, configFile, func(a *app.App) error {
					return a.Output(format, target)
				})
			},
		}
		output.Flags().StringVarP(&format, "format", "f", plugin.FormatDefault, fmt.Sprintf("Write outputs in this format. The only supported value is %q", plugin.FormatGitHubActions))
		output.Flags().StringVarP(&target, "target", "t", "", "Targeted job's name for collecting and writings outputs")
		cmd.AddCommand(output)
	}

	return cmd
}

func run(cmd *cobra.Command, configFile string, do func(*app.App) error) error {
	app, err := app.New(configFile)
	if err != nil {
		cmd.SilenceUsage = true
		return err
	}

	return do(app)
}
