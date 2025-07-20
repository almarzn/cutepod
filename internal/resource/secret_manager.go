package resource

import (
	"context"
	"cutepod/internal/podman"
	"fmt"
)

// SecretManager implements ResourceManager for secret resources
type SecretManager struct {
	client podman.PodmanClient
}

// NewSecretManager creates a new SecretManager
func NewSecretManager(client podman.PodmanClient) *SecretManager {
	return &SecretManager{
		client: client,
	}
}

// GetResourceType returns the resource type this manager handles
func (sm *SecretManager) GetResourceType() ResourceType {
	return ResourceTypeSecret
}

// GetDesiredState extracts secret resources from manifests
func (sm *SecretManager) GetDesiredState(manifests []Resource) ([]Resource, error) {
	var secrets []Resource

	for _, manifest := range manifests {
		if manifest.GetType() == ResourceTypeSecret {
			secrets = append(secrets, manifest)
		}
	}

	return secrets, nil
}

// GetActualState retrieves current secret resources from Podman
func (sm *SecretManager) GetActualState(ctx context.Context, namespace string) ([]Resource, error) {
	connectedClient := podman.NewConnectedClient(sm.client)
	defer connectedClient.Close()

	podmanClient, err := connectedClient.GetClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to podman: %w", err)
	}

	secrets, err := podmanClient.ListSecrets(
		ctx,
		map[string][]string{
			"label": {"cutepod.Namespace=" + namespace},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("unable to list secrets: %w", err)
	}

	var resources []Resource
	for _, secret := range secrets {
		// Convert Podman secret to SecretResource
		resource := sm.convertPodmanSecretToResource(secret)
		resources = append(resources, resource)
	}

	return resources, nil
}

// CreateResource creates a new secret resource
func (sm *SecretManager) CreateResource(ctx context.Context, resource Resource) error {
	secret, ok := resource.(*SecretResource)
	if !ok {
		return fmt.Errorf("expected SecretResource, got %T", resource)
	}

	connectedClient := podman.NewConnectedClient(sm.client)
	defer connectedClient.Close()

	podmanClient, err := connectedClient.GetClient(ctx)
	if err != nil {
		return fmt.Errorf("unable to connect to podman: %w", err)
	}

	// Get decoded data from the secret
	decodedData, err := secret.GetDecodedData()
	if err != nil {
		return fmt.Errorf("unable to decode secret data: %w", err)
	}

	// Create secret spec for each key-value pair
	// Note: Podman secrets store single values, so we need to create one secret per key
	// or combine all data into a single secret
	spec := sm.buildSecretSpec(secret, decodedData)

	// Create secret
	_, err = podmanClient.CreateSecret(ctx, spec)
	if err != nil {
		return fmt.Errorf("unable to create secret: %w", err)
	}

	return nil
}

// UpdateResource updates an existing secret resource
func (sm *SecretManager) UpdateResource(ctx context.Context, desired, actual Resource) error {
	desiredSecret, ok := desired.(*SecretResource)
	if !ok {
		return fmt.Errorf("expected SecretResource for desired, got %T", desired)
	}

	connectedClient := podman.NewConnectedClient(sm.client)
	defer connectedClient.Close()

	podmanClient, err := connectedClient.GetClient(ctx)
	if err != nil {
		return fmt.Errorf("unable to connect to podman: %w", err)
	}

	// Get decoded data from the desired secret
	decodedData, err := desiredSecret.GetDecodedData()
	if err != nil {
		return fmt.Errorf("unable to decode secret data: %w", err)
	}

	// Create secret spec
	spec := sm.buildSecretSpec(desiredSecret, decodedData)

	// Update secret (this will remove and recreate in the adapter)
	err = podmanClient.UpdateSecret(ctx, desiredSecret.GetName(), spec)
	if err != nil {
		return fmt.Errorf("unable to update secret: %w", err)
	}

	return nil
}

// DeleteResource deletes a secret resource
func (sm *SecretManager) DeleteResource(ctx context.Context, resource Resource) error {
	secret, ok := resource.(*SecretResource)
	if !ok {
		return fmt.Errorf("expected SecretResource, got %T", resource)
	}

	connectedClient := podman.NewConnectedClient(sm.client)
	defer connectedClient.Close()

	podmanClient, err := connectedClient.GetClient(ctx)
	if err != nil {
		return fmt.Errorf("unable to connect to podman: %w", err)
	}

	return podmanClient.RemoveSecret(ctx, secret.GetName())
}

// CompareResources compares desired vs actual secret resource
func (sm *SecretManager) CompareResources(desired, actual Resource) (bool, error) {
	desiredSecret, ok := desired.(*SecretResource)
	if !ok {
		return false, fmt.Errorf("expected SecretResource for desired, got %T", desired)
	}

	actualSecret, ok := actual.(*SecretResource)
	if !ok {
		return false, fmt.Errorf("expected SecretResource for actual, got %T", actual)
	}

	// Compare secret type
	if desiredSecret.Spec.Type != actualSecret.Spec.Type {
		return false, nil
	}

	// Compare secret data
	if !sm.compareSecretData(desiredSecret.Spec.Data, actualSecret.Spec.Data) {
		return false, nil
	}

	return true, nil
}

// Helper methods

func (sm *SecretManager) convertPodmanSecretToResource(secret podman.SecretInfo) *SecretResource {
	resource := NewSecretResource()
	resource.ObjectMeta.Name = secret.Name
	resource.SetLabels(secret.Labels)

	// Extract namespace from labels
	if namespace, exists := secret.Labels["cutepod.Namespace"]; exists {
		resource.SetNamespace(namespace)
	}

	// Set default secret type
	resource.Spec.Type = SecretTypeOpaque

	// Note: Podman doesn't expose secret data for security reasons,
	// so we can't populate the actual data. This is expected behavior.
	// The comparison will be based on metadata and labels only.
	resource.Spec.Data = make(map[string]string)

	return resource
}

func (sm *SecretManager) buildSecretSpec(secret *SecretResource, decodedData map[string][]byte) podman.SecretSpec {
	// Combine all secret data into a single JSON-like format
	// This is a common approach when dealing with multi-key secrets in Podman
	var combinedData []byte

	// If there's only one key, use its value directly
	if len(decodedData) == 1 {
		for _, value := range decodedData {
			combinedData = value
			break
		}
	} else {
		// For multiple keys, we'll create a simple key=value format
		// This is a limitation of Podman's secret model compared to Kubernetes
		var dataStr string
		for key, value := range decodedData {
			if dataStr != "" {
				dataStr += "\n"
			}
			dataStr += fmt.Sprintf("%s=%s", key, string(value))
		}
		combinedData = []byte(dataStr)
	}

	spec := podman.SecretSpec{
		Name:   secret.GetName(),
		Data:   combinedData,
		Labels: secret.GetLabels(),
	}

	// Initialize labels map if nil
	if spec.Labels == nil {
		spec.Labels = make(map[string]string)
	}

	return spec
}

func (sm *SecretManager) compareSecretData(desired, actual map[string]string) bool {
	if len(desired) != len(actual) {
		return false
	}

	for key, desiredValue := range desired {
		if actualValue, exists := actual[key]; !exists || actualValue != desiredValue {
			return false
		}
	}

	return true
}
