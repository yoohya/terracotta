package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/yoohya/terracotta/config"
	"github.com/yoohya/terracotta/terraform"
)

type applyResult struct {
	Module string
	Status string // "success", "failed", "skipped"
	Error  error
}

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply Terraform modules for a specified environment",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.LoadConfig(configPath)
		if err != nil {
			fmt.Printf("Failed to load config: %v\n", err)
			os.Exit(1)
		}

		graph, err := config.BuildExecutionGraph(cfg)
		if err != nil {
			fmt.Printf("Failed to build execution graph: %v\n", err)
			os.Exit(1)
		}

		sortedModules, err := graph.TopoSortedModules()
		if err != nil {
			fmt.Printf("Failed to resolve module order: %v\n", err)
			os.Exit(1)
		}

		var results []applyResult

		for _, mod := range sortedModules {
			modulePath := filepath.Join(cfg.BasePath, mod.Path)
			fmt.Printf("[%s] INIT (%s)\n", mod.Path, modulePath)
			if err := terraform.RunCommand(modulePath, "init", "-input=false"); err != nil {
				fmt.Printf("✖ [%s] Terraform init failed!\n", mod.Path)
				fmt.Printf("    Module path : %s\n", modulePath)
				fmt.Printf("    Command     : terraform init -input=false\n")
				fmt.Printf("    Error       : %v\n", err)
				results = append(results, applyResult{Module: mod.Path, Status: "failed", Error: fmt.Errorf("init failed: %v", err)})
				break
			}

			fmt.Printf("[%s] APPLY (%s)\n", mod.Path, modulePath)
			if err := terraform.RunCommand(modulePath, "apply", "-auto-approve"); err != nil {
				fmt.Printf("✖ [%s] Terraform apply failed!\n", mod.Path)
				fmt.Printf("    Module path : %s\n", modulePath)
				fmt.Printf("    Command     : terraform apply -auto-approve\n")
				fmt.Printf("    Error       : %v\n", err)
				results = append(results, applyResult{Module: mod.Path, Status: "failed", Error: fmt.Errorf("apply failed: %v", err)})
				break
			}

			results = append(results, applyResult{Module: mod.Path, Status: "success"})
		}

		fmt.Println("\nApply Summary:")
		encounteredFailure := false
		executed := map[string]bool{}
		for _, res := range results {
			executed[res.Module] = true
			switch res.Status {
			case "success":
				fmt.Printf("✔ %s: applied successfully\n", res.Module)
			case "failed":
				fmt.Printf("✖ %s: failed - %v\n", res.Module, res.Error)
				encounteredFailure = true
			}
		}
		for _, mod := range sortedModules {
			if !executed[mod.Path] {
				fmt.Printf("⏭ %s: skipped\n", mod.Path)
			}
		}
		if encounteredFailure {
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(applyCmd)
	applyCmd.Flags().StringVarP(&configPath, "config", "c", "terracotta.yaml", "Path to config file")
}
