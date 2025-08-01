package resource

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Namespaced,shortName=cp
// +kubebuilder:subresource:status

// PodResource represents a pod resource that implements the Resource interface
type PodResource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec CutePodSpec `json:"spec"`
}

// +kubebuilder:object:generate=true

// CutePodSpec defines the specification for a pod
type CutePodSpec struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	Containers []string `json:"containers"` // Container name references
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=Always;OnFailure;Never
	// +kubebuilder:default:="Always"
	RestartPolicy string `json:"restartPolicy,omitempty"` // Always, OnFailure, Never
}

// NewPodResource creates a new PodResource
func NewPodResource() *PodResource {
	return &PodResource{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "cutepod/v1alpha1",
			Kind:       "CutePod",
		},
	}
}

// GetType implements Resource interface
func (p *PodResource) GetType() ResourceType {
	return ResourceTypePod
}

// GetName implements Resource interface
func (p *PodResource) GetName() string {
	return p.ObjectMeta.Name
}

// GetLabels implements Resource interface
func (p *PodResource) GetLabels() map[string]string {
	if p.ObjectMeta.Labels == nil {
		return make(map[string]string)
	}
	return p.ObjectMeta.Labels
}

// SetLabels implements Resource interface
func (p *PodResource) SetLabels(labels map[string]string) {
	p.ObjectMeta.Labels = labels
}

// GetDependencies returns the resources this pod depends on
func (p *PodResource) GetDependencies() []ResourceReference {
	var deps []ResourceReference

	// Add container dependencies
	for _, containerName := range p.Spec.Containers {
		deps = append(deps, ResourceReference{
			Type: ResourceTypeContainer,
			Name: containerName,
		})
	}

	return deps
}
