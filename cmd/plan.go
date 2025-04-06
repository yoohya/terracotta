package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/yoohya/terracotta/config"
	"github.com/yoohya/terracotta/terraform"
)

type planResult struct {
	Module string
	Error  error
}

var planCmd = &cobra.Command{
	Use:   "plan",
	Short: "Plan Terraform modules",
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

		var results []planResult

		for _, mod := range sortedModules {
			modulePath := filepath.Join(cfg.BasePath, mod.Path)
			fmt.Printf("[%s] INIT (%s)\n", mod.Path, modulePath)
			if err := terraform.RunCommand(modulePath, "init", "-input=false"); err != nil {
				fmt.Printf("[%s] Error running init: %v\n", mod.Path, err)
				results = append(results, planResult{Module: mod.Path, Error: fmt.Errorf("init failed: %v", err)})
				continue
			}

			fmt.Printf("[%s] PLAN (%s)\n", mod.Path, modulePath)
			if err := terraform.RunCommand(modulePath, "plan"); err != nil {
				fmt.Printf("[%s] Error running plan: %v\n", mod.Path, err)
				results = append(results, planResult{Module: mod.Path, Error: fmt.Errorf("plan failed: %v", err)})
				continue
			}

			results = append(results, planResult{Module: mod.Path, Error: nil})
		}

		fmt.Println("\nPlan Summary:")
		var failed bool
		for _, res := range results {
			if res.Error != nil {
				fmt.Printf("✖ %s: %v\n", res.Module, res.Error)
				failed = true
			} else {
				fmt.Printf("✔ %s: plan succeeded\n", res.Module)
			}
		}
		if failed {
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(planCmd)
	planCmd.Flags().StringVarP(&configPath, "config", "c", "terracotta.yaml", "Path to config file")
}
