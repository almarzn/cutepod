package resource

import (
	"context"
	"cutepod/internal/podman"
	"fmt"
	"os"
	"path/filepath"
)

// VolumeCreator defines the interface for creating different types of volumes
type VolumeCreator interface {
	// CreateVolume creates the volume and returns path information
	CreateVolume(ctx context.Context, client podman.PodmanClient, volume *VolumeResource) (*VolumePathInfo, error)

	// DeleteVolume cleans up the volume
	DeleteVolume(ctx context.Context, client podman.PodmanClient, volume *VolumeResource) error

	// SupportsType returns true if this creator supports the given volume type
	SupportsType(volumeType VolumeType) bool
}

// VolumeCreatorRegistry manages volume creators for different types
type VolumeCreatorRegistry struct {
	creators []VolumeCreator
}

// NewVolumeCreatorRegistry creates a new registry with default creators
func NewVolumeCreatorRegistry(pathManager *VolumePathManager, permissionMgr *VolumePermissionManager) *VolumeCreatorRegistry {
	return &VolumeCreatorRegistry{
		creators: []VolumeCreator{
			NewHostPathVolumeCreator(pathManager, permissionMgr),
			NewEmptyDirVolumeCreator(pathManager, permissionMgr),
			NewNamedVolumeCreator(),
		},
	}
}

// GetCreator returns the appropriate creator for the given volume type
func (r *VolumeCreatorRegistry) GetCreator(volumeType VolumeType) (VolumeCreator, error) {
	for _, creator := range r.creators {
		if creator.SupportsType(volumeType) {
			return creator, nil
		}
	}
	return nil, fmt.Errorf("no creator found for volume type: %s", volumeType)
}

// HostPathVolumeCreator handles hostPath volume creation
type HostPathVolumeCreator struct {
	pathManager   *VolumePathManager
	permissionMgr *VolumePermissionManager
}

// NewHostPathVolumeCreator creates a new hostPath volume creator
func NewHostPathVolumeCreator(pathManager *VolumePathManager, permissionMgr *VolumePermissionManager) *HostPathVolumeCreator {
	return &HostPathVolumeCreator{
		pathManager:   pathManager,
		permissionMgr: permissionMgr,
	}
}

// SupportsType returns true for hostPath volumes
func (c *HostPathVolumeCreator) SupportsType(volumeType VolumeType) bool {
	return volumeType == VolumeTypeHostPath
}

// CreateVolume creates a hostPath volume
func (c *HostPathVolumeCreator) CreateVolume(ctx context.Context, client podman.PodmanClient, volume *VolumeResource) (*VolumePathInfo, error) {
	if volume.Spec.HostPath == nil {
		return nil, fmt.Errorf("hostPath specification is required for hostPath volume type")
	}

	// Resolve the base volume path
	pathInfo, err := c.resolveHostPathBase(volume)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve hostPath volume path: %w", err)
	}

	// Ensure the path exists
	if err := c.pathManager.EnsureVolumePath(pathInfo, volume); err != nil {
		return nil, fmt.Errorf("failed to ensure hostPath volume path: %w", err)
	}

	// Apply permission management if available and security context is specified
	if c.permissionMgr != nil && volume.Spec.SecurityContext != nil {
		if err := c.permissionMgr.ManageHostDirectoryOwnership(pathInfo.SourcePath, volume); err != nil {
			return nil, fmt.Errorf("failed to manage host directory ownership: %w", err)
		}
	}

	return pathInfo, nil
}

// DeleteVolume deletes a hostPath volume (no-op since we don't delete host directories)
func (c *HostPathVolumeCreator) DeleteVolume(ctx context.Context, client podman.PodmanClient, volume *VolumeResource) error {
	// For hostPath volumes, we don't delete the host directory
	return nil
}

