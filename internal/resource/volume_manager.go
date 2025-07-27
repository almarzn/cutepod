package resource

import (
	"context"
	"cutepod/internal/labels"
	"cutepod/internal/podman"
	"fmt"
)

// VolumeManager implements ResourceManager for volume resources
type VolumeManager struct {
	client        podman.PodmanClient
	pathManager   *VolumePathManager
	permissionMgr *VolumePermissionManager
}

// NewVolumeManager creates a new VolumeManager
func NewVolumeManager(client podman.PodmanClient) *VolumeManager {
	permissionMgr, err := NewVolumePermissionManager()
	if err != nil {
		// Log error but continue with nil permission manager
		// This allows the system to work even if permission detection fails
		permissionMgr = nil
	}

	return &VolumeManager{
		client:        client,
		pathManager:   NewVolumePathManager(""),
		permissionMgr: permissionMgr,
	}
}

// NewVolumeManagerWithPathManager creates a new VolumeManager with a custom VolumePathManager
func NewVolumeManagerWithPathManager(client podman.PodmanClient, pathManager *VolumePathManager) *VolumeManager {
	permissionMgr, err := NewVolumePermissionManager()
	if err != nil {
		// Log error but continue with nil permission manager
		permissionMgr = nil
	}

	return &VolumeManager{
		client:        client,
		pathManager:   pathManager,
		permissionMgr: permissionMgr,
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
	// Create a dummy mount to resolve the base path (no subPath)
	dummyMount := &VolumeMount{
		Name:      volume.GetName(),
		MountPath: "/dummy", // Not used for path resolution
	}

	// Resolve the base path using the path manager
	pathInfo, err := vm.pathManager.ResolveVolumePath(volume, dummyMount)
	if err != nil {
		return fmt.Errorf("failed to resolve legacy bind mount path: %w", err)
	}

	// Ensure the path exists
	if err := vm.pathManager.EnsureVolumePath(pathInfo, volume); err != nil {
		return fmt.Errorf("failed to ensure legacy bind mount path: %w", err)
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
	// Create a dummy mount to resolve the base path (no subPath)
	dummyMount := &VolumeMount{
		Name:      volume.GetName(),
		MountPath: "/dummy", // Not used for path resolution
	}

	// Resolve the base path
	pathInfo, err := vm.pathManager.ResolveVolumePath(volume, dummyMount)
	if err != nil {
		return fmt.Errorf("failed to resolve hostPath volume path: %w", err)
	}

	// Ensure the path exists
	if err := vm.pathManager.EnsureVolumePath(pathInfo, volume); err != nil {
		return fmt.Errorf("failed to ensure hostPath volume path: %w", err)
	}

	// Apply permission management if available
	if vm.permissionMgr != nil {
		// Create a dummy container for permission management (we don't have container context here)
		dummyContainer := &ContainerResource{}

		if err := vm.permissionMgr.ManageHostDirectoryOwnership(pathInfo.SourcePath, volume, dummyContainer); err != nil {
			return fmt.Errorf("failed to manage host directory ownership: %w", err)
		}

		// Validate permissions
		if err := vm.permissionMgr.ValidateVolumePermissions(volume, dummyMount, pathInfo.SourcePath); err != nil {
			return fmt.Errorf("volume permission validation failed: %w", err)
		}
	}

	return nil
}

// createEmptyDirVolume creates an emptyDir volume by creating a temporary directory
func (vm *VolumeManager) createEmptyDirVolume(volume *VolumeResource) error {
	// Create a dummy mount to resolve the base path (no subPath)
	dummyMount := &VolumeMount{
		Name:      volume.GetName(),
		MountPath: "/dummy", // Not used for path resolution
	}

	// Resolve the base path
	pathInfo, err := vm.pathManager.ResolveVolumePath(volume, dummyMount)
	if err != nil {
		return fmt.Errorf("failed to resolve emptyDir volume path: %w", err)
	}

	// Ensure the path exists
	if err := vm.pathManager.EnsureVolumePath(pathInfo, volume); err != nil {
		return fmt.Errorf("failed to ensure emptyDir volume path: %w", err)
	}

	// Apply permission management if available
	if vm.permissionMgr != nil {
		// Create a dummy container for permission management (we don't have container context here)
		dummyContainer := &ContainerResource{}

		if err := vm.permissionMgr.ManageHostDirectoryOwnership(pathInfo.SourcePath, volume, dummyContainer); err != nil {
			return fmt.Errorf("failed to manage host directory ownership: %w", err)
		}

		// Validate permissions
		if err := vm.permissionMgr.ValidateVolumePermissions(volume, dummyMount, pathInfo.SourcePath); err != nil {
			return fmt.Errorf("volume permission validation failed: %w", err)
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
	return vm.pathManager.CleanupEmptyDirVolume(volume.GetName())
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

// GetVolumePathManager returns the VolumePathManager instance for external use
func (vm *VolumeManager) GetVolumePathManager() *VolumePathManager {
	return vm.pathManager
}

// GetVolumePermissionManager returns the VolumePermissionManager instance for external use
func (vm *VolumeManager) GetVolumePermissionManager() *VolumePermissionManager {
	return vm.permissionMgr
}
