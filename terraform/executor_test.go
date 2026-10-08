package terraform

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeTerraformScript は作業ディレクトリと引数を出力し、FAKE_TF_EXIT の値で終了する。
const fakeTerraformScript = `#!/bin/sh
echo "pwd=$(pwd -P)"
echo "args=$*"
echo "to-stderr" 1>&2
exit "${FAKE_TF_EXIT:-0}"
`

// installFakeTerraform は偽の terraform を PATH の先頭に置く。
func installFakeTerraform(t *testing.T) {
	t.Helper()
	requireSh(t)
	binDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(binDir, "terraform"), []byte(fakeTerraformScript), 0755); err != nil {
		t.Fatalf("failed to write fake terraform: %v", err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// captureStdout は f の実行中に os.Stdout へ書かれた内容を返す。
func captureStdout(t *testing.T, f func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	original := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = original }()

	outCh := make(chan string)
	go func() {
		b, _ := io.ReadAll(r)
		outCh <- string(b)
	}()

	f()
	_ = w.Close()
	return <-outCh
}

func TestRunCommand_RunsInModuleDirWithArgs(t *testing.T) {
	installFakeTerraform(t)
	moduleDir := t.TempDir()
	resolvedDir, err := filepath.EvalSymlinks(moduleDir)
	if err != nil {
		t.Fatalf("failed to resolve temp dir: %v", err)
	}

	var runErr error
	out := captureStdout(t, func() {
		runErr = RunCommand("shared/network", moduleDir, "apply", "-auto-approve", "-input=false")
	})

	if runErr != nil {
		t.Fatalf("unexpected error: %v", runErr)
	}
	for _, want := range []string{
		"[shared/network] Running: terraform [apply -auto-approve -input=false]\n",
		"[shared/network] pwd=" + resolvedDir + "\n",
		"[shared/network] args=apply -auto-approve -input=false\n",
		"[shared/network] to-stderr\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("expected output to contain %q, got:\n%s", want, out)
		}
	}
}

func TestRunCommand_ReturnsErrorOnNonZeroExit(t *testing.T) {
	installFakeTerraform(t)
	t.Setenv("FAKE_TF_EXIT", "2")

	var runErr error
	out := captureStdout(t, func() {
		runErr = RunCommand("mod", t.TempDir(), "plan")
	})

	if runErr == nil {
		t.Fatal("expected error for non-zero exit, got none")
	}
	if !strings.Contains(runErr.Error(), "exit status 2") {
		t.Errorf("expected exit status 2, got %v", runErr)
	}
	// 失敗しても、それまでの出力は表示される
	if !strings.Contains(out, "[mod] args=plan") {
		t.Errorf("expected output before failure, got:\n%s", out)
	}
}

func TestRunCommand_NonexistentDirectory(t *testing.T) {
	installFakeTerraform(t)

	var runErr error
	captureStdout(t, func() {
		runErr = RunCommand("mod", filepath.Join(t.TempDir(), "missing"), "init")
	})

	if runErr == nil {
		t.Error("expected error for nonexistent directory, got none")
	}
}

func TestRunCommand_TerraformNotFound(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	var runErr error
	captureStdout(t, func() {
		runErr = RunCommand("mod", t.TempDir(), "version")
	})

	if runErr == nil {
		t.Error("expected error when terraform is not in PATH, got none")
	}
}
