package resource

// VolumeResource represents a volume resource that implements the Resource interface
type VolumeResource struct {
	BaseResource `json:",inline"`
	Spec         CuteVolumeSpec `json:"spec"`
}

// CuteVolumeSpec defines the specification for a volume
type CuteVolumeSpec struct {
	Driver  string            `json:"driver,omitempty"`
	Options map[string]string `json:"options,omitempty"`
}

// VolumeType represents the type of volume
type VolumeType string

// NewVolumeResource creates a new VolumeResource
func NewVolumeResource() *VolumeResource {
	return &VolumeResource{
		BaseResource: BaseResource{
			ResourceType: ResourceTypeVolume,
		},
	}
}

// GetDependencies returns the resources this volume depends on
// Volumes typically don't depend on other resources
func (v *VolumeResource) GetDependencies() []ResourceReference {
	return []ResourceReference{}
}
