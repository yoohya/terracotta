package config

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildExecutionGraph(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		validate func(*testing.T, *ExecutionGraph)
	}{
		{
			name:     "valid graph",
			filename: "valid.yaml",
			validate: func(t *testing.T, g *ExecutionGraph) {
				if len(g.Nodes) != 3 {
					t.Errorf("expected 3 nodes, got %d", len(g.Nodes))
				}
				if _, exists := g.Nodes["module-a"]; !exists {
					t.Error("expected module-a to exist in graph")
				}
				if _, exists := g.Nodes["module-b"]; !exists {
					t.Error("expected module-b to exist in graph")
				}
				if _, exists := g.Nodes["module-c"]; !exists {
					t.Error("expected module-c to exist in graph")
				}
			},
		},
		{
			name:     "no dependencies",
			filename: "no-deps.yaml",
			validate: func(t *testing.T, g *ExecutionGraph) {
				if len(g.Nodes) != 3 {
					t.Errorf("expected 3 nodes, got %d", len(g.Nodes))
				}
				for path, node := range g.Nodes {
					if len(node.DependsOn) != 0 {
						t.Errorf("expected node %s to have no dependencies, got %d", path, len(node.DependsOn))
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join("..", "testdata", tt.filename)
			cfg, err := LoadConfig(path)
			if err != nil {
				t.Fatalf("failed to load config: %v", err)
			}

			graph, err := BuildExecutionGraph(cfg)
			if err != nil {
				t.Fatalf("failed to build graph: %v", err)
			}

			if tt.validate != nil {
				tt.validate(t, graph)
			}
		})
	}
}

func TestTopoSortedModules(t *testing.T) {
	tests := []struct {
		name      string
		filename  string
		wantError bool
		validate  func(*testing.T, []*ModuleNode)
	}{
		{
			name:      "valid topological sort",
			filename:  "valid.yaml",
			wantError: false,
			validate: func(t *testing.T, sorted []*ModuleNode) {
				if len(sorted) != 3 {
					t.Fatalf("expected 3 modules, got %d", len(sorted))
				}

				// module-a should come before module-b and module-c
				// module-b should come before module-c
				positions := make(map[string]int)
				for i, node := range sorted {
					positions[node.Path] = i
				}

				if positions["module-a"] >= positions["module-b"] {
					t.Error("module-a should come before module-b")
				}
				if positions["module-a"] >= positions["module-c"] {
					t.Error("module-a should come before module-c")
				}
				if positions["module-b"] >= positions["module-c"] {
					t.Error("module-b should come before module-c")
				}
			},
		},
		{
			name:      "no dependencies - any order valid",
			filename:  "no-deps.yaml",
			wantError: false,
			validate: func(t *testing.T, sorted []*ModuleNode) {
				if len(sorted) != 3 {
					t.Errorf("expected 3 modules, got %d", len(sorted))
				}
			},
		},
		{
			name:      "cyclic dependency detected",
			filename:  "cyclic.yaml",
			wantError: true,
			validate:  nil,
		},
		{
			name:      "unknown dependency",
			filename:  "unknown-dep.yaml",
			wantError: true,
			validate:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join("..", "testdata", tt.filename)
			cfg, err := LoadConfig(path)
			if err != nil {
				t.Fatalf("failed to load config: %v", err)
			}

			graph, err := BuildExecutionGraph(cfg)
			if err != nil {
				t.Fatalf("failed to build graph: %v", err)
			}

			sorted, err := graph.TopoSortedModules()

			if tt.wantError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if tt.validate != nil {
				tt.validate(t, sorted)
			}
		})
	}
}

func TestCyclicDependencyDetection(t *testing.T) {
	path := filepath.Join("..", "testdata", "cyclic.yaml")
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	graph, err := BuildExecutionGraph(cfg)
	if err != nil {
		t.Fatalf("failed to build graph: %v", err)
	}

	_, err = graph.TopoSortedModules()
	if err == nil {
		t.Fatal("expected cyclic dependency error but got none")
	}

	if !strings.Contains(err.Error(), "cyclic dependency") {
		t.Errorf("expected error to mention 'cyclic dependency', got: %v", err)
	}
}

func TestUnknownDependency(t *testing.T) {
	path := filepath.Join("..", "testdata", "unknown-dep.yaml")
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	graph, err := BuildExecutionGraph(cfg)
	if err != nil {
		t.Fatalf("failed to build graph: %v", err)
	}

	_, err = graph.TopoSortedModules()
	if err == nil {
		t.Fatal("expected unknown dependency error but got none")
	}

	if !strings.Contains(err.Error(), "unknown dependency") {
		t.Errorf("expected error to mention 'unknown dependency', got: %v", err)
	}
}

func TestComplexDependencyGraph(t *testing.T) {
	// Test a more complex dependency structure
	cfg := &Config{
		BasePath: "test",
		Modules: []Module{
			{Path: "a"},
			{Path: "b", DependsOn: []string{"a"}},
			{Path: "c", DependsOn: []string{"a"}},
			{Path: "d", DependsOn: []string{"b", "c"}},
			{Path: "e", DependsOn: []string{"d"}},
		},
	}

	graph, err := BuildExecutionGraph(cfg)
	if err != nil {
		t.Fatalf("failed to build graph: %v", err)
	}

	sorted, err := graph.TopoSortedModules()
	if err != nil {
		t.Fatalf("failed to sort: %v", err)
	}

	if len(sorted) != 5 {
		t.Fatalf("expected 5 modules, got %d", len(sorted))
	}

	// Verify order constraints
	positions := make(map[string]int)
	for i, node := range sorted {
		positions[node.Path] = i
	}

	// a must come before b, c, d, e
	if positions["a"] >= positions["b"] || positions["a"] >= positions["c"] ||
		positions["a"] >= positions["d"] || positions["a"] >= positions["e"] {
		t.Error("module a should come before all others")
	}

	// b and c must come before d
	if positions["b"] >= positions["d"] || positions["c"] >= positions["d"] {
		t.Error("modules b and c should come before d")
	}

	// d must come before e
	if positions["d"] >= positions["e"] {
		t.Error("module d should come before e")
	}
}
