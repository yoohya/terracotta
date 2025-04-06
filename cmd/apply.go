package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/yoohya/terracotta/config"
	"github.com/yoohya/terracotta/terraform"
)

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply Terraform modules for a specified environment",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.LoadConfig(configPath)
		if err != nil {
			fmt.Printf("Failed to load config: %v\n", err)
			os.Exit(1)
		}

		for _, mod := range cfg.Modules {
			modulePath := filepath.Join("environments", cfg.Environment, mod.Service, mod.Name)
			fmt.Printf("[APPLY] %s (%s)\n", mod.Name, modulePath)
			err := terraform.RunCommand(modulePath, "apply", "-auto-approve")
			if err != nil {
				fmt.Printf("Error running apply for %s: %v\n", mod.Name, err)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(applyCmd)
	applyCmd.Flags().StringP("env", "e", "", "Specify the environment (required)")
	applyCmd.MarkFlagRequired("env")
	applyCmd.Flags().StringVarP(&configPath, "config", "c", "terracotta.yaml", "Path to config file")
}
