package orchestrator

import "fmt"

// orderByDependency returns a list of verteces ordered by their dependencies.
// representing allocations as vertexes in a graph and dependencies as edges, the following returns an ordered list of
// lists such that: outer list is ordered, the inner list is unordered for allocations to be executed in parallel
func orderByDependency(vertices map[string][]string) ([][]string, error) {
	// Build reverse dependency map and in-degree counts
	dependentMap := make(map[string][]string)
	inDegree := make(map[string]int)

	// Initialize data structures
	for vertexName, vertex := range vertices {
		inDegree[vertexName] = len(vertex)
		for _, dep := range vertex {
			if _, ok := vertices[dep]; !ok {
				return nil, fmt.Errorf("service %s depends on non-existent service %s", vertexName, dep)
			}
			dependentMap[dep] = append(dependentMap[dep], vertexName)
		}
	}

	// Initialize queue with services that have no dependencies
	queue := make([]string, 0)
	for vertex, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, vertex)
		}
	}

	result := [][]string{}

	// Process levels using BFS
	for len(queue) > 0 {
		levelSize := len(queue)
		currentLevel := make([]string, 0, levelSize)

		for i := 0; i < levelSize; i++ {
			service := queue[0]
			queue = queue[1:]
			currentLevel = append(currentLevel, service)

			// Update dependents' in-degrees
			for _, dependent := range dependentMap[service] {
				inDegree[dependent]--
				if inDegree[dependent] == 0 {
					queue = append(queue, dependent)
				}
			}
		}

		result = append(result, currentLevel)
	}

	// Check for cycles
	total := 0
	for _, level := range result {
		total += len(level)
	}
	if total != len(vertices) {
		return nil, fmt.Errorf("cycle detected in dependencies")
	}

	return result, nil
}

// aggregateErrors aggregates multiple errors coming from
// a channel until there are no error msgs anymore
func aggregateErrors(errCh chan error) error {
	var aggErr error
	for err := range errCh {
		if aggErr == nil {
			aggErr = err
			continue
		} else if err != nil {
			aggErr = fmt.Errorf("%w\n%w", aggErr, err)
		}
	}
	return aggErr
}
