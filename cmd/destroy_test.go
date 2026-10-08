package cmd

import (
	"bytes"
	"strings"
	"testing"

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
