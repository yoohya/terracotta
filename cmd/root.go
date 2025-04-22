package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var configPath string
var awsProfile string

var rootCmd = &cobra.Command{
	Use:   "terracotta",
	Short: "Terracotta is a lightweight Terraform module orchestrator",
	Long:  `Terracotta helps you plan and apply multiple Terraform modules in order, based on configuration.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Try 'terracotta plan --env dev' or 'terracotta apply --env dev'")
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
