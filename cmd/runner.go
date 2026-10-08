package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yoohya/terracotta/config"
	"github.com/yoohya/terracotta/terraform"
)

type moduleResult struct {
	Module string
	Status string // "success", "failed"
	Error  error
}

// prepareModules はコンフィグを読み込み、依存順にソートしたモジュール一覧を返す。
// --profile が指定されていれば AWS_PROFILE も設定する。
func prepareModules() (*config.Config, []*config.ModuleNode) {
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

	if awsProfile != "" {
		if err := os.Setenv("AWS_PROFILE", awsProfile); err != nil {
			fmt.Printf("Warning: failed to set AWS_PROFILE: %v\n", err)
		}
	}

	return cfg, sortedModules
}

// buildInitArgs は terraform init の引数を構築する。
func buildInitArgs() []string {
	initArgs := []string{"init", "-input=false"}
	if upgradeProviders {
		initArgs = append(initArgs, "-upgrade")
	}
	return initArgs
}

// runModulesFailFast は各モジュールで init → terraform <actionArgs> を順に実行し、
// 最初の失敗で停止する。未実行のモジュールはサマリーで skipped と表示する。
// 失敗があれば true を返す。
func runModulesFailFast(cfg *config.Config, modules []*config.ModuleNode, actionArgs []string, successMsg string) bool {
	actionName := strings.ToUpper(actionArgs[0])
	actionCmdStr := "terraform " + strings.Join(actionArgs, " ")
	initArgs := buildInitArgs()
	initCmdStr := "terraform " + strings.Join(initArgs, " ")

	var results []moduleResult

	for _, mod := range modules {
		modulePath := filepath.Join(cfg.BasePath, mod.Path)
		fmt.Printf("[%s] INIT (%s)\n", mod.Path, modulePath)
		if upgradeProviders {
			fmt.Printf("[%s] Provider upgrade enabled\n", mod.Path)
		}

		if err := terraform.RunCommand(mod.Path, modulePath, initArgs...); err != nil {
			fmt.Printf("✖ [%s] Terraform init failed!\n", mod.Path)
			fmt.Printf("    Module path : %s\n", modulePath)
			fmt.Printf("    Command     : %s\n", initCmdStr)
			fmt.Printf("    Error       : %v\n", err)
			results = append(results, moduleResult{Module: mod.Path, Status: "failed", Error: fmt.Errorf("init failed: %v", err)})
			break
		}

		fmt.Printf("[%s] %s (%s)\n", mod.Path, actionName, modulePath)
		if err := terraform.RunCommand(mod.Path, modulePath, actionArgs...); err != nil {
			fmt.Printf("✖ [%s] Terraform %s failed!\n", mod.Path, actionArgs[0])
			fmt.Printf("    Module path : %s\n", modulePath)
			fmt.Printf("    Command     : %s\n", actionCmdStr)
			fmt.Printf("    Error       : %v\n", err)
			results = append(results, moduleResult{Module: mod.Path, Status: "failed", Error: fmt.Errorf("%s failed: %v", actionArgs[0], err)})
			break
		}

		results = append(results, moduleResult{Module: mod.Path, Status: "success"})
	}

	fmt.Printf("\n%s Summary:\n", strings.ToUpper(actionArgs[0][:1])+actionArgs[0][1:])
	encounteredFailure := false
	executed := map[string]bool{}
	for _, res := range results {
		executed[res.Module] = true
		switch res.Status {
		case "success":
			fmt.Printf("✔ %s: %s\n", res.Module, successMsg)
		case "failed":
			fmt.Printf("✖ %s: failed - %v\n", res.Module, res.Error)
			encounteredFailure = true
		}
	}
	for _, mod := range modules {
		if !executed[mod.Path] {
			fmt.Printf("⏭ %s: skipped\n", mod.Path)
		}
	}
	return encounteredFailure
}
