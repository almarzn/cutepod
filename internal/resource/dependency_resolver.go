package resource

import (
	"fmt"
	"sort"
)

// DependencyResolver manages resource creation order and dependency tracking
type DependencyResolver interface {
	// BuildDependencyGraph analyzes resources and creates a dependency graph
	BuildDependencyGraph(resources []Resource) (*DependencyGraph, error)

	// GetCreationOrder returns resources in dependency order for creation
	GetCreationOrder(graph *DependencyGraph) ([][]Resource, error)

	// GetDeletionOrder returns resources in reverse dependency order for deletion
	GetDeletionOrder(graph *DependencyGraph) ([][]Resource, error)
}

// DependencyGraph represents the dependency relationships between resources
type DependencyGraph struct {
	Nodes map[string]*ResourceNode `json:"nodes"`
	Edges map[string][]string      `json:"edges"`
}

// ResourceNode represents a node in the dependency graph
type ResourceNode struct {
	Resource     Resource `json:"resource"`
	Dependencies []string `json:"dependencies"`
	Dependents   []string `json:"dependents"`
}

// DefaultDependencyResolver implements DependencyResolver
type DefaultDependencyResolver struct{}

// NewDependencyResolver creates a new dependency resolver
func NewDependencyResolver() DependencyResolver {
	return &DefaultDependencyResolver{}
}

// BuildDependencyGraph analyzes resources and creates a dependency graph
func (dr *DefaultDependencyResolver) BuildDependencyGraph(resources []Resource) (*DependencyGraph, error) {
	graph := &DependencyGraph{
		Nodes: make(map[string]*ResourceNode),
		Edges: make(map[string][]string),
	}

	// Create a map for quick resource lookup
	resourceMap := make(map[string]Resource)
	for _, resource := range resources {
		key := dr.getResourceKey(resource)
		resourceMap[key] = resource
	}

	// First pass: create all nodes
	for _, resource := range resources {
		key := dr.getResourceKey(resource)
		graph.Nodes[key] = &ResourceNode{
			Resource:     resource,
			Dependencies: make([]string, 0),
			Dependents:   make([]string, 0),
		}
		graph.Edges[key] = make([]string, 0)
	}

	// Second pass: build dependency relationships
	for _, resource := range resources {
		resourceKey := dr.getResourceKey(resource)
		dependencies := dr.extractDependencies(resource, resourceMap)

		for _, depKey := range dependencies {
			// Add dependency edge
			graph.Edges[resourceKey] = append(graph.Edges[resourceKey], depKey)
			graph.Nodes[resourceKey].Dependencies = append(graph.Nodes[resourceKey].Dependencies, depKey)

			// Add reverse dependency (dependent)
			if depNode, exists := graph.Nodes[depKey]; exists {
				depNode.Dependents = append(depNode.Dependents, resourceKey)
			}
		}
	}

	// Validate the graph for circular dependencies
	if err := dr.validateGraph(graph); err != nil {
		return nil, err
	}

	return graph, nil
}

// GetCreationOrder returns resources in dependency order for creation
func (dr *DefaultDependencyResolver) GetCreationOrder(graph *DependencyGraph) ([][]Resource, error) {
	return dr.topologicalSort(graph, false)
}

// GetDeletionOrder returns resources in reverse dependency order for deletion
func (dr *DefaultDependencyResolver) GetDeletionOrder(graph *DependencyGraph) ([][]Resource, error) {
	creationOrder, err := dr.topologicalSort(graph, false)
	if err != nil {
		return nil, err
	}

	// Reverse the order for deletion
	deletionOrder := make([][]Resource, len(creationOrder))
	for i, level := range creationOrder {
		deletionOrder[len(creationOrder)-1-i] = level
	}

	return deletionOrder, nil
}

// extractDependencies extracts dependencies from a resource
func (dr *DefaultDependencyResolver) extractDependencies(resource Resource, resourceMap map[string]Resource) []string {
	dependencies := make([]string, 0)

	// Get explicit dependencies from the resource
	for _, dep := range resource.GetDependencies() {
		depKey := fmt.Sprintf("%s/%s", dep.Type, dep.Name)
		if _, exists := resourceMap[depKey]; exists {
			dependencies = append(dependencies, depKey)
		}
	}

	// Add implicit dependencies based on resource type
	switch res := resource.(type) {
	case *ContainerResource:
		dependencies = append(dependencies, dr.extractContainerDependencies(res, resourceMap)...)
	case *PodResource:
		dependencies = append(dependencies, dr.extractPodDependencies(res, resourceMap)...)
	}

	return dependencies
}

// extractContainerDependencies extracts dependencies specific to containers
func (dr *DefaultDependencyResolver) extractContainerDependencies(container *ContainerResource, resourceMap map[string]Resource) []string {
	dependencies := make([]string, 0)

	// Network dependencies
	for _, networkName := range container.Spec.Networks {
		networkKey := fmt.Sprintf("%s/%s", ResourceTypeNetwork, networkName)
		if _, exists := resourceMap[networkKey]; exists {
			dependencies = append(dependencies, networkKey)
		}
	}

	// Volume dependencies
	for _, volume := range container.Spec.Volumes {
		volumeKey := fmt.Sprintf("%s/%s", ResourceTypeVolume, volume.Name)
		if _, exists := resourceMap[volumeKey]; exists {
			dependencies = append(dependencies, volumeKey)
		}
	}

	// Secret dependencies
	for _, secret := range container.Spec.Secrets {
		secretKey := fmt.Sprintf("%s/%s", ResourceTypeSecret, secret.Name)
		if _, exists := resourceMap[secretKey]; exists {
			dependencies = append(dependencies, secretKey)
		}
	}

	return dependencies
}

