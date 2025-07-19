package resource

import (
	"fmt"
	"sort"
)

// ManifestRegistry manages parsed resources and their dependencies
type ManifestRegistry struct {
	Resources    map[string]Resource
	Dependencies map[string][]string
}

// NewManifestRegistry creates a new empty registry
func NewManifestRegistry() *ManifestRegistry {
	return &ManifestRegistry{
		Resources:    make(map[string]Resource),
		Dependencies: make(map[string][]string),
	}
}

// AddResource adds a resource to the registry and builds its dependency graph
func (r *ManifestRegistry) AddResource(resource Resource) error {
	name := resource.GetName()
	if name == "" {
		return fmt.Errorf("resource name cannot be empty")
	}

	// Check for duplicate names
	if _, exists := r.Resources[name]; exists {
		return fmt.Errorf("resource with name '%s' already exists", name)
	}

	// Add the resource
	r.Resources[name] = resource

	// Build dependency list
	var deps []string
	for _, dep := range resource.GetDependencies() {
		deps = append(deps, dep.Name)
	}
	r.Dependencies[name] = deps

	return nil
}

// GetResource retrieves a resource by name
func (r *ManifestRegistry) GetResource(name string) (Resource, bool) {
	resource, exists := r.Resources[name]
	return resource, exists
}

// GetResourcesByType returns all resources of a specific type
func (r *ManifestRegistry) GetResourcesByType(resourceType ResourceType) []Resource {
	var resources []Resource
	for _, resource := range r.Resources {
		if resource.GetType() == resourceType {
			resources = append(resources, resource)
		}
	}
	return resources
}

// GetAllResources returns all resources in the registry
func (r *ManifestRegistry) GetAllResources() []Resource {
	var resources []Resource
	for _, resource := range r.Resources {
		resources = append(resources, resource)
	}
	return resources
}

// ValidateDependencies checks that all resource dependencies exist and detects circular dependencies
func (r *ManifestRegistry) ValidateDependencies() error {
	// Check that all dependencies exist
	for resourceName, deps := range r.Dependencies {
		for _, depName := range deps {
			if _, exists := r.Resources[depName]; !exists {
				return fmt.Errorf("resource '%s' depends on '%s' which does not exist", resourceName, depName)
			}
		}
	}

	// Check for circular dependencies using DFS
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	for resourceName := range r.Resources {
		if !visited[resourceName] {
			if r.hasCycle(resourceName, visited, recStack) {
				return fmt.Errorf("circular dependency detected involving resource '%s'", resourceName)
			}
		}
	}

	return nil
}

// hasCycle performs DFS to detect cycles in the dependency graph
func (r *ManifestRegistry) hasCycle(resourceName string, visited, recStack map[string]bool) bool {
	visited[resourceName] = true
	recStack[resourceName] = true

	for _, dep := range r.Dependencies[resourceName] {
		if !visited[dep] {
			if r.hasCycle(dep, visited, recStack) {
				return true
			}
		} else if recStack[dep] {
			return true
		}
	}

	recStack[resourceName] = false
	return false
}

// GetCreationOrder returns resources in dependency order for creation
func (r *ManifestRegistry) GetCreationOrder() ([][]Resource, error) {
	if err := r.ValidateDependencies(); err != nil {
		return nil, err
	}

	return r.topologicalSort()
}

// GetDeletionOrder returns resources in reverse dependency order for deletion
func (r *ManifestRegistry) GetDeletionOrder() ([][]Resource, error) {
	creationOrder, err := r.GetCreationOrder()
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

// topologicalSort performs topological sorting using Kahn's algorithm
func (r *ManifestRegistry) topologicalSort() ([][]Resource, error) {
	// Calculate in-degrees (how many resources depend on each resource)
	inDegree := make(map[string]int)
	for resourceName := range r.Resources {
		inDegree[resourceName] = 0
	}

	// For each resource, increment the in-degree of its dependencies
	for resourceName, deps := range r.Dependencies {
		for range deps {
			inDegree[resourceName]++ // The resource depends on deps, so resource has higher in-degree
		}
	}

	var result [][]Resource
	queue := make([]string, 0)

	// Find all resources with no dependencies
	for resourceName, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, resourceName)
		}
	}

	for len(queue) > 0 {
		// Process all resources at the current level
		currentLevel := make([]Resource, 0)
		nextQueue := make([]string, 0)

		// Sort queue for consistent ordering
		sort.Strings(queue)

		for _, resourceName := range queue {
			resource := r.Resources[resourceName]
			currentLevel = append(currentLevel, resource)

			// Reduce in-degree for resources that depend on this one
			for otherResource, deps := range r.Dependencies {
				for _, depName := range deps {
					if depName == resourceName {
						inDegree[otherResource]--
						if inDegree[otherResource] == 0 {
							nextQueue = append(nextQueue, otherResource)
						}
					}
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
	if totalProcessed != len(r.Resources) {
		return nil, fmt.Errorf("circular dependency detected in resource graph")
	}

	return result, nil
}

// ObjectReference represents a reference to another resource by name
type ObjectReference struct {
	Name string       `json:"name"`
	Type ResourceType `json:"type,omitempty"`
}

// ResolveReference resolves an object reference to the actual resource
func (r *ManifestRegistry) ResolveReference(ref ObjectReference) (Resource, error) {
	resource, exists := r.GetResource(ref.Name)
	if !exists {
		return nil, fmt.Errorf("referenced resource '%s' not found", ref.Name)
	}

	// If type is specified, validate it matches
	if ref.Type != "" && resource.GetType() != ref.Type {
		return nil, fmt.Errorf("referenced resource '%s' is of type '%s', expected '%s'",
			ref.Name, resource.GetType(), ref.Type)
	}

	return resource, nil
}
