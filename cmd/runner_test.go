package cmd

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/yoohya/terracotta/config"
)

// terraformCall は runTerraform への 1 回の呼び出しを記録したもの。
type terraformCall struct {
	Module string
	Path   string
	Args   []string
}

// fakeTerraform は runTerraform を差し替え、呼び出しを記録する。
// failOn に "module:subcommand"（例: "b:init"）を入れると、その呼び出しでエラーを返す。
type fakeTerraform struct {
	calls  []terraformCall
	failOn map[string]bool
}

func installFakeTerraform(t *testing.T, failOn ...string) *fakeTerraform {
	t.Helper()
	fake := &fakeTerraform{failOn: map[string]bool{}}
	for _, f := range failOn {
		fake.failOn[f] = true
	}

	original := runTerraform
	runTerraform = func(prefix string, modulePath string, args ...string) error {
		fake.calls = append(fake.calls, terraformCall{Module: prefix, Path: modulePath, Args: args})
		if fake.failOn[prefix+":"+args[0]] {
			return errors.New("exit status 1")
		}
		return nil
	}
	t.Cleanup(func() { runTerraform = original })
	return fake
}

// steps は呼び出しを "module:subcommand" の列にまとめる。
func (f *fakeTerraform) steps() []string {
	var steps []string
	for _, c := range f.calls {
		steps = append(steps, c.Module+":"+c.Args[0])
	}
	return steps
}

func nodes(paths ...string) []*config.ModuleNode {
	var ns []*config.ModuleNode
	for _, p := range paths {
		ns = append(ns, &config.ModuleNode{Path: p})
	}
	return ns
}

func TestRunModules_AllSucceed(t *testing.T) {
	fake := installFakeTerraform(t)
	cfg := &config.Config{BasePath: "environments/dev"}
	var out bytes.Buffer

	results := runModules(&out, cfg, nodes("shared/network", "serviceA/backend"), []string{"apply", "-auto-approve"}, false, stopOnFailure)

	wantCalls := []terraformCall{
		{Module: "shared/network", Path: filepath.Join("environments/dev", "shared/network"), Args: []string{"init", "-input=false"}},
		{Module: "shared/network", Path: filepath.Join("environments/dev", "shared/network"), Args: []string{"apply", "-auto-approve"}},
		{Module: "serviceA/backend", Path: filepath.Join("environments/dev", "serviceA/backend"), Args: []string{"init", "-input=false"}},
		{Module: "serviceA/backend", Path: filepath.Join("environments/dev", "serviceA/backend"), Args: []string{"apply", "-auto-approve"}},
	}
	if diff := cmp.Diff(wantCalls, fake.calls); diff != "" {
		t.Errorf("terraform calls mismatch (-want +got):\n%s", diff)
	}

	wantResults := []moduleResult{
		{Module: "shared/network", Status: "success"},
		{Module: "serviceA/backend", Status: "success"},
	}
	if diff := cmp.Diff(wantResults, results, cmp.Comparer(sameError)); diff != "" {
		t.Errorf("results mismatch (-want +got):\n%s", diff)
	}

	for _, want := range []string{
		"[shared/network] INIT (environments/dev/shared/network)",
		"[shared/network] APPLY (environments/dev/shared/network)",
	} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("expected output to contain %q, got:\n%s", want, out.String())
		}
	}
}

func TestRunModules_UpgradeAddsFlagToInit(t *testing.T) {
	fake := installFakeTerraform(t)
	var out bytes.Buffer

	runModules(&out, &config.Config{}, nodes("a"), []string{"plan"}, true, continueOnFailure)

	want := []string{"init", "-input=false", "-upgrade"}
	if diff := cmp.Diff(want, fake.calls[0].Args); diff != "" {
		t.Errorf("init args mismatch (-want +got):\n%s", diff)
	}
	if !strings.Contains(out.String(), "[a] Provider upgrade enabled") {
		t.Errorf("expected upgrade notice, got:\n%s", out.String())
	}
}

