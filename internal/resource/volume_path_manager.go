package resource

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// VolumePathManager handles directory creation, path resolution, and security validation for volumes
type VolumePathManager struct {
	tempDirBase       string
	hostPathValidator *HostPathValidator
}

// HostPathValidator provides security validation for host paths
type HostPathValidator struct {
	allowedPrefixes []string // Optional: restrict hostPath to specific prefixes
}

// VolumePathInfo contains resolved path information for a volume mount
type VolumePathInfo struct {
	SourcePath       string       // Resolved source path on the host
	IsFile           bool         // True if the source is a file, false if directory
	RequiresCreation bool         // True if the path needs to be created
	PathType         HostPathType // Type of path (for validation)
}

// NewVolumePathManager creates a new VolumePathManager
func NewVolumePathManager(tempDirBase string) *VolumePathManager {
	if tempDirBase == "" {
		tempDirBase = "/tmp/cutepod-volumes"
	}

	return &VolumePathManager{
		tempDirBase: tempDirBase,
		hostPathValidator: &HostPathValidator{
			allowedPrefixes: []string{}, // Empty means allow all paths
		},
	}
}

// NewVolumePathManagerWithRestrictions creates a VolumePathManager with path restrictions
func NewVolumePathManagerWithRestrictions(tempDirBase string, allowedPrefixes []string) *VolumePathManager {
	if tempDirBase == "" {
		tempDirBase = "/tmp/cutepod-volumes"
	}

	return &VolumePathManager{
		tempDirBase: tempDirBase,
		hostPathValidator: &HostPathValidator{
			allowedPrefixes: allowedPrefixes,
		},
	}
}