// resolveHostPathBase resolves the base path for a hostPath volume
func (c *HostPathVolumeCreator) resolveHostPathBase(volume *VolumeResource) (*VolumePathInfo, error) {
	hostPath := volume.Spec.HostPath.Path

	// Validate the host path
	if err := c.pathManager.hostPathValidator.validateHostPath(hostPath); err != nil {
		return nil, fmt.Errorf("invalid hostPath: %w", err)
	}

	// Determine path type
	pathType := HostPathDirectoryOrCreate
	if volume.Spec.HostPath.Type != nil {
		pathType = *volume.Spec.HostPath.Type
	}

	pathInfo := &VolumePathInfo{
		SourcePath: hostPath,
		PathType:   pathType,
	}

	// Check if path exists and determine if it's a file or directory
	if stat, err := os.Stat(hostPath); err == nil {
		pathInfo.IsFile = !stat.IsDir()
		pathInfo.RequiresCreation = false
	} else if os.IsNotExist(err) {
		// Path doesn't exist - determine what to create based on pathType
		pathInfo.RequiresCreation = true
		pathInfo.IsFile = c.shouldCreateAsFile(pathType)
	} else {
		return nil, fmt.Errorf("failed to stat path %s: %w", hostPath, err)
	}

	return pathInfo, nil
}

// shouldCreateAsFile determines if a path should be created as a file based on pathType
func (c *HostPathVolumeCreator) shouldCreateAsFile(pathType HostPathType) bool {
	switch pathType {
	case HostPathFile, HostPathFileOrCreate:
		return true
	case HostPathDirectory, HostPathDirectoryOrCreate:
		return false
	default:
		// For other types (Socket, CharDevice, BlockDevice), default to false
		return false
	}
}

// EmptyDirVolumeCreator handles emptyDir volume creation
type EmptyDirVolumeCreator struct {
	pathManager   *VolumePathManager
	permissionMgr *VolumePermissionManager
}

// NewEmptyDirVolumeCreator creates a new emptyDir volume creator
func NewEmptyDirVolumeCreator(pathManager *VolumePathManager, permissionMgr *VolumePermissionManager) *EmptyDirVolumeCreator {
	return &EmptyDirVolumeCreator{
		pathManager:   pathManager,
		permissionMgr: permissionMgr,
	}
}

// SupportsType returns true for emptyDir volumes
func (c *EmptyDirVolumeCreator) SupportsType(volumeType VolumeType) bool {
	return volumeType == VolumeTypeEmptyDir
}

// CreateVolume creates an emptyDir volume
func (c *EmptyDirVolumeCreator) CreateVolume(ctx context.Context, client podman.PodmanClient, volume *VolumeResource) (*VolumePathInfo, error) {
	if volume.Spec.EmptyDir == nil {
		return nil, fmt.Errorf("emptyDir specification is required for emptyDir volume type")
	}

	// Resolve the base volume path
	pathInfo, err := c.resolveEmptyDirBase(volume)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve emptyDir volume path: %w", err)
	}

	// Ensure the path exists
	if err := c.pathManager.EnsureVolumePath(pathInfo, volume); err != nil {
		return nil, fmt.Errorf("failed to ensure emptyDir volume path: %w", err)
	}

	// Handle sizeLimit if specified
	if volume.Spec.EmptyDir.SizeLimit != nil && *volume.Spec.EmptyDir.SizeLimit != "" {
		if err := c.applySizeLimit(pathInfo.SourcePath, *volume.Spec.EmptyDir.SizeLimit); err != nil {
			return nil, fmt.Errorf("failed to apply size limit to emptyDir volume: %w", err)
		}
	}

	// Handle memory medium (tmpfs) if specified
	if volume.Spec.EmptyDir.Medium == StorageMediumMemory {
		if err := c.setupMemoryVolume(pathInfo.SourcePath, volume.Spec.EmptyDir.SizeLimit); err != nil {
			return nil, fmt.Errorf("failed to setup memory-backed emptyDir volume: %w", err)
		}
	}

	// Apply permission management if available and security context is specified
	if c.permissionMgr != nil && volume.Spec.SecurityContext != nil {
		if err := c.permissionMgr.ManageHostDirectoryOwnership(pathInfo.SourcePath, volume); err != nil {
			return nil, fmt.Errorf("failed to manage host directory ownership: %w", err)
		}
	}

	return pathInfo, nil
}

