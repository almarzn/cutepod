package resource

// NetworkResource represents a network resource that implements the Resource interface
type NetworkResource struct {
	BaseResource `json:",inline"`
	Spec         CuteNetworkSpec `json:"spec"`
}

// CuteNetworkSpec defines the specification for a network
type CuteNetworkSpec struct {
	Driver  string            `json:"driver,omitempty"`
	Options map[string]string `json:"options,omitempty"`
	Subnet  string            `json:"subnet,omitempty"`
	Gateway string            `json:"gateway,omitempty"`
}

// NewNetworkResource creates a new NetworkResource
func NewNetworkResource() *NetworkResource {
	return &NetworkResource{
		BaseResource: BaseResource{
			ResourceType: ResourceTypeNetwork,
		},
	}
}

// GetDependencies returns the resources this network depends on
// Networks typically don't depend on other resources
func (n *NetworkResource) GetDependencies() []ResourceReference {
	return []ResourceReference{}
}