// ResolveVolumePath resolves the source path for a volume mount, handling subPath resolution
func (vpm *VolumePathManager) ResolveVolumePath(volume *VolumeResource, mount *VolumeMount) (*VolumePathInfo, error) {
	if volume == nil {
		return nil, fmt.Errorf("volume resource cannot be nil")
	}
	if mount == nil {
		return nil, fmt.Errorf("volume mount cannot be nil")
	}

	// Validate subPath for security
	if err := vpm.validateSubPath(mount.SubPath); err != nil {
		return nil, fmt.Errorf("invalid subPath '%s': %w", mount.SubPath, err)
	}

	switch volume.Spec.Type {
	case VolumeTypeHostPath:
		return vpm.resolveHostPathVolume(volume, mount)
	case VolumeTypeEmptyDir:
		return vpm.resolveEmptyDirVolume(volume, mount)
	case VolumeTypeVolume:
		// Named volumes don't need path resolution - handled by Podman
		return &VolumePathInfo{
			SourcePath:       volume.GetName(),
			IsFile:           false,
			RequiresCreation: false,
			PathType:         HostPathDirectory,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported volume type: %s", volume.Spec.Type)
	}
}

// EnsureVolumePath creates the necessary directories/files for a volume mount
func (vpm *VolumePathManager) EnsureVolumePath(pathInfo *VolumePathInfo, volume *VolumeResource) error {
	if !pathInfo.RequiresCreation {
		return nil // Path already exists or doesn't need creation
	}

	if pathInfo.IsFile {
		return vpm.ensureFilePath(pathInfo.SourcePath, volume)
	}
	return vpm.ensureDirectoryPath(pathInfo.SourcePath, volume)
}

// CleanupEmptyDirVolume removes an emptyDir volume's temporary directory
func (vpm *VolumePathManager) CleanupEmptyDirVolume(volumeName string) error {
	tempDir := vpm.getEmptyDirPath(volumeName)

	if err := os.RemoveAll(tempDir); err != nil {
		return fmt.Errorf("failed to cleanup emptyDir volume %s at %s: %w", volumeName, tempDir, err)
	}

	return nil
}

// validateSubPath validates subPath for security issues
func (vpm *VolumePathManager) validateSubPath(subPath string) error {
	if subPath == "" {
		return nil // Empty subPath is valid
	}

	// Prevent path traversal attacks
	if strings.Contains(subPath, "..") {
		return fmt.Errorf("subPath cannot contain '..' (path traversal not allowed)")
	}

	// SubPath must be relative
	if strings.HasPrefix(subPath, "/") {
		return fmt.Errorf("subPath must be relative (cannot start with '/')")
	}

	// Prevent consecutive slashes
	if strings.Contains(subPath, "//") {
		return fmt.Errorf("subPath cannot contain consecutive slashes")
	}

	// Prevent empty path components
	parts := strings.Split(subPath, "/")
	for _, part := range parts {
		if strings.TrimSpace(part) == "" {
			return fmt.Errorf("subPath cannot contain empty path components")
		}
	}

	// Additional security checks for dangerous characters
	dangerousChars := []string{"\x00", "\n", "\r", "\t"}
	for _, char := range dangerousChars {
		if strings.Contains(subPath, char) {
			return fmt.Errorf("subPath contains invalid character")
		}
	}

	return nil
}

// resolveHostPathVolume resolves paths for hostPath volumes
func (vpm *VolumePathManager) resolveHostPathVolume(volume *VolumeResource, mount *VolumeMount) (*VolumePathInfo, error) {
	if volume.Spec.HostPath == nil {
		return nil, fmt.Errorf("hostPath specification is required for hostPath volume")
	}

	basePath := volume.Spec.HostPath.Path

	// Validate host path security
	if err := vpm.hostPathValidator.validateHostPath(basePath); err != nil {
		return nil, fmt.Errorf("hostPath validation failed: %w", err)
	}

	// Resolve final path with subPath
	finalPath := basePath
	if mount.SubPath != "" {
		finalPath = filepath.Join(basePath, mount.SubPath)
	}

	// Clean the path to resolve any . or .. components that might have been introduced
	finalPath = filepath.Clean(finalPath)

	// Ensure the final path is still within the base path (additional security check)
	if !strings.HasPrefix(finalPath, basePath) {
		return nil, fmt.Errorf("resolved path %s is outside base path %s", finalPath, basePath)
	}

	// Determine path type and requirements
	pathType := HostPathDirectoryOrCreate
	if volume.Spec.HostPath.Type != nil {
		pathType = *volume.Spec.HostPath.Type
	}

	pathInfo := &VolumePathInfo{
		SourcePath: finalPath,
		PathType:   pathType,
	}

	// Check if path exists and determine if it's a file or directory
	if stat, err := os.Stat(finalPath); err == nil {
		pathInfo.IsFile = !stat.IsDir()
		pathInfo.RequiresCreation = false
	} else if os.IsNotExist(err) {
		// Path doesn't exist - determine what to create based on pathType and subPath
		pathInfo.RequiresCreation = true
		pathInfo.IsFile = vpm.shouldCreateAsFile(pathType, mount.SubPath)
	} else {
		return nil, fmt.Errorf("failed to stat path %s: %w", finalPath, err)
	}

	// Validate path type requirements
	if err := vpm.validatePathTypeRequirements(pathInfo, pathType); err != nil {
		return nil, err
	}

	return pathInfo, nil
}

// resolveEmptyDirVolume resolves paths for emptyDir volumes
func (vpm *VolumePathManager) resolveEmptyDirVolume(volume *VolumeResource, mount *VolumeMount) (*VolumePathInfo, error) {
	if volume.Spec.EmptyDir == nil {
		return nil, fmt.Errorf("emptyDir specification is required for emptyDir volume")
	}

	// Get base temporary directory for this volume
	basePath := vpm.getEmptyDirPath(volume.GetName())

	// Resolve final path with subPath
	finalPath := basePath
	if mount.SubPath != "" {
		finalPath = filepath.Join(basePath, mount.SubPath)
	}

	pathInfo := &VolumePathInfo{
		SourcePath:       finalPath,
		IsFile:           false, // EmptyDir volumes are always directories
		RequiresCreation: true,  // EmptyDir volumes always need creation
		PathType:         HostPathDirectoryOrCreate,
	}

	return pathInfo, nil
}

// shouldCreateAsFile determines if a path should be created as a file based on pathType and subPath
func (vpm *VolumePathManager) shouldCreateAsFile(pathType HostPathType, subPath string) bool {
	switch pathType {
	case HostPathFile, HostPathFileOrCreate:
		return true
	case HostPathDirectory:
		return false
	case HostPathDirectoryOrCreate:
		// For DirectoryOrCreate, infer from subPath extension
		if subPath != "" {
			ext := filepath.Ext(subPath)
			// If subPath has an extension, assume it's a file
			return ext != ""
		}
		return false
	default:
		// For other types, infer from subPath extension
		if subPath != "" {
			ext := filepath.Ext(subPath)
			// If subPath has an extension, assume it's a file
			return ext != ""
		}
		return false
	}
}

// validatePathTypeRequirements validates that the path meets the requirements of its type
func (vpm *VolumePathManager) validatePathTypeRequirements(pathInfo *VolumePathInfo, pathType HostPathType) error {
	if pathInfo.RequiresCreation {
		return nil // Will be validated after creation
	}

	// Path exists - validate it matches the expected type
	switch pathType {
	case HostPathFile:
		if !pathInfo.IsFile {
			return fmt.Errorf("path %s exists but is not a file (required by hostPath type 'File')", pathInfo.SourcePath)
		}
	case HostPathDirectory:
		if pathInfo.IsFile {
			return fmt.Errorf("path %s exists but is a file, not a directory (required by hostPath type 'Directory')", pathInfo.SourcePath)
		}
	case HostPathFileOrCreate, HostPathDirectoryOrCreate:
		// These types accept either existing files/directories or will create them
		break
	case HostPathSocket, HostPathCharDevice, HostPathBlockDevice:
		// For special file types, we just verify they exist (already done in resolveHostPathVolume)
		break
	}

	return nil
}

// getEmptyDirPath returns the temporary directory path for an emptyDir volume
func (vpm *VolumePathManager) getEmptyDirPath(volumeName string) string {
	return filepath.Join(vpm.tempDirBase, "emptydir", volumeName)
}

// ensureDirectoryPath creates a directory path with proper permissions
func (vpm *VolumePathManager) ensureDirectoryPath(path string, volume *VolumeResource) error {
	// Create directory with default permissions
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", path, err)
	}

	// Apply security context if specified
	if volume.Spec.SecurityContext != nil {
		if err := vpm.applySecurityContext(path, volume.Spec.SecurityContext); err != nil {
			// In rootless mode, ownership changes may fail - log warning but continue
			fmt.Printf("Warning: failed to apply security context to directory %s: %v (continuing anyway)\n", path, err)
		}
	}

	return nil
}

