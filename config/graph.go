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
}

// BuildExecutionGraph builds a graph from the given Config.
func BuildExecutionGraph(cfg *Config) (*ExecutionGraph, error) {
	graph := &ExecutionGraph{Nodes: make(map[string]*ModuleNode)}

	// Initialize nodes
	for _, mod := range cfg.Modules {
		graph.Nodes[mod.Path] = &ModuleNode{
			Path:      mod.Path,
			DependsOn: mod.DependsOn,
		}
	}

	return graph, nil
}

// TopoSortedModules performs topological sort to determine execution order.
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

	for _, node := range g.Nodes {
		if !visited[node.Path] {
			if err := visit(node); err != nil {
				return nil, err
			}
		}
	}

	return sorted, nil
}
