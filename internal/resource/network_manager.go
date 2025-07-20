package resource

import (
	"context"
	"cutepod/internal/labels"
	"cutepod/internal/podman"
	"fmt"
)

// NetworkManager implements ResourceManager for network resources
type NetworkManager struct {
	client podman.PodmanClient
}

// NewNetworkManager creates a new NetworkManager
func NewNetworkManager(client podman.PodmanClient) *NetworkManager {
	return &NetworkManager{
		client: client,
	}
}

// GetResourceType returns the resource type this manager handles
func (nm *NetworkManager) GetResourceType() ResourceType {
	return ResourceTypeNetwork
}

// GetDesiredState extracts network resources from manifests
func (nm *NetworkManager) GetDesiredState(manifests []Resource) ([]Resource, error) {
	var networks []Resource

	for _, manifest := range manifests {
		if manifest.GetType() == ResourceTypeNetwork {
			networks = append(networks, manifest)
		}
	}

	return networks, nil
}

// GetActualState retrieves current network resources from Podman
func (nm *NetworkManager) GetActualState(ctx context.Context, chartName string) ([]Resource, error) {
	connectedClient := podman.NewConnectedClient(nm.client)
	defer connectedClient.Close()

	podmanClient, err := connectedClient.GetClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to podman: %w", err)
	}

	networks, err := podmanClient.ListNetworks(
		ctx,
		map[string][]string{
			"label": {labels.GetChartLabelValue(chartName)},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("unable to list networks: %w", err)
	}

	var resources []Resource
	for _, network := range networks {
		// Convert Podman network to NetworkResource
		resource := nm.convertPodmanNetworkToResource(network)
		resources = append(resources, resource)
	}

	return resources, nil
}

// CreateResource creates a new network resource
func (nm *NetworkManager) CreateResource(ctx context.Context, resource Resource) error {
	network, ok := resource.(*NetworkResource)
	if !ok {
		return fmt.Errorf("expected NetworkResource, got %T", resource)
	}

	connectedClient := podman.NewConnectedClient(nm.client)
	defer connectedClient.Close()

	podmanClient, err := connectedClient.GetClient(ctx)
	if err != nil {
		return fmt.Errorf("unable to connect to podman: %w", err)
	}

	// Create network spec
	spec := nm.buildNetworkSpec(network)

	// Create network
	_, err = podmanClient.CreateNetwork(ctx, spec)
	if err != nil {
		return fmt.Errorf("unable to create network: %w", err)
	}

	return nil
}

// UpdateResource updates an existing network resource
func (nm *NetworkManager) UpdateResource(ctx context.Context, desired, actual Resource) error {
	// For networks, update typically means recreate
	// First remove the existing network, then create the new one
	if err := nm.DeleteResource(ctx, actual); err != nil {
		return fmt.Errorf("unable to remove existing network for update: %w", err)
	}

	if err := nm.CreateResource(ctx, desired); err != nil {
		return fmt.Errorf("unable to create updated network: %w", err)
	}

	return nil
}

// DeleteResource deletes a network resource
func (nm *NetworkManager) DeleteResource(ctx context.Context, resource Resource) error {
	network, ok := resource.(*NetworkResource)
	if !ok {
		return fmt.Errorf("expected NetworkResource, got %T", resource)
	}

	connectedClient := podman.NewConnectedClient(nm.client)
	defer connectedClient.Close()

	podmanClient, err := connectedClient.GetClient(ctx)
	if err != nil {
		return fmt.Errorf("unable to connect to podman: %w", err)
	}

	return podmanClient.RemoveNetwork(ctx, network.GetName())
}

// CompareResources compares desired vs actual network resource
func (nm *NetworkManager) CompareResources(desired, actual Resource) (bool, error) {
	desiredNetwork, ok := desired.(*NetworkResource)
	if !ok {
		return false, fmt.Errorf("expected NetworkResource for desired, got %T", desired)
	}

	actualNetwork, ok := actual.(*NetworkResource)
	if !ok {
		return false, fmt.Errorf("expected NetworkResource for actual, got %T", actual)
	}

	// Compare key fields that would require recreation
	if desiredNetwork.Spec.Driver != actualNetwork.Spec.Driver {
		return false, nil
	}

	if desiredNetwork.Spec.Subnet != actualNetwork.Spec.Subnet {
		return false, nil
	}

	if desiredNetwork.Spec.Gateway != actualNetwork.Spec.Gateway {
		return false, nil
	}

	// Compare options
	if !nm.compareOptions(desiredNetwork.Spec.Options, actualNetwork.Spec.Options) {
		return false, nil
	}

	return true, nil
}

// Helper methods

func (nm *NetworkManager) convertPodmanNetworkToResource(network podman.NetworkInfo) *NetworkResource {
	resource := NewNetworkResource()
	resource.ObjectMeta.Name = network.Name
	resource.SetLabels(network.Labels)

	// Convert network spec
	resource.Spec.Driver = network.Driver
	resource.Spec.Options = network.Options
	resource.Spec.Subnet = network.Subnet

	return resource
}

func (nm *NetworkManager) buildNetworkSpec(network *NetworkResource) podman.NetworkSpec {
	spec := podman.NetworkSpec{
		Name:    network.GetName(),
		Driver:  network.Spec.Driver,
		Options: network.Spec.Options,
		Subnet:  network.Spec.Subnet,
		Labels:  network.GetLabels(),
	}

	// Set default driver if not specified
	if spec.Driver == "" {
		spec.Driver = "bridge"
	}

	// Initialize options map if nil
	if spec.Options == nil {
		spec.Options = make(map[string]string)
	}

	// Initialize labels map if nil
	if spec.Labels == nil {
		spec.Labels = make(map[string]string)
	}

	return spec
}

func (nm *NetworkManager) compareOptions(desired, actual map[string]string) bool {
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
