package cmd

import (
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/yoohya/terracotta/config"
)

var planDestroy bool

var planCmd = &cobra.Command{
	Use:   "plan",
	Short: "Plan Terraform modules",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, sortedModules := prepareModules()

		if runPlan(os.Stdout, cfg, sortedModules, planDestroy, upgradeProviders) {
			os.Exit(1)
		}
	},
}

// runPlan は各モジュールで plan を実行し、失敗があれば true を返す。
// plan はすべてのモジュールの問題をまとめて把握できるよう、失敗しても続行する。
// destroy が true の場合は destroy コマンドと同じく依存の逆順で plan -destroy を実行する。
func runPlan(out io.Writer, cfg *config.Config, sortedModules []*config.ModuleNode, destroy bool, upgrade bool) bool {
	modules := sortedModules
	actionArgs := []string{"plan"}
	title := "Plan"
	if destroy {
		modules = reverseModules(sortedModules)
		actionArgs = []string{"plan", "-destroy"}
		title = "Destroy Plan"
	}

	results := runModules(out, cfg, modules, actionArgs, upgrade, continueOnFailure)
	return printSummary(out, title, modules, results, "plan succeeded")
}

func init() {
	rootCmd.AddCommand(planCmd)
	planCmd.Flags().StringVarP(&configPath, "config", "c", "terracotta.yaml", "Path to config file")
	planCmd.Flags().StringVar(&awsProfile, "profile", "", "AWS profile to use")
	planCmd.Flags().BoolVar(&upgradeProviders, "upgrade", false, "Upgrade providers to the latest version")
	planCmd.Flags().BoolVar(&planDestroy, "destroy", false, "Plan destruction of all modules in reverse dependency order")
}
