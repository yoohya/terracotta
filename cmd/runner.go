package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/yoohya/terracotta/config"
	"github.com/yoohya/terracotta/terraform"
)

// runTerraform は terraform を実行する関数。テストで差し替えられるよう変数にしている。
var runTerraform = terraform.RunCommand

type moduleResult struct {
	Module string
	Status string // "success", "failed"
	Error  error
}

// failureMode はモジュールの実行に失敗したときの振る舞いを表す。
type failureMode int

const (
	// stopOnFailure は最初の失敗で停止し、残りのモジュールを実行しない（apply / destroy）。
	stopOnFailure failureMode = iota
	// continueOnFailure は失敗しても残りのモジュールを実行する（plan）。
	continueOnFailure
)

// prepareModules はコンフィグを読み込み、依存順にソートしたモジュール一覧を返す。
// --profile が指定されていれば AWS_PROFILE も設定する。失敗した場合はプロセスを終了する。
func prepareModules() (*config.Config, []*config.ModuleNode) {
	cfg, sortedModules, err := loadSortedModules(configPath)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	applyAWSProfile(awsProfile)
	return cfg, sortedModules
}

// loadSortedModules はコンフィグを読み込み、依存順にソートしたモジュール一覧を返す。
func loadSortedModules(path string) (*config.Config, []*config.ModuleNode, error) {
	cfg, err := config.LoadConfig(path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load config: %v", err)
	}

	graph, err := config.BuildExecutionGraph(cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build execution graph: %v", err)
	}

	sortedModules, err := graph.TopoSortedModules()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to resolve module order: %v", err)
	}

	return cfg, sortedModules, nil
}

// applyAWSProfile は profile が指定されていれば AWS_PROFILE に設定する。
func applyAWSProfile(profile string) {
	if profile == "" {
		return
	}
	if err := os.Setenv("AWS_PROFILE", profile); err != nil {
		fmt.Printf("Warning: failed to set AWS_PROFILE: %v\n", err)
	}
}

// buildInitArgs は terraform init の引数を構築する。
func buildInitArgs(upgrade bool) []string {
	initArgs := []string{"init", "-input=false"}
	if upgrade {
		initArgs = append(initArgs, "-upgrade")
	}
	return initArgs
}

// runModules は各モジュールで init → terraform <actionArgs> を順に実行し、結果を返す。
// mode が stopOnFailure の場合は最初の失敗で停止する。
func runModules(out io.Writer, cfg *config.Config, modules []*config.ModuleNode, actionArgs []string, upgrade bool, mode failureMode) []moduleResult {
	actionName := strings.ToUpper(actionArgs[0])
	actionCmdStr := "terraform " + strings.Join(actionArgs, " ")
	initArgs := buildInitArgs(upgrade)
	initCmdStr := "terraform " + strings.Join(initArgs, " ")

	var results []moduleResult

	for _, mod := range modules {
		modulePath := filepath.Join(cfg.BasePath, mod.Path)
		_, _ = fmt.Fprintf(out, "[%s] INIT (%s)\n", mod.Path, modulePath)
		if upgrade {
			_, _ = fmt.Fprintf(out, "[%s] Provider upgrade enabled\n", mod.Path)
		}

		if err := runTerraform(mod.Path, modulePath, initArgs...); err != nil {
			printStepFailure(out, mod.Path, modulePath, "init", initCmdStr, err)
			results = append(results, moduleResult{Module: mod.Path, Status: "failed", Error: fmt.Errorf("init failed: %v", err)})
			if mode == stopOnFailure {
				break
			}
			continue
		}

		_, _ = fmt.Fprintf(out, "[%s] %s (%s)\n", mod.Path, actionName, modulePath)
		if err := runTerraform(mod.Path, modulePath, actionArgs...); err != nil {
			printStepFailure(out, mod.Path, modulePath, actionArgs[0], actionCmdStr, err)
			results = append(results, moduleResult{Module: mod.Path, Status: "failed", Error: fmt.Errorf("%s failed: %v", actionArgs[0], err)})
			if mode == stopOnFailure {
				break
			}
			continue
		}

		results = append(results, moduleResult{Module: mod.Path, Status: "success"})
	}

	return results
}

func printStepFailure(out io.Writer, module, modulePath, step, cmdStr string, err error) {
	_, _ = fmt.Fprintf(out, "✖ [%s] Terraform %s failed!\n", module, step)
	_, _ = fmt.Fprintf(out, "    Module path : %s\n", modulePath)
	_, _ = fmt.Fprintf(out, "    Command     : %s\n", cmdStr)
	_, _ = fmt.Fprintf(out, "    Error       : %v\n", err)
}

// printSummary は実行結果のサマリーを表示する。結果がないモジュールは skipped と表示する。
// 失敗があれば true を返す。
func printSummary(out io.Writer, title string, modules []*config.ModuleNode, results []moduleResult, successMsg string) bool {
	_, _ = fmt.Fprintf(out, "\n%s Summary:\n", title)
	encounteredFailure := false
	executed := map[string]bool{}
	for _, res := range results {
		executed[res.Module] = true
		switch res.Status {
		case "success":
			_, _ = fmt.Fprintf(out, "✔ %s: %s\n", res.Module, successMsg)
		case "failed":
			_, _ = fmt.Fprintf(out, "✖ %s: failed - %v\n", res.Module, res.Error)
			encounteredFailure = true
		}
	}
	for _, mod := range modules {
		if !executed[mod.Path] {
			_, _ = fmt.Fprintf(out, "⏭ %s: skipped\n", mod.Path)
		}
	}
	return encounteredFailure
}
