package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/yoohya/terracotta/config"
	"github.com/yoohya/terracotta/terraform"
)

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

		for _, mod := range sortedModules {
			modulePath := filepath.Join(cfg.BasePath, mod.Path)
			fmt.Printf("[INIT] %s (%s)\n", mod.Path, modulePath)
			if err := terraform.RunCommand(modulePath, "init", "-input=false"); err != nil {
				fmt.Printf("Error running init for %s: %v\n", mod.Path, err)
				os.Exit(1)
			}

			fmt.Printf("[PLAN] %s (%s)\n", mod.Path, modulePath)
			if err := terraform.RunCommand(modulePath, "plan"); err != nil {
				fmt.Printf("Error running plan for %s: %v\n", mod.Path, err)
				os.Exit(1)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(planCmd)
	planCmd.Flags().StringVarP(&configPath, "config", "c", "terracotta.yaml", "Path to config file")
}
