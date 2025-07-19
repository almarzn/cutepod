package resource

// PodResource represents a pod resource that implements the Resource interface
type PodResource struct {
	BaseResource `json:",inline"`
	Spec         CutePodSpec `json:"spec"`
}

// CutePodSpec defines the specification for a pod
type CutePodSpec struct {
	Containers    []string `json:"containers"`              // Container name references
	RestartPolicy string   `json:"restartPolicy,omitempty"` // Always, OnFailure, Never
}

// NewPodResource creates a new PodResource
func NewPodResource() *PodResource {
	return &PodResource{
		BaseResource: BaseResource{
			ResourceType: ResourceTypePod,
		},
	}
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
