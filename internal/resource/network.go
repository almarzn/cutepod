package resource

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Namespaced,shortName=cn
// +kubebuilder:subresource:status

// NetworkResource represents a network resource that implements the Resource interface
type NetworkResource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec CuteNetworkSpec `json:"spec"`
}

// +kubebuilder:object:generate=true

// CuteNetworkSpec defines the specification for a network
type CuteNetworkSpec struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:="bridge"
	Driver  string            `json:"driver,omitempty"`
	Options map[string]string `json:"options,omitempty"`
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Pattern="^([0-9]{1,3}\\.){3}[0-9]{1,3}/[0-9]{1,2}$"
	Subnet string `json:"subnet,omitempty"`
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Pattern="^([0-9]{1,3}\\.){3}[0-9]{1,3}$"
	Gateway string `json:"gateway,omitempty"`
}

// NewNetworkResource creates a new NetworkResource
func NewNetworkResource() *NetworkResource {
	return &NetworkResource{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "cutepod/v1alpha1",
			Kind:       "CuteNetwork",
		},
	}
}

// GetType implements Resource interface
func (n *NetworkResource) GetType() ResourceType {
	return ResourceTypeNetwork
}

// GetName implements Resource interface
func (n *NetworkResource) GetName() string {
	return n.ObjectMeta.Name
}

// GetLabels implements Resource interface
func (n *NetworkResource) GetLabels() map[string]string {
	if n.ObjectMeta.Labels == nil {
		return make(map[string]string)
	}
	return n.ObjectMeta.Labels
}

// SetLabels implements Resource interface
func (n *NetworkResource) SetLabels(labels map[string]string) {
	n.ObjectMeta.Labels = labels
}

// GetDependencies returns the resources this network depends on
// Networks typically don't depend on other resources
func (n *NetworkResource) GetDependencies() []ResourceReference {
	return []ResourceReference{}
}
