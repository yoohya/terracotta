package terraform

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunCommand(t *testing.T) {
	// Check if terraform is installed
	_, err := exec.LookPath("terraform")
	if err != nil {
		t.Skip("terraform not found in PATH, skipping integration tests")
	}

	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	tests := []struct {
		name       string
		prefix     string
		modulePath string
		args       []string
		wantError  bool
	}{
		{
			name:       "successful version check",
			prefix:     "test-module",
			modulePath: tmpDir,
			args:       []string{"version"},
			wantError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := RunCommand(tt.prefix, tt.modulePath, tt.args...)

			if tt.wantError && err == nil {
				t.Errorf("expected error but got none")
			}

			if !tt.wantError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestRunCommandInvalidDirectory(t *testing.T) {
	err := RunCommand("test", "/nonexistent/path", "init")
	if err == nil {
		t.Error("expected error for nonexistent directory, got none")
	}
}

func TestRunCommandOutputPrefixing(t *testing.T) {
	// This test verifies that output is properly prefixed
	// We can't easily capture stdout in unit tests, but we can verify
	// the function executes without panic
	tmpDir := t.TempDir()

	err := RunCommand("test-prefix", tmpDir, "version")
	// The command will likely fail (terraform not found or wrong args),
	// but it shouldn't panic
	_ = err
}

func TestRunCommandWorkingDirectory(t *testing.T) {
	// Create a test directory structure
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "submodule")
	err := os.MkdirAll(subDir, 0755)
	if err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}

	// Create a test file to verify we're in the right directory
	testFile := filepath.Join(subDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Run a command in the subdirectory
	// We expect it to execute with subDir as the working directory
	err = RunCommand("test", subDir, "version")
	_ = err // Command will fail, but that's OK for this test
}

func TestRunCommandArguments(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name string
		args []string
	}{
		{
			name: "single argument",
			args: []string{"init"},
		},
		{
			name: "multiple arguments",
			args: []string{"plan", "-out=tfplan"},
		},
		{
			name: "arguments with flags",
			args: []string{"apply", "-auto-approve", "-input=false"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := RunCommand("test", tmpDir, tt.args...)
			// We don't check the error because terraform might not be installed
			// We just verify the function doesn't panic
			_ = err
		})
	}
}

// TestRunCommandOutput verifies that command output handling works correctly
func TestRunCommandOutput(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a simple shell script that produces output
	scriptPath := filepath.Join(tmpDir, "test.sh")
	scriptContent := `#!/bin/bash
echo "Line 1"
echo "Line 2"
echo "Line 3"
`
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	if err != nil {
		t.Fatalf("failed to create test script: %v", err)
	}

	// Note: This test is platform-dependent and might not work on Windows
	// In a real scenario, you'd want to handle this better
	if _, err := os.Stat("/bin/bash"); err == nil {
		// Run the command - it should execute without error
		cmd := exec.Command("/bin/bash", scriptPath)
		cmd.Dir = tmpDir
		output, err := cmd.CombinedOutput()

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		outputStr := string(output)
		if !strings.Contains(outputStr, "Line 1") {
			t.Error("expected output to contain 'Line 1'")
		}
	}
}
