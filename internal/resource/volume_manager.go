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

	// Handle different volume types
	switch volume.Spec.Type {
	case VolumeTypeHostPath:
		return vm.createHostPathVolume(volume)
	case VolumeTypeEmptyDir:
		return vm.createEmptyDirVolume(volume)
	case VolumeTypeVolume:
		return vm.createNamedVolume(ctx, podmanClient, volume)
	case VolumeTypeBind:
		// Legacy support - treat as hostPath
		return vm.createBindMount(volume)
	default:
		return fmt.Errorf("unsupported volume type: %s", volume.Spec.Type)
	}

	return fmt.Errorf("unexpected code path - volume type should be handled above")
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

	// Handle different volume types for deletion
	switch volume.Spec.Type {
	case VolumeTypeHostPath, VolumeTypeBind:
		// For hostPath and bind mounts, we don't need to delete anything from Podman
		return nil
	case VolumeTypeEmptyDir:
		// For emptyDir, we should clean up the temporary directory
		return vm.deleteEmptyDirVolume(volume)
	case VolumeTypeVolume:
		// For named volumes, delete from Podman
		break
	default:
		return fmt.Errorf("unsupported volume type for deletion: %s", volume.Spec.Type)
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

	// Compare type-specific fields
	switch desiredVolume.Spec.Type {
	case VolumeTypeHostPath:
		if !vm.compareHostPathSpecs(desiredVolume.Spec.HostPath, actualVolume.Spec.HostPath) {
			return false, nil
		}
	case VolumeTypeEmptyDir:
		if !vm.compareEmptyDirSpecs(desiredVolume.Spec.EmptyDir, actualVolume.Spec.EmptyDir) {
			return false, nil
		}
	case VolumeTypeVolume:
		if !vm.compareVolumeSpecs(desiredVolume.Spec.Volume, actualVolume.Spec.Volume) {
			return false, nil
		}
	case VolumeTypeBind:
		// Legacy comparison - compare driver and options
		if desiredVolume.Spec.Driver != actualVolume.Spec.Driver {
			return false, nil
		}
		if !vm.compareOptions(desiredVolume.Spec.Options, actualVolume.Spec.Options) {
			return false, nil
		}
	}

	// Compare security context
	if !vm.compareSecurityContexts(desiredVolume.Spec.SecurityContext, actualVolume.Spec.SecurityContext) {
		return false, nil
	}

	return true, nil
}

// Helper methods