// extractPodDependencies extracts dependencies specific to pods
func (dr *DefaultDependencyResolver) extractPodDependencies(pod *PodResource, resourceMap map[string]Resource) []string {
	dependencies := make([]string, 0)

	// Container dependencies (pods depend on their containers)
	for _, containerName := range pod.Spec.Containers {
		containerKey := fmt.Sprintf("%s/%s", ResourceTypeContainer, containerName)
		if _, exists := resourceMap[containerKey]; exists {
			dependencies = append(dependencies, containerKey)
		}
	}

	return dependencies
}

// validateGraph validates the dependency graph for circular dependencies
func (dr *DefaultDependencyResolver) validateGraph(graph *DependencyGraph) error {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	for nodeKey := range graph.Nodes {
		if !visited[nodeKey] {
			if dr.hasCycle(nodeKey, graph, visited, recStack) {
				return fmt.Errorf("circular dependency detected involving resource '%s'", nodeKey)
			}
		}
	}

	return nil
}

// hasCycle performs DFS to detect cycles in the dependency graph
func (dr *DefaultDependencyResolver) hasCycle(nodeKey string, graph *DependencyGraph, visited, recStack map[string]bool) bool {
	visited[nodeKey] = true
	recStack[nodeKey] = true

	for _, depKey := range graph.Edges[nodeKey] {
		if !visited[depKey] {
			if dr.hasCycle(depKey, graph, visited, recStack) {
				return true
			}
		} else if recStack[depKey] {
			return true
		}
	}

	recStack[nodeKey] = false
	return false
}

// topologicalSort performs topological sorting using Kahn's algorithm
func (dr *DefaultDependencyResolver) topologicalSort(graph *DependencyGraph, reverse bool) ([][]Resource, error) {
	// Calculate in-degrees (how many dependencies each resource has)
	inDegree := make(map[string]int)
	for nodeKey := range graph.Nodes {
		inDegree[nodeKey] = len(graph.Edges[nodeKey])
	}

	var result [][]Resource
	queue := make([]string, 0)

	// Find all resources with no dependencies
	for nodeKey, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, nodeKey)
		}
	}

	for len(queue) > 0 {
		// Process all resources at the current level
		currentLevel := make([]Resource, 0)
		nextQueue := make([]string, 0)

		// Sort queue for consistent ordering
		sort.Strings(queue)

		for _, nodeKey := range queue {
			node := graph.Nodes[nodeKey]
			currentLevel = append(currentLevel, node.Resource)

			// Reduce in-degree for resources that depend on this one
			for _, dependentKey := range node.Dependents {
				inDegree[dependentKey]--
				if inDegree[dependentKey] == 0 {
					nextQueue = append(nextQueue, dependentKey)
				}
			}
		}

		result = append(result, currentLevel)
		queue = nextQueue
	}

	// Check if all resources were processed (no cycles)
	totalProcessed := 0
	for _, level := range result {
		totalProcessed += len(level)
	}
	if totalProcessed != len(graph.Nodes) {
		return nil, fmt.Errorf("circular dependency detected in resource graph")
	}

	return result, nil
}

// getResourceKey generates a unique key for a resource
func (dr *DefaultDependencyResolver) getResourceKey(resource Resource) string {
	return fmt.Sprintf("%s/%s", resource.GetType(), resource.GetName())
}

// GetDependencyChain returns the full dependency chain for a resource
func (dr *DefaultDependencyResolver) GetDependencyChain(graph *DependencyGraph, resourceKey string) ([]string, error) {
	visited := make(map[string]bool)
	chain := make([]string, 0)

	if err := dr.buildDependencyChain(resourceKey, graph, visited, &chain); err != nil {
		return nil, err
	}

	return chain, nil
}

// buildDependencyChain recursively builds the dependency chain
func (dr *DefaultDependencyResolver) buildDependencyChain(resourceKey string, graph *DependencyGraph, visited map[string]bool, chain *[]string) error {
	if visited[resourceKey] {
		return fmt.Errorf("circular dependency detected in chain for resource '%s'", resourceKey)
	}

	visited[resourceKey] = true
	*chain = append(*chain, resourceKey)

	node, exists := graph.Nodes[resourceKey]
	if !exists {
		return fmt.Errorf("resource '%s' not found in dependency graph", resourceKey)
	}

	for _, depKey := range node.Dependencies {
		if err := dr.buildDependencyChain(depKey, graph, visited, chain); err != nil {
			return err
		}
	}

	visited[resourceKey] = false
	return nil
}

// GetResourcesWithoutDependencies returns resources that have no dependencies
func (dr *DefaultDependencyResolver) GetResourcesWithoutDependencies(graph *DependencyGraph) []Resource {
	resources := make([]Resource, 0)

	for _, node := range graph.Nodes {
		if len(node.Dependencies) == 0 {
			resources = append(resources, node.Resource)
		}
	}

	return resources
}

// GetResourcesWithoutDependents returns resources that have no dependents (can be safely deleted)
func (dr *DefaultDependencyResolver) GetResourcesWithoutDependents(graph *DependencyGraph) []Resource {
	resources := make([]Resource, 0)

	for _, node := range graph.Nodes {
		if len(node.Dependents) == 0 {
			resources = append(resources, node.Resource)
		}
	}

	return resources
}
