package resource

import (
	"context"
	"cutepod/internal/labels"
	"cutepod/internal/podman"
	"fmt"
	"os"
	"path/filepath"
)

// VolumeManager implements ResourceManager for volume resources
type VolumeManager struct {
	client podman.PodmanClient
}

// NewVolumeManager creates a new VolumeManager
func NewVolumeManager(client podman.PodmanClient) *VolumeManager {
	return &VolumeManager{
		client: client,
	}
}

// GetResourceType returns the resource type this manager handles
func (vm *VolumeManager) GetResourceType() ResourceType {
	return ResourceTypeVolume
}

// GetDesiredState extracts volume resources from manifests
func (vm *VolumeManager) GetDesiredState(manifests []Resource) ([]Resource, error) {
	var volumes []Resource

	for _, manifest := range manifests {
		if manifest.GetType() == ResourceTypeVolume {
			volumes = append(volumes, manifest)
		}
	}

	return volumes, nil
}

// GetActualState retrieves current volume resources from Podman
func (vm *VolumeManager) GetActualState(ctx context.Context, chartName string) ([]Resource, error) {
	connectedClient := podman.NewConnectedClient(vm.client)
	defer connectedClient.Close()

	podmanClient, err := connectedClient.GetClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to podman: %w", err)
	}

	volumes, err := podmanClient.ListVolumes(
		ctx,
		map[string][]string{
			"label": {labels.GetChartLabelValue(chartName)},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("unable to list volumes: %w", err)
	}

	var resources []Resource
	for _, volume := range volumes {
		// Convert Podman volume to VolumeResource
		resource := vm.convertPodmanVolumeToResource(volume)
		resources = append(resources, resource)
	}

	return resources, nil
}

// CreateResource creates a new volume resource
func (vm *VolumeManager) CreateResource(ctx context.Context, resource Resource) error {
	volume, ok := resource.(*VolumeResource)
	if !ok {
		return fmt.Errorf("expected VolumeResource, got %T", resource)
	}

	connectedClient := podman.NewConnectedClient(vm.client)
	defer connectedClient.Close()

	podmanClient, err := connectedClient.GetClient(ctx)
	if err != nil {
		return fmt.Errorf("unable to connect to podman: %w", err)
	}

	// Handle bind mounts differently from named volumes
	if volume.Spec.Type == VolumeTypeBind {
		return vm.createBindMount(volume)
	}

	// Create volume spec for named volumes
	spec := vm.buildVolumeSpec(volume)

	// Create volume
	_, err = podmanClient.CreateVolume(ctx, spec)
	if err != nil {
		return fmt.Errorf("unable to create volume: %w", err)
	}

	return nil
}

// UpdateResource updates an existing volume resource
func (vm *VolumeManager) UpdateResource(ctx context.Context, desired, actual Resource) error {
	// For volumes, update typically means recreate
	// First remove the existing volume, then create the new one
	if err := vm.DeleteResource(ctx, actual); err != nil {
		return fmt.Errorf("unable to remove existing volume for update: %w", err)
	}

	if err := vm.CreateResource(ctx, desired); err != nil {
		return fmt.Errorf("unable to create updated volume: %w", err)
	}

	return nil
}

// DeleteResource deletes a volume resource
func (vm *VolumeManager) DeleteResource(ctx context.Context, resource Resource) error {
	volume, ok := resource.(*VolumeResource)
	if !ok {
		return fmt.Errorf("expected VolumeResource, got %T", resource)
	}

	// For bind mounts, we don't need to delete anything from Podman
	if volume.Spec.Type == VolumeTypeBind {
		return nil
	}

	connectedClient := podman.NewConnectedClient(vm.client)
	defer connectedClient.Close()

	podmanClient, err := connectedClient.GetClient(ctx)
	if err != nil {
		return fmt.Errorf("unable to connect to podman: %w", err)
	}

	return podmanClient.RemoveVolume(ctx, volume.GetName())
}

// CompareResources compares desired vs actual volume resource
func (vm *VolumeManager) CompareResources(desired, actual Resource) (bool, error) {
	desiredVolume, ok := desired.(*VolumeResource)
	if !ok {
		return false, fmt.Errorf("expected VolumeResource for desired, got %T", desired)
	}

	actualVolume, ok := actual.(*VolumeResource)
	if !ok {
		return false, fmt.Errorf("expected VolumeResource for actual, got %T", actual)
	}

	// Compare key fields that would require recreation
	if desiredVolume.Spec.Type != actualVolume.Spec.Type {
		return false, nil
	}

	if desiredVolume.Spec.Driver != actualVolume.Spec.Driver {
		return false, nil
	}

	if desiredVolume.Spec.HostPath != actualVolume.Spec.HostPath {
		return false, nil
	}

	// Compare options
	if !vm.compareOptions(desiredVolume.Spec.Options, actualVolume.Spec.Options) {
		return false, nil
	}

	return true, nil
}

// Helper methods

func (vm *VolumeManager) convertPodmanVolumeToResource(volume podman.VolumeInfo) *VolumeResource {
	resource := NewVolumeResource()
	resource.ObjectMeta.Name = volume.Name
	resource.SetLabels(volume.Labels)

	// Convert volume spec
	resource.Spec.Driver = volume.Driver
	resource.Spec.Options = volume.Options

	// Determine volume type based on driver or options
	if volume.Driver == "local" {
		// Check if it's a bind mount by looking at options
		if device, exists := volume.Options["device"]; exists && device != "" {
			resource.Spec.Type = VolumeTypeBind
			resource.Spec.HostPath = device
		} else {
			resource.Spec.Type = VolumeTypeVolume
		}
	} else {
		resource.Spec.Type = VolumeTypeVolume
	}

	return resource
}

func (vm *VolumeManager) buildVolumeSpec(volume *VolumeResource) podman.VolumeSpec {
	spec := podman.VolumeSpec{
		Name:    volume.GetName(),
		Driver:  volume.Spec.Driver,
		Options: volume.Spec.Options,
		Labels:  volume.GetLabels(),
	}

	// Set default driver if not specified
	if spec.Driver == "" {
		spec.Driver = "local"
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

func (vm *VolumeManager) createBindMount(volume *VolumeResource) error {
	if volume.Spec.HostPath == "" {
		return fmt.Errorf("hostPath is required for bind mount volumes")
	}

	// Ensure the host path exists
	if err := os.MkdirAll(volume.Spec.HostPath, 0755); err != nil {
		return fmt.Errorf("unable to create host path %s: %w", volume.Spec.HostPath, err)
	}

	// Validate that the path is absolute
	if !filepath.IsAbs(volume.Spec.HostPath) {
		return fmt.Errorf("hostPath must be an absolute path, got: %s", volume.Spec.HostPath)
	}

	return nil
}

func (vm *VolumeManager) compareOptions(desired, actual map[string]string) bool {
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