func TestRunModules_StopOnFailure(t *testing.T) {
	tests := []struct {
		name        string
		failOn      string
		wantSteps   []string
		wantResults []moduleResult
	}{
		{
			name:      "init failure stops before action and remaining modules",
			failOn:    "b:init",
			wantSteps: []string{"a:init", "a:apply", "b:init"},
			wantResults: []moduleResult{
				{Module: "a", Status: "success"},
				{Module: "b", Status: "failed", Error: errors.New("init failed: exit status 1")},
			},
		},
		{
			name:      "action failure stops remaining modules",
			failOn:    "b:apply",
			wantSteps: []string{"a:init", "a:apply", "b:init", "b:apply"},
			wantResults: []moduleResult{
				{Module: "a", Status: "success"},
				{Module: "b", Status: "failed", Error: errors.New("apply failed: exit status 1")},
			},
		},
		{
			name:      "first module failure runs nothing else",
			failOn:    "a:apply",
			wantSteps: []string{"a:init", "a:apply"},
			wantResults: []moduleResult{
				{Module: "a", Status: "failed", Error: errors.New("apply failed: exit status 1")},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := installFakeTerraform(t, tt.failOn)
			var out bytes.Buffer

			results := runModules(&out, &config.Config{}, nodes("a", "b", "c"), []string{"apply", "-auto-approve"}, false, stopOnFailure)

			if diff := cmp.Diff(tt.wantSteps, fake.steps()); diff != "" {
				t.Errorf("steps mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.wantResults, results, cmp.Comparer(sameError)); diff != "" {
				t.Errorf("results mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRunModules_ContinueOnFailure(t *testing.T) {
	fake := installFakeTerraform(t, "a:init", "b:plan")
	var out bytes.Buffer

	results := runModules(&out, &config.Config{}, nodes("a", "b", "c"), []string{"plan"}, false, continueOnFailure)

	wantSteps := []string{"a:init", "b:init", "b:plan", "c:init", "c:plan"}
	if diff := cmp.Diff(wantSteps, fake.steps()); diff != "" {
		t.Errorf("steps mismatch (-want +got):\n%s", diff)
	}

	wantResults := []moduleResult{
		{Module: "a", Status: "failed", Error: errors.New("init failed: exit status 1")},
		{Module: "b", Status: "failed", Error: errors.New("plan failed: exit status 1")},
		{Module: "c", Status: "success"},
	}
	if diff := cmp.Diff(wantResults, results, cmp.Comparer(sameError)); diff != "" {
		t.Errorf("results mismatch (-want +got):\n%s", diff)
	}
}

func TestRunModules_PrintsFailureDetails(t *testing.T) {
	installFakeTerraform(t, "a:destroy")
	var out bytes.Buffer

	runModules(&out, &config.Config{BasePath: "base"}, nodes("a"), []string{"destroy", "-auto-approve"}, false, stopOnFailure)

	want := "✖ [a] Terraform destroy failed!\n" +
		"    Module path : base/a\n" +
		"    Command     : terraform destroy -auto-approve\n" +
		"    Error       : exit status 1\n"
	if !strings.Contains(out.String(), want) {
		t.Errorf("expected failure details %q, got:\n%s", want, out.String())
	}
}

func TestPrintSummary(t *testing.T) {
	tests := []struct {
		name        string
		results     []moduleResult
		wantOut     string
		wantFailure bool
	}{
		{
			name: "all succeeded",
			results: []moduleResult{
				{Module: "a", Status: "success"},
				{Module: "b", Status: "success"},
				{Module: "c", Status: "success"},
			},
			wantOut:     "\nApply Summary:\n✔ a: applied successfully\n✔ b: applied successfully\n✔ c: applied successfully\n",
			wantFailure: false,
		},
		{
			name: "failure with skipped modules",
			results: []moduleResult{
				{Module: "a", Status: "success"},
				{Module: "b", Status: "failed", Error: errors.New("apply failed: exit status 1")},
			},
			wantOut:     "\nApply Summary:\n✔ a: applied successfully\n✖ b: failed - apply failed: exit status 1\n⏭ c: skipped\n",
			wantFailure: true,
		},
		{
			name:        "nothing executed",
			results:     nil,
			wantOut:     "\nApply Summary:\n⏭ a: skipped\n⏭ b: skipped\n⏭ c: skipped\n",
			wantFailure: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer

			failed := printSummary(&out, "Apply", nodes("a", "b", "c"), tt.results, "applied successfully")

			if failed != tt.wantFailure {
				t.Errorf("expected failure=%v, got %v", tt.wantFailure, failed)
			}
			if diff := cmp.Diff(tt.wantOut, out.String()); diff != "" {
				t.Errorf("summary mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestLoadSortedModules(t *testing.T) {
	tests := []struct {
		name      string
		filename  string
		wantPaths []string
		wantErr   string
	}{
		{
			name:      "valid config",
			filename:  "valid.yaml",
			wantPaths: []string{"module-a", "module-b", "module-c"},
		},
		{name: "missing file", filename: "nonexistent.yaml", wantErr: "failed to load config"},
		{name: "invalid yaml", filename: "invalid.yaml", wantErr: "failed to load config"},
		{name: "duplicate module path", filename: "duplicate-path.yaml", wantErr: "failed to build execution graph"},
		{name: "no modules", filename: "empty-modules.yaml", wantErr: "failed to build execution graph"},
		{name: "cyclic dependency", filename: "cyclic.yaml", wantErr: "failed to resolve module order"},
		{name: "unknown dependency", filename: "unknown-dep.yaml", wantErr: "failed to resolve module order"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, sorted, err := loadSortedModules(filepath.Join("..", "testdata", tt.filename))

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got none", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("expected error containing %q, got %q", tt.wantErr, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg.BasePath != "test/path" {
				t.Errorf("expected base path test/path, got %s", cfg.BasePath)
			}
			var got []string
			for _, n := range sorted {
				got = append(got, n.Path)
			}
			if diff := cmp.Diff(tt.wantPaths, got); diff != "" {
				t.Errorf("sorted modules mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestApplyAWSProfile(t *testing.T) {
	t.Run("sets AWS_PROFILE when profile is given", func(t *testing.T) {
		t.Setenv("AWS_PROFILE", "original")
		applyAWSProfile("my-profile")
		if got := os.Getenv("AWS_PROFILE"); got != "my-profile" {
			t.Errorf("expected AWS_PROFILE=my-profile, got %q", got)
		}
	})

	t.Run("keeps AWS_PROFILE when profile is empty", func(t *testing.T) {
		t.Setenv("AWS_PROFILE", "original")
		applyAWSProfile("")
		if got := os.Getenv("AWS_PROFILE"); got != "original" {
			t.Errorf("expected AWS_PROFILE=original, got %q", got)
		}
	})
}

func TestBuildInitArgs(t *testing.T) {
	if diff := cmp.Diff([]string{"init", "-input=false"}, buildInitArgs(false)); diff != "" {
		t.Errorf("without upgrade (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]string{"init", "-input=false", "-upgrade"}, buildInitArgs(true)); diff != "" {
		t.Errorf("with upgrade (-want +got):\n%s", diff)
	}
}

// sameError はエラーメッセージが等しければ同じエラーとみなす。
func sameError(a, b error) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Error() == b.Error()
}
