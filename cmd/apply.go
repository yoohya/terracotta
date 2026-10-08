package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply Terraform modules for a specified environment",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, sortedModules := prepareModules()

		if runModulesFailFast(cfg, sortedModules, []string{"apply", "-auto-approve"}, "applied successfully") {
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(applyCmd)
	applyCmd.Flags().StringVarP(&configPath, "config", "c", "terracotta.yaml", "Path to config file")
	applyCmd.Flags().StringVar(&awsProfile, "profile", "", "AWS profile to use")
	applyCmd.Flags().BoolVar(&upgradeProviders, "upgrade", false, "Upgrade providers to the latest version")
}
