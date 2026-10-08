package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yoohya/terracotta/config"
)

const destroyConfirmWord = "destroy"

var destroyAutoApprove bool

var destroyCmd = &cobra.Command{
	Use:   "destroy",
	Short: "Destroy Terraform modules in reverse dependency order",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, sortedModules := prepareModules()
		// 依存されている側を最後に消すため、apply と逆順で実行する
		modules := reverseModules(sortedModules)

		fmt.Println("The following modules will be DESTROYED in this order:")
		for i, mod := range modules {
			fmt.Printf("  %d. %s\n", i+1, mod.Path)
		}

		if !destroyAutoApprove {
			ok, err := confirmDestroy(os.Stdin, os.Stdout)
			if err != nil {
				fmt.Printf("Failed to read confirmation: %v\n", err)
				os.Exit(1)
			}
			if !ok {
				fmt.Println("Destroy cancelled.")
				return
			}
		}

		if runModulesFailFast(cfg, modules, []string{"destroy", "-auto-approve"}, "destroyed successfully") {
			os.Exit(1)
		}
	},
}

// reverseModules は元のスライスを変更せずに逆順のスライスを返す。
func reverseModules(modules []*config.ModuleNode) []*config.ModuleNode {
	reversed := make([]*config.ModuleNode, len(modules))
	for i, mod := range modules {
		reversed[len(modules)-1-i] = mod
	}
	return reversed
}

// confirmDestroy は確認ワードの入力を求め、完全一致した場合のみ true を返す。
func confirmDestroy(in io.Reader, out io.Writer) (bool, error) {
	if _, err := fmt.Fprintf(out, "\nType '%s' to confirm: ", destroyConfirmWord); err != nil {
		return false, err
	}
	line, err := bufio.NewReader(in).ReadString('\n')
	if err != nil && err != io.EOF {
		return false, err
	}
	return strings.TrimSpace(line) == destroyConfirmWord, nil
}

func init() {
	rootCmd.AddCommand(destroyCmd)
	destroyCmd.Flags().StringVarP(&configPath, "config", "c", "terracotta.yaml", "Path to config file")
	destroyCmd.Flags().StringVar(&awsProfile, "profile", "", "AWS profile to use")
	destroyCmd.Flags().BoolVar(&upgradeProviders, "upgrade", false, "Upgrade providers to the latest version")
	destroyCmd.Flags().BoolVar(&destroyAutoApprove, "auto-approve", false, "Skip interactive confirmation before destroying")
}
