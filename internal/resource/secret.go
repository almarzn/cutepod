package resource

import (
	"encoding/base64"
	"fmt"
)

// SecretResource represents a secret resource that implements the Resource interface
type SecretResource struct {
	BaseResource `json:",inline"`
	Spec         CuteSecretSpec `json:"spec"`
}

// CuteSecretSpec defines the specification for a secret
type CuteSecretSpec struct {
	Type SecretType        `json:"type,omitempty"`
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
		BaseResource: BaseResource{
			ResourceType: ResourceTypeSecret,
		},
	}
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
