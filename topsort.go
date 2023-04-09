package kanvas

import (
	"errors"
	"sort"
)

func topologicalSort(dependencies map[string][]string) ([][]string, error) {
	var result [][]string
	inDegree := make(map[string]int)
	graph := make(map[string][]string)

	// Initialize the graph, in-degree and the set of nodes with zero in-degree
	for node, deps := range dependencies {
		inDegree[node] = 0

		for _, dep := range deps {
			inDegree[node]++
			graph[dep] = append(graph[dep], node)
		}
	}

	// Perform the topological sort algorithm
	queue := []string{}
	for node := range inDegree {
		if inDegree[node] == 0 {
			queue = append(queue, node)
		}
	}

	for len(queue) > 0 {
		size := len(queue)
		var level []string

		for i := 0; i < size; i++ {
			node := queue[0]
			queue = queue[1:]
			level = append(level, node)

			for _, neighbor := range graph[node] {
				inDegree[neighbor]--
				if inDegree[neighbor] == 0 {
					queue = append(queue, neighbor)
				}
			}
		}

		sort.StringSlice(level).Sort()

		result = append(result, level)
	}

	var resultSize int
	for _, l := range result {
		resultSize += len(l)
	}

	if resultSize != len(inDegree) {
		return nil, errors.New("the graph contains a cycle")
	}

	return result, nil
}
