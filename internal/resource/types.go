package resource

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ResourceType represents the type of a resource
type ResourceType string

const (
	ResourceTypeContainer ResourceType = "container"
	ResourceTypeNetwork   ResourceType = "network"
	ResourceTypeVolume    ResourceType = "volume"
	ResourceTypeSecret    ResourceType = "secret"
	ResourceTypePod       ResourceType = "pod"
)

// ResourceReference represents a reference to another resource
type ResourceReference struct {
	Type ResourceType `json:"type"`
	Name string       `json:"name"`
}

// Resource is the core interface that all managed resources must implement
type Resource interface {
	// GetType returns the resource type
	GetType() ResourceType

	// GetName returns the resource name
	GetName() string

	// GetLabels returns the resource labels
	GetLabels() map[string]string

	// GetDependencies returns the resources this resource depends on
	GetDependencies() []ResourceReference

	// SetLabels sets the labels for the resource
	SetLabels(labels map[string]string)
}

// BaseResource provides common functionality for all resources
type BaseResource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	ResourceType      ResourceType `json:"-"`
}

// GetType implements Resource interface
func (b *BaseResource) GetType() ResourceType {
	return b.ResourceType
}

// GetName implements Resource interface
func (b *BaseResource) GetName() string {
	return b.ObjectMeta.Name
}

// GetLabels implements Resource interface
func (b *BaseResource) GetLabels() map[string]string {
	if b.ObjectMeta.Labels == nil {
		return make(map[string]string)
	}
	return b.ObjectMeta.Labels
}

// SetLabels implements Resource interface
func (b *BaseResource) SetLabels(labels map[string]string) {
	b.ObjectMeta.Labels = labels
}

// GetDependencies provides a default implementation that returns no dependencies
// Resources with dependencies should override this method
func (b *BaseResource) GetDependencies() []ResourceReference {
	return []ResourceReference{}
}

// ResourceManager defines the interface for managing a specific resource type
type ResourceManager interface {
	// GetDesiredState extracts resources of this type from manifests
	GetDesiredState(manifests []Resource) ([]Resource, error)

	// GetActualState retrieves current resources of this type from the system
	GetActualState(ctx context.Context, chartName string) ([]Resource, error)

	// CreateResource creates a new resource
	CreateResource(ctx context.Context, resource Resource) error

	// UpdateResource updates an existing resource
	UpdateResource(ctx context.Context, desired, actual Resource) error

	// DeleteResource deletes a resource
	DeleteResource(ctx context.Context, resource Resource) error

	// CompareResources compares desired vs actual resource and returns true if they match
	CompareResources(desired, actual Resource) (bool, error)

	// GetResourceType returns the type of resources this manager handles
	GetResourceType() ResourceType
}
