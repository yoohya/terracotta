package config

import (
	"fmt"
)

// ModuleNode represents a node in the execution graph.
type ModuleNode struct {
	Path      string
	DependsOn []string
	Visited   bool
	TempMark  bool
}

// ExecutionGraph holds all module nodes for dependency resolution.
type ExecutionGraph struct {
	Nodes map[string]*ModuleNode
	// Order は設定ファイルに書かれたモジュールの順序。ソート結果を毎回同じにするために使う。
	Order []string
}

// BuildExecutionGraph builds a graph from the given Config.
// It returns an error if no modules are defined, a module path is empty,
// or the same module path is defined more than once.
func BuildExecutionGraph(cfg *Config) (*ExecutionGraph, error) {
	if len(cfg.Modules) == 0 {
		return nil, fmt.Errorf("no modules defined in config")
	}

	graph := &ExecutionGraph{Nodes: make(map[string]*ModuleNode)}

	// Initialize nodes
	for i, mod := range cfg.Modules {
		if mod.Path == "" {
			return nil, fmt.Errorf("module at index %d has an empty path", i)
		}
		if _, exists := graph.Nodes[mod.Path]; exists {
			return nil, fmt.Errorf("duplicate module path %s", mod.Path)
		}
		graph.Nodes[mod.Path] = &ModuleNode{
			Path:      mod.Path,
			DependsOn: mod.DependsOn,
		}
		graph.Order = append(graph.Order, mod.Path)
	}

	return graph, nil
}

// TopoSortedModules performs topological sort to determine execution order.
// Modules without ordering constraints keep the order in which they appear in the config.
func (g *ExecutionGraph) TopoSortedModules() ([]*ModuleNode, error) {
	var sorted []*ModuleNode
	visited := make(map[string]bool)

	var visit func(n *ModuleNode) error
	visit = func(n *ModuleNode) error {
		if n.TempMark {
			return fmt.Errorf("cyclic dependency detected at %s", n.Path)
		}
		if !visited[n.Path] {
			n.TempMark = true
			for _, dep := range n.DependsOn {
				depNode, exists := g.Nodes[dep]
				if !exists {
					return fmt.Errorf("unknown dependency %s for module %s", dep, n.Path)
				}
				if err := visit(depNode); err != nil {
					return err
				}
			}
			n.TempMark = false
			visited[n.Path] = true
			sorted = append(sorted, n)
		}
		return nil
	}

	for _, path := range g.Order {
		node := g.Nodes[path]
		if !visited[node.Path] {
			if err := visit(node); err != nil {
				return nil, err
			}
		}
	}

	return sorted, nil
}