// DeleteVolume deletes an emptyDir volume
func (c *EmptyDirVolumeCreator) DeleteVolume(ctx context.Context, client podman.PodmanClient, volume *VolumeResource) error {
	return c.pathManager.CleanupEmptyDirVolume(volume.GetName())
}

// resolveEmptyDirBase resolves the base path for an emptyDir volume
func (c *EmptyDirVolumeCreator) resolveEmptyDirBase(volume *VolumeResource) (*VolumePathInfo, error) {
	// Create a unique temporary directory for this emptyDir volume
	emptyDirPath := filepath.Join(c.pathManager.tempDirBase, volume.GetName())

	return &VolumePathInfo{
		SourcePath:       emptyDirPath,
		IsFile:           false, // emptyDir is always a directory
		RequiresCreation: true,  // Always needs creation
		PathType:         HostPathDirectoryOrCreate,
	}, nil
}

// applySizeLimit applies size constraints to an emptyDir volume
func (c *EmptyDirVolumeCreator) applySizeLimit(volumePath, sizeLimit string) error {
	// This is a placeholder implementation - in practice, this would:
	// 1. Parse the size limit (e.g., "1Gi" -> 1073741824 bytes)
	// 2. Apply filesystem quotas or use bind mounts with size constraints
	// 3. Set up monitoring to enforce the limit
	return nil
}

// setupMemoryVolume configures a memory-backed (tmpfs) volume
func (c *EmptyDirVolumeCreator) setupMemoryVolume(volumePath string, sizeLimit *string) error {
	// For memory volumes, we need to ensure the directory will be mounted as tmpfs
	// This is typically handled at the container runtime level
	// The actual tmpfs mounting will be handled by the container runtime
	return nil
}

// NamedVolumeCreator handles named Podman volume creation
type NamedVolumeCreator struct{}

// NewNamedVolumeCreator creates a new named volume creator
func NewNamedVolumeCreator() *NamedVolumeCreator {
	return &NamedVolumeCreator{}
}

// SupportsType returns true for named volumes
func (c *NamedVolumeCreator) SupportsType(volumeType VolumeType) bool {
	return volumeType == VolumeTypeVolume
}

// CreateVolume creates a named Podman volume
func (c *NamedVolumeCreator) CreateVolume(ctx context.Context, client podman.PodmanClient, volume *VolumeResource) (*VolumePathInfo, error) {
	if volume.Spec.Volume == nil {
		return nil, fmt.Errorf("volume specification is required for volume type")
	}

	connectedClient := podman.NewConnectedClient(client)
	defer connectedClient.Close()

	podmanClient, err := connectedClient.GetClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to podman: %w", err)
	}

	// Build volume spec for named volumes
	spec := c.buildNamedVolumeSpec(volume)

	// Create volume
	_, err = podmanClient.CreateVolume(ctx, spec)
	if err != nil {
		return nil, fmt.Errorf("unable to create volume: %w", err)
	}

	// Return path info for named volumes
	return &VolumePathInfo{
		SourcePath:       volume.GetName(),
		IsFile:           false,
		RequiresCreation: false,
		PathType:         HostPathDirectory,
	}, nil
}

// DeleteVolume deletes a named Podman volume
func (c *NamedVolumeCreator) DeleteVolume(ctx context.Context, client podman.PodmanClient, volume *VolumeResource) error {
	connectedClient := podman.NewConnectedClient(client)
	defer connectedClient.Close()

	podmanClient, err := connectedClient.GetClient(ctx)
	if err != nil {
		return fmt.Errorf("unable to connect to podman: %w", err)
	}

	return podmanClient.RemoveVolume(ctx, volume.GetName())
}

// buildNamedVolumeSpec builds a Podman volume spec for named volumes
func (c *NamedVolumeCreator) buildNamedVolumeSpec(volume *VolumeResource) podman.VolumeSpec {
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
