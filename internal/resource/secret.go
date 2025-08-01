package resource

import (
	"encoding/base64"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Namespaced,shortName=cs
// +kubebuilder:subresource:status

// SecretResource represents a secret resource that implements the Resource interface
type SecretResource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec CuteSecretSpec `json:"spec"`
}

// +kubebuilder:object:generate=true

// CuteSecretSpec defines the specification for a secret
type CuteSecretSpec struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:="opaque"
	Type SecretType `json:"type,omitempty"`
	// +kubebuilder:validation:Required
	Data map[string]string `json:"data,omitempty"` // Base64 encoded data
}

// SecretType represents the type of secret
type SecretType string

const (
	SecretTypeOpaque SecretType = "opaque"
)

// NewSecretResource creates a new SecretResource
func NewSecretResource() *SecretResource {
	return &SecretResource{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "cutepod/v1alpha1",
			Kind:       "CuteSecret",
		},
	}
}

// GetType implements Resource interface
func (s *SecretResource) GetType() ResourceType {
	return ResourceTypeSecret
}

// GetName implements Resource interface
func (s *SecretResource) GetName() string {
	return s.ObjectMeta.Name
}

// GetLabels implements Resource interface
func (s *SecretResource) GetLabels() map[string]string {
	if s.ObjectMeta.Labels == nil {
		return make(map[string]string)
	}
	return s.ObjectMeta.Labels
}

// SetLabels implements Resource interface
func (s *SecretResource) SetLabels(labels map[string]string) {
	s.ObjectMeta.Labels = labels
}

// GetDependencies returns the resources this secret depends on
// Secrets typically don't depend on other resources
func (s *SecretResource) GetDependencies() []ResourceReference {
	return []ResourceReference{}
}

// GetDecodedData returns the base64-decoded secret data
func (s *SecretResource) GetDecodedData() (map[string][]byte, error) {
	decoded := make(map[string][]byte)

	for key, value := range s.Spec.Data {
		data, err := base64.StdEncoding.DecodeString(value)
		if err != nil {
			return nil, fmt.Errorf("failed to decode base64 data for key '%s': %w", key, err)
		}
		decoded[key] = data
	}

	return decoded, nil
}

// SetData sets the secret data with base64 encoding
func (s *SecretResource) SetData(data map[string][]byte) {
	if s.Spec.Data == nil {
		s.Spec.Data = make(map[string]string)
	}

	for key, value := range data {
		s.Spec.Data[key] = base64.StdEncoding.EncodeToString(value)
	}
}
