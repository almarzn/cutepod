package resource

// VolumeResource represents a volume resource that implements the Resource interface
type VolumeResource struct {
	BaseResource `json:",inline"`
	Spec         CuteVolumeSpec `json:"spec"`
}

// CuteVolumeSpec defines the specification for a volume
type CuteVolumeSpec struct {
	Type     VolumeType        `json:"type,omitempty"`
	Driver   string            `json:"driver,omitempty"`
	Options  map[string]string `json:"options,omitempty"`
	HostPath string            `json:"hostPath,omitempty"` // for bind mounts
}

// VolumeType represents the type of volume
type VolumeType string

const (
	VolumeTypeBind   VolumeType = "bind"
	VolumeTypeVolume VolumeType = "volume"
)

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