func (vm *VolumeManager) convertPodmanVolumeToResource(volume podman.VolumeInfo) *VolumeResource {
	resource := NewVolumeResource()
	resource.ObjectMeta.Name = volume.Name
	resource.SetLabels(volume.Labels)

	// Determine volume type based on driver and options
	if volume.Driver == "local" {
		// Check if it's a bind mount by looking at options
		if device, exists := volume.Options["device"]; exists && device != "" {
			// This is likely a hostPath volume (or legacy bind)
			resource.Spec.Type = VolumeTypeHostPath
			resource.Spec.HostPath = &HostPathVolumeSource{
				Path: device,
			}
		} else {
			// This is a named volume
			resource.Spec.Type = VolumeTypeVolume
			resource.Spec.Volume = &VolumeVolumeSource{
				Driver:  volume.Driver,
				Options: volume.Options,
			}
		}
	} else {
		// Non-local driver - treat as named volume
		resource.Spec.Type = VolumeTypeVolume
		resource.Spec.Volume = &VolumeVolumeSource{
			Driver:  volume.Driver,
			Options: volume.Options,
		}
	}

	// Note: EmptyDir volumes are not persisted in Podman, so they won't appear here
	// They are temporary directories managed by Cutepod

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
	// Legacy support - check for old-style hostPath field in options or driver
	var hostPath string

	// Try to get hostPath from legacy fields
	if device, exists := volume.Spec.Options["device"]; exists {
		hostPath = device
	} else {
		return fmt.Errorf("legacy bind mount requires device option or use hostPath volume type instead")
	}

	if hostPath == "" {
		return fmt.Errorf("hostPath is required for bind mount volumes")
	}

	// Ensure the host path exists
	if err := os.MkdirAll(hostPath, 0755); err != nil {
		return fmt.Errorf("unable to create host path %s: %w", hostPath, err)
	}

	// Validate that the path is absolute
	if !filepath.IsAbs(hostPath) {
		return fmt.Errorf("hostPath must be an absolute path, got: %s", hostPath)
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

// createHostPathVolume creates a hostPath volume by ensuring the host directory exists
func (vm *VolumeManager) createHostPathVolume(volume *VolumeResource) error {
	if volume.Spec.HostPath == nil {
		return fmt.Errorf("hostPath specification is required for hostPath volume")
	}

	hostPath := volume.Spec.HostPath.Path

	// Validate the path is absolute
	if !filepath.IsAbs(hostPath) {
		return fmt.Errorf("hostPath must be an absolute path, got: %s", hostPath)
	}

	// Handle different hostPath types
	pathType := HostPathDirectoryOrCreate
	if volume.Spec.HostPath.Type != nil {
		pathType = *volume.Spec.HostPath.Type
	}

	switch pathType {
	case HostPathDirectoryOrCreate:
		// Create directory if it doesn't exist
		if err := os.MkdirAll(hostPath, 0755); err != nil {
			return fmt.Errorf("unable to create host path %s: %w", hostPath, err)
		}
	case HostPathDirectory:
		// Verify directory exists
		if info, err := os.Stat(hostPath); err != nil {
			return fmt.Errorf("hostPath directory %s does not exist: %w", hostPath, err)
		} else if !info.IsDir() {
			return fmt.Errorf("hostPath %s exists but is not a directory", hostPath)
		}
	case HostPathFileOrCreate:
		// Create file if it doesn't exist
		if _, err := os.Stat(hostPath); os.IsNotExist(err) {
			// Create parent directory
			if err := os.MkdirAll(filepath.Dir(hostPath), 0755); err != nil {
				return fmt.Errorf("unable to create parent directory for %s: %w", hostPath, err)
			}
			// Create empty file
			if file, err := os.Create(hostPath); err != nil {
				return fmt.Errorf("unable to create host file %s: %w", hostPath, err)
			} else {
				file.Close()
			}
		}
	case HostPathFile:
		// Verify file exists
		if info, err := os.Stat(hostPath); err != nil {
			return fmt.Errorf("hostPath file %s does not exist: %w", hostPath, err)
		} else if info.IsDir() {
			return fmt.Errorf("hostPath %s exists but is a directory, expected file", hostPath)
		}
	case HostPathSocket, HostPathCharDevice, HostPathBlockDevice:
		// For special file types, just verify they exist
		if _, err := os.Stat(hostPath); err != nil {
			return fmt.Errorf("hostPath %s does not exist: %w", hostPath, err)
		}
	}

	// Apply security context if specified
	if volume.Spec.SecurityContext != nil {
		if err := vm.applyVolumeSecurityContext(hostPath, volume.Spec.SecurityContext); err != nil {
			return fmt.Errorf("failed to apply security context to %s: %w", hostPath, err)
		}
	}

	return nil
}

// createEmptyDirVolume creates an emptyDir volume by creating a temporary directory
func (vm *VolumeManager) createEmptyDirVolume(volume *VolumeResource) error {
	if volume.Spec.EmptyDir == nil {
		return fmt.Errorf("emptyDir specification is required for emptyDir volume")
	}

	// Create temporary directory
	// In a real implementation, you might want to use a specific base directory
	tempDir := filepath.Join("/tmp", "cutepod-emptydir", volume.GetName())

	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return fmt.Errorf("unable to create emptyDir %s: %w", tempDir, err)
	}

	// Apply security context if specified
	if volume.Spec.SecurityContext != nil {
		if err := vm.applyVolumeSecurityContext(tempDir, volume.Spec.SecurityContext); err != nil {
			return fmt.Errorf("failed to apply security context to emptyDir %s: %w", tempDir, err)
		}
	}

	// TODO: Handle sizeLimit and medium (Memory) - these would require additional Podman configuration

	return nil
}

// createNamedVolume creates a named Podman volume
func (vm *VolumeManager) createNamedVolume(ctx context.Context, podmanClient podman.PodmanClient, volume *VolumeResource) error {
	if volume.Spec.Volume == nil {
		return fmt.Errorf("volume specification is required for volume type")
	}

	// Build volume spec for named volumes
	spec := vm.buildNamedVolumeSpec(volume)

	// Create volume
	_, err := podmanClient.CreateVolume(ctx, spec)
	if err != nil {
		return fmt.Errorf("unable to create volume: %w", err)
	}

	return nil
}

// deleteEmptyDirVolume cleans up an emptyDir volume
func (vm *VolumeManager) deleteEmptyDirVolume(volume *VolumeResource) error {
	tempDir := filepath.Join("/tmp", "cutepod-emptydir", volume.GetName())

	if err := os.RemoveAll(tempDir); err != nil {
		return fmt.Errorf("unable to remove emptyDir %s: %w", tempDir, err)
	}

	return nil
}

// applyVolumeSecurityContext applies security context settings to a volume path
func (vm *VolumeManager) applyVolumeSecurityContext(path string, securityContext *VolumeSecurityContext) error {
	if securityContext.Owner != nil {
		uid := -1
		gid := -1

		if securityContext.Owner.User != nil {
			uid = int(*securityContext.Owner.User)
		}
		if securityContext.Owner.Group != nil {
			gid = int(*securityContext.Owner.Group)
		}

		if err := os.Chown(path, uid, gid); err != nil {
			return fmt.Errorf("failed to set ownership on %s: %w", path, err)
		}
	}

	// SELinux handling would be implemented here in a production system
	// For now, we just validate the configuration
	if securityContext.SELinuxOptions != nil {
		if securityContext.SELinuxOptions.Level != "" {
			switch securityContext.SELinuxOptions.Level {
			case "shared", "private":
				// Valid - would be applied during mount
			default:
				return fmt.Errorf("invalid SELinux level: %s", securityContext.SELinuxOptions.Level)
			}
		}
	}

	return nil
}

// buildNamedVolumeSpec builds a Podman volume spec for named volumes
func (vm *VolumeManager) buildNamedVolumeSpec(volume *VolumeResource) podman.VolumeSpec {
	spec := podman.VolumeSpec{
		Name:   volume.GetName(),
		Labels: volume.GetLabels(),
	}

	if volume.Spec.Volume != nil {
		spec.Driver = volume.Spec.Volume.Driver
		spec.Options = volume.Spec.Volume.Options
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

// Comparison helper methods for different volume types

func (vm *VolumeManager) compareHostPathSpecs(desired, actual *HostPathVolumeSource) bool {
	if desired == nil && actual == nil {
		return true
	}
	if desired == nil || actual == nil {
		return false
	}

	if desired.Path != actual.Path {
		return false
	}

	// Compare types
	desiredType := HostPathDirectoryOrCreate
	if desired.Type != nil {
		desiredType = *desired.Type
	}

	actualType := HostPathDirectoryOrCreate
	if actual.Type != nil {
		actualType = *actual.Type
	}

	return desiredType == actualType
}

func (vm *VolumeManager) compareEmptyDirSpecs(desired, actual *EmptyDirVolumeSource) bool {
	if desired == nil && actual == nil {
		return true
	}
	if desired == nil || actual == nil {
		return false
	}

	if desired.Medium != actual.Medium {
		return false
	}

	// Compare size limits
	desiredSize := ""
	if desired.SizeLimit != nil {
		desiredSize = *desired.SizeLimit
	}

	actualSize := ""
	if actual.SizeLimit != nil {
		actualSize = *actual.SizeLimit
	}

	return desiredSize == actualSize
}

func (vm *VolumeManager) compareVolumeSpecs(desired, actual *VolumeVolumeSource) bool {
	if desired == nil && actual == nil {
		return true
	}
	if desired == nil || actual == nil {
		return false
	}

	if desired.Driver != actual.Driver {
		return false
	}

	return vm.compareOptions(desired.Options, actual.Options)
}

func (vm *VolumeManager) compareSecurityContexts(desired, actual *VolumeSecurityContext) bool {
	if desired == nil && actual == nil {
		return true
	}
	if desired == nil || actual == nil {
		return false
	}

	// Compare SELinux options
	if !vm.compareSELinuxOptions(desired.SELinuxOptions, actual.SELinuxOptions) {
		return false
	}

	// Compare ownership
	if !vm.compareOwnership(desired.Owner, actual.Owner) {
		return false
	}

	return true
}

func (vm *VolumeManager) compareSELinuxOptions(desired, actual *SELinuxVolumeOptions) bool {
	if desired == nil && actual == nil {
		return true
	}
	if desired == nil || actual == nil {
		return false
	}

	return desired.Level == actual.Level
}

func (vm *VolumeManager) compareOwnership(desired, actual *VolumeOwnership) bool {
	if desired == nil && actual == nil {
		return true
	}
	if desired == nil || actual == nil {
		return false
	}

	// Compare user
	desiredUser := int64(-1)
	if desired.User != nil {
		desiredUser = *desired.User
	}

	actualUser := int64(-1)
	if actual.User != nil {
		actualUser = *actual.User
	}

	if desiredUser != actualUser {
		return false
	}

	// Compare group
	desiredGroup := int64(-1)
	if desired.Group != nil {
		desiredGroup = *desired.Group
	}

	actualGroup := int64(-1)
	if actual.Group != nil {
		actualGroup = *actual.Group
	}

	return desiredGroup == actualGroup
}
