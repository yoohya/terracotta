package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var planCmd = &cobra.Command{
	Use:   "plan",
	Short: "Plan Terraform modules",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, sortedModules := prepareModules()

		// plan はすべてのモジュールの問題をまとめて把握できるよう、失敗しても続行する
		results := runModules(os.Stdout, cfg, sortedModules, []string{"plan"}, upgradeProviders, continueOnFailure)
		if printSummary(os.Stdout, "Plan", sortedModules, results, "plan succeeded") {
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(planCmd)
	planCmd.Flags().StringVarP(&configPath, "config", "c", "terracotta.yaml", "Path to config file")
	planCmd.Flags().StringVar(&awsProfile, "profile", "", "AWS profile to use")
	planCmd.Flags().BoolVar(&upgradeProviders, "upgrade", false, "Upgrade providers to the latest version")
}