// ensureFilePath creates a file path with proper permissions
func (vpm *VolumePathManager) ensureFilePath(path string, volume *VolumeResource) error {
	// Create parent directory first
	parentDir := filepath.Dir(path)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("failed to create parent directory %s: %w", parentDir, err)
	}

	// Create empty file if it doesn't exist
	if _, err := os.Stat(path); os.IsNotExist(err) {
		file, err := os.Create(path)
		if err != nil {
			return fmt.Errorf("failed to create file %s: %w", path, err)
		}
		file.Close()
	}

	// Apply security context if specified
	if volume.Spec.SecurityContext != nil {
		if err := vpm.applySecurityContext(path, volume.Spec.SecurityContext); err != nil {
			// In rootless mode, ownership changes may fail - log warning but continue
			fmt.Printf("Warning: failed to apply security context to file %s: %v (continuing anyway)\n", path, err)
		}
	}

	return nil
}

// applySecurityContext applies ownership and other security settings to a path
func (vpm *VolumePathManager) applySecurityContext(path string, securityContext *VolumeSecurityContext) error {
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

// validateHostPath validates a host path for security
func (hpv *HostPathValidator) validateHostPath(hostPath string) error {
	// Validate path is absolute
	if !filepath.IsAbs(hostPath) {
		return fmt.Errorf("hostPath must be an absolute path, got: %s", hostPath)
	}

	// Prevent path traversal in the base path itself
	if strings.Contains(hostPath, "..") {
		return fmt.Errorf("hostPath cannot contain '..' for security reasons")
	}

	// Clean the path and ensure it hasn't changed (additional security check)
	cleanPath := filepath.Clean(hostPath)
	if cleanPath != hostPath {
		return fmt.Errorf("hostPath contains invalid path components: %s", hostPath)
	}

	// Check against allowed prefixes if configured
	if len(hpv.allowedPrefixes) > 0 {
		allowed := false
		for _, prefix := range hpv.allowedPrefixes {
			if strings.HasPrefix(hostPath, prefix) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("hostPath %s is not within allowed prefixes: %v", hostPath, hpv.allowedPrefixes)
		}
	}

	return nil
}
