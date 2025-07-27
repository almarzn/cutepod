package resource

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/goccy/go-yaml"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ManifestParser handles parsing of YAML manifests into resources
type ManifestParser struct {
	registry *ManifestRegistry
}

// NewManifestParser creates a new manifest parser
func NewManifestParser() *ManifestParser {
	return &ManifestParser{
		registry: NewManifestRegistry(),
	}
}

// ParseManifest parses a single YAML manifest and adds it to the registry
func (p *ManifestParser) ParseManifest(content []byte) error {
	// Split multi-document YAML
	documents := bytes.Split(content, []byte("---"))

	for _, doc := range documents {
		doc = bytes.TrimSpace(doc)
		if len(doc) == 0 {
			continue
		}

		resource, err := p.parseDocument(doc)
		if err != nil {
			return err
		}

		if resource != nil {
			if err := p.registry.AddResource(resource); err != nil {
				return fmt.Errorf("failed to add resource to registry: %w", err)
			}
		}
	}

	return nil
}

// parseDocument parses a single YAML document into a resource
func (p *ManifestParser) parseDocument(content []byte) (Resource, error) {
	// First, parse the basic metadata to determine the resource type
	var base struct {
		metav1.TypeMeta   `json:",inline"`
		metav1.ObjectMeta `json:"metadata,omitempty"`
	}

	if err := yaml.Unmarshal(content, &base); err != nil {
		return nil, fmt.Errorf("failed to parse YAML metadata: %w", err)
	}

	// Skip empty documents or those without Kind
	if base.Kind == "" {
		return nil, nil
	}

	// Parse based on the Kind
	switch base.Kind {
	case "CuteContainer":
		return p.parseContainer(content)
	case "CuteNetwork":
		return p.parseNetwork(content)
	case "CuteVolume":
		return p.parseVolume(content)
	case "CuteSecret":
		return p.parseSecret(content)
	case "CutePod":
		return p.parsePod(content)
	default:
		return nil, fmt.Errorf("unsupported resource kind: %s", base.Kind)
	}
}

// parseContainer parses a CuteContainer resource
func (p *ManifestParser) parseContainer(content []byte) (Resource, error) {
	var container ContainerResource
	if err := yaml.Unmarshal(content, &container); err != nil {
		return nil, fmt.Errorf("failed to parse CuteContainer: %w", err)
	}

	// Set the resource type
	container.ResourceType = ResourceTypeContainer

	// Validate the container
	if err := p.validateContainer(&container, string(content)); err != nil {
		return nil, err
	}

	return &container, nil
}

// parseNetwork parses a CuteNetwork resource
func (p *ManifestParser) parseNetwork(content []byte) (Resource, error) {
	var network NetworkResource
	if err := yaml.Unmarshal(content, &network); err != nil {
		return nil, fmt.Errorf("failed to parse CuteNetwork: %w", err)
	}

	// Set the resource type
	network.ResourceType = ResourceTypeNetwork

	// Validate the network
	if err := p.validateNetwork(&network); err != nil {
		return nil, err
	}

	return &network, nil
}

// parseVolume parses a CuteVolume resource
func (p *ManifestParser) parseVolume(content []byte) (Resource, error) {
	var volume VolumeResource
	if err := yaml.Unmarshal(content, &volume); err != nil {
		return nil, fmt.Errorf("failed to parse CuteVolume: %w", err)
	}

	// Set the resource type
	volume.ResourceType = ResourceTypeVolume

	// Validate the volume
	if err := p.validateVolume(&volume); err != nil {
		return nil, err
	}

	return &volume, nil
}

// parseSecret parses a CuteSecret resource
func (p *ManifestParser) parseSecret(content []byte) (Resource, error) {
	var secret SecretResource
	if err := yaml.Unmarshal(content, &secret); err != nil {
		return nil, fmt.Errorf("failed to parse CuteSecret: %w", err)
	}

	// Set the resource type
	secret.ResourceType = ResourceTypeSecret

	// Validate the secret
	if err := p.validateSecret(&secret); err != nil {
		return nil, err
	}

	return &secret, nil
}

// parsePod parses a CutePod resource
func (p *ManifestParser) parsePod(content []byte) (Resource, error) {
	var pod PodResource
	if err := yaml.Unmarshal(content, &pod); err != nil {
		return nil, fmt.Errorf("failed to parse CutePod: %w", err)
	}

	// Set the resource type
	pod.ResourceType = ResourceTypePod

	// Validate the pod
	if err := p.validatePod(&pod); err != nil {
		return nil, err
	}

	return &pod, nil
}

// validateContainer validates a container resource
func (p *ManifestParser) validateContainer(container *ContainerResource, yml string) error {
	if container.GetName() == "" {
		return fmt.Errorf("container name cannot be empty")
	}

	if container.Spec.Image == "" {
		return fmt.Errorf("container image cannot be empty")
	}

	// Use the existing validation from the container resource
	errors := container.Validate(yml)
	if len(errors) > 0 {
		var errorMessages []string
		for _, err := range errors {
			errorMessages = append(errorMessages, err.Error())
		}
		return fmt.Errorf("container validation failed:\n%s", strings.Join(errorMessages, "\n"))
	}

	return nil
}

// validateNetwork validates a network resource
func (p *ManifestParser) validateNetwork(network *NetworkResource) error {
	if network.GetName() == "" {
		return fmt.Errorf("network name cannot be empty")
	}

	if network.Spec.Driver == "" {
		network.Spec.Driver = "bridge" // Default driver
	}

	return nil
}

// validateVolume validates a volume resource
func (p *ManifestParser) validateVolume(volume *VolumeResource) error {
	if volume.GetName() == "" {
		return fmt.Errorf("volume name cannot be empty")
	}

	if volume.Spec.Type == "" {
		volume.Spec.Type = VolumeTypeVolume // Default type
	}

	// Use the new validation method
	if errs := volume.Validate(); len(errs) > 0 {
		// Return the first validation error
		return errs[0]
	}

	return nil
}

// validateSecret validates a secret resource
func (p *ManifestParser) validateSecret(secret *SecretResource) error {
	if secret.GetName() == "" {
		return fmt.Errorf("secret name cannot be empty")
	}

	if len(secret.Spec.Data) == 0 {
		return fmt.Errorf("secret must contain at least one data entry")
	}

	return nil
}

// validatePod validates a pod resource
func (p *ManifestParser) validatePod(pod *PodResource) error {
	if pod.GetName() == "" {
		return fmt.Errorf("pod name cannot be empty")
	}

	if len(pod.Spec.Containers) == 0 {
		return fmt.Errorf("pod must contain at least one container reference")
	}

	return nil
}

// GetRegistry returns the populated registry
func (p *ManifestParser) GetRegistry() *ManifestRegistry {
	return p.registry
}
