package authz

import "fmt"

// DetectCycles checks for circular dependencies in a directed graph.
// Graph is represented as map[node] -> []children.
func DetectCycles(graph map[string][]string) error {
	visited := make(map[string]bool)
	recursionStack := make(map[string]bool)

	for node := range graph {
		if detectCycleDFS(node, graph, visited, recursionStack) {
			return fmt.Errorf("cycle detected involving node %s", node)
		}
	}
	return nil
}

func detectCycleDFS(node string, graph map[string][]string, visited, recursionStack map[string]bool) bool {
	visited[node] = true
	recursionStack[node] = true

	for _, neighbor := range graph[node] {
		if !visited[neighbor] {
			if detectCycleDFS(neighbor, graph, visited, recursionStack) {
				return true
			}
		} else if recursionStack[neighbor] {
			return true
		}
	}

	recursionStack[node] = false
	return false
}
