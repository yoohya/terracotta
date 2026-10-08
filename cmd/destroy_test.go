package cmd

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/yoohya/terracotta/config"
)

func TestReverseModules(t *testing.T) {
	a := &config.ModuleNode{Path: "shared/network"}
	b := &config.ModuleNode{Path: "serviceA/backend"}
	c := &config.ModuleNode{Path: "serviceA/frontend"}
	original := []*config.ModuleNode{a, b, c}

	got := reverseModules(original)

	want := []string{"serviceA/frontend", "serviceA/backend", "shared/network"}
	if len(got) != len(want) {
		t.Fatalf("expected %d modules, got %d", len(want), len(got))
	}
	for i, path := range want {
		if got[i].Path != path {
			t.Errorf("index %d: expected %s, got %s", i, path, got[i].Path)
		}
	}
	if original[0] != a {
		t.Errorf("original slice must not be modified")
	}
}

func TestReverseModules_DependentsBeforeDependencies(t *testing.T) {
	cfg := &config.Config{
		Modules: []config.Module{
			{Path: "shared/network"},
			{Path: "serviceA/backend", DependsOn: []string{"shared/network"}},
			{Path: "serviceA/frontend", DependsOn: []string{"serviceA/backend"}},
		},
	}
	graph, err := config.BuildExecutionGraph(cfg)
	if err != nil {
		t.Fatalf("BuildExecutionGraph failed: %v", err)
	}
	sorted, err := graph.TopoSortedModules()
	if err != nil {
		t.Fatalf("TopoSortedModules failed: %v", err)
	}

	pos := map[string]int{}
	for i, mod := range reverseModules(sorted) {
		pos[mod.Path] = i
	}

	for _, mod := range cfg.Modules {
		for _, dep := range mod.DependsOn {
			if pos[mod.Path] >= pos[dep] {
				t.Errorf("%s must be destroyed before its dependency %s", mod.Path, dep)
			}
		}
	}
}

func TestConfirmDestroy(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"exact match", "destroy\n", true},
		{"surrounding whitespace", "  destroy  \n", true},
		{"no trailing newline", "destroy", true},
		{"yes is rejected", "yes\n", false},
		{"uppercase is rejected", "DESTROY\n", false},
		{"empty input", "\n", false},
		{"EOF", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			got, err := confirmDestroy(strings.NewReader(tt.input), &out)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("expected %v, got %v", tt.want, got)
			}
			if !strings.Contains(out.String(), "Type 'destroy' to confirm") {
				t.Errorf("prompt not written, got %q", out.String())
			}
		})
	}
}

func TestRunDestroy(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		autoApprove bool
		failOn      []string
		wantSteps   []string
		wantFailed  bool
		wantOut     []string
		notWantOut  []string
	}{
		{
			name:       "confirmed destroy runs in reverse order",
			input:      "destroy\n",
			wantSteps:  []string{"c:init", "c:destroy", "b:init", "b:destroy", "a:init", "a:destroy"},
			wantFailed: false,
			wantOut: []string{
				"The following modules will be DESTROYED in this order:\n  1. c\n  2. b\n  3. a\n",
				"Type 'destroy' to confirm: ",
				"\nDestroy Summary:\n✔ c: destroyed successfully\n✔ b: destroyed successfully\n✔ a: destroyed successfully\n",
			},
		},
		{
			name:       "wrong confirmation cancels without running terraform",
			input:      "yes\n",
			wantSteps:  nil,
			wantFailed: false,
			wantOut:    []string{"Destroy cancelled."},
			notWantOut: []string{"Destroy Summary"},
		},
		{
			name:       "EOF on stdin cancels without running terraform",
			input:      "",
			wantSteps:  nil,
			wantFailed: false,
			wantOut:    []string{"Destroy cancelled."},
		},
		{
			name:        "auto-approve skips confirmation",
			input:       "",
			autoApprove: true,
			wantSteps:   []string{"c:init", "c:destroy", "b:init", "b:destroy", "a:init", "a:destroy"},
			wantFailed:  false,
			notWantOut:  []string{"Type 'destroy' to confirm", "Destroy cancelled."},
		},
		{
			name:       "failure stops and skips remaining modules",
			input:      "destroy\n",
			failOn:     []string{"b:destroy"},
			wantSteps:  []string{"c:init", "c:destroy", "b:init", "b:destroy"},
			wantFailed: true,
			wantOut: []string{
				"✖ b: failed - destroy failed: exit status 1\n",
				"⏭ a: skipped\n",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := installFakeTerraform(t, tt.failOn...)
			var out bytes.Buffer

			failed, err := runDestroy(strings.NewReader(tt.input), &out, &config.Config{}, nodes("a", "b", "c"), tt.autoApprove, false)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if failed != tt.wantFailed {
				t.Errorf("expected failed=%v, got %v", tt.wantFailed, failed)
			}
			if diff := cmp.Diff(tt.wantSteps, fake.steps()); diff != "" {
				t.Errorf("steps mismatch (-want +got):\n%s", diff)
			}
			for _, want := range tt.wantOut {
				if !strings.Contains(out.String(), want) {
					t.Errorf("expected output to contain %q, got:\n%s", want, out.String())
				}
			}
			for _, notWant := range tt.notWantOut {
				if strings.Contains(out.String(), notWant) {
					t.Errorf("expected output not to contain %q, got:\n%s", notWant, out.String())
				}
			}
		})
	}
}

// errReader は常にエラーを返す io.Reader。
type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("read failed") }

func TestRunDestroy_ConfirmationReadError(t *testing.T) {
	fake := installFakeTerraform(t)
	var out bytes.Buffer

	_, err := runDestroy(errReader{}, &out, &config.Config{}, nodes("a"), false, false)

	if err == nil || !strings.Contains(err.Error(), "read failed") {
		t.Fatalf("expected read error, got %v", err)
	}
	if len(fake.calls) != 0 {
		t.Errorf("expected no terraform calls, got %v", fake.steps())
	}
}
