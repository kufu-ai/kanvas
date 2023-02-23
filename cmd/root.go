package cmd

import (
	"kanvas/app"
	"kanvas/build"

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
	cmd.PersistentFlags().StringVarP(&configFile, "config", "f", "kanvas.yaml", "The path to the config file that declares the deployment workflow")

	diff := &cobra.Command{
		Use:   "diff",
		Short: "Shows the diff between the desired state and the current state",
		RunE: func(cmd *cobra.Command, args []string) error {
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
			return run(cmd, configFile, func(a *app.App) error {
				return a.Apply()
			})
		},
	}
	cmd.AddCommand(apply)

	var (
		exportDir string
	)
	exportActions := &cobra.Command{
		Use:   "export",
		Short: "Export the apply and the diff workflows to GitHub Actions",
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cmd, configFile, func(a *app.App) error {
				return a.Export(exportDir)
			})
		},
	}
	exportActions.Flags().StringVarP(&exportDir, "dir", "d", "", "Writes the exported workflow definitions to this directory")
	cmd.AddCommand(exportActions)

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
