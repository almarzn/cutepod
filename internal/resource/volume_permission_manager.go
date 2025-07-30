package resource

import (
	"fmt"
	"os"
	"os/user"
	"strconv"
	"strings"
	"syscall"
)

// VolumePermissionManager handles SELinux, user namespaces, and ownership for volume mounts
type VolumePermissionManager struct {
	seLinuxEnabled bool
	rootlessMode   bool
	userNSMapping  *UserNamespaceMapping
}

// UserNamespaceMapping represents user namespace mapping configuration
type UserNamespaceMapping struct {
	UIDMapStart int64 // Starting UID in host namespace for rootless user
	GIDMapStart int64 // Starting GID in host namespace for rootless user
	MapSize     int64 // Size of the mapping range
}

// Note: UIDGIDMapping and VolumeMountOptions are defined in container.go

// VolumePermissionError represents permission-related errors with diagnostic information
type VolumePermissionError struct {
	VolumeName    string
	MountPath     string
	HostPath      string
	ErrorType     PermissionErrorType
	Suggestion    string
	OriginalError error
}

// PermissionErrorType represents different types of permission errors
type PermissionErrorType string

const (
	SELinuxDenied     PermissionErrorType = "selinux_denied"
	OwnershipMismatch PermissionErrorType = "ownership_mismatch"
	PathNotAccessible PermissionErrorType = "path_not_accessible"
	UserNSMappingFail PermissionErrorType = "user_namespace_mapping_failed"
)

// Error implements the error interface for VolumePermissionError
func (vpe *VolumePermissionError) Error() string {
	return fmt.Sprintf("volume permission error for %s: %s (suggestion: %s)",
		vpe.VolumeName, vpe.OriginalError.Error(), vpe.Suggestion)
}

// NewVolumePermissionManager creates a new VolumePermissionManager
func NewVolumePermissionManager() (*VolumePermissionManager, error) {
	vpm := &VolumePermissionManager{}

	// Detect SELinux status
	vpm.seLinuxEnabled = vpm.detectSELinuxEnabled()

	// Detect rootless mode
	vpm.rootlessMode = vpm.detectRootlessMode()

	// Initialize user namespace mapping for rootless mode
	if vpm.rootlessMode {
		mapping, err := vpm.detectUserNamespaceMapping()
		if err != nil {
			return nil, fmt.Errorf("failed to detect user namespace mapping: %w", err)
		}
		vpm.userNSMapping = mapping
	}

	return vpm, nil
}

// detectSELinuxEnabled checks if SELinux is enabled on the system
func (vpm *VolumePermissionManager) detectSELinuxEnabled() bool {
	// Check if SELinux is enabled by looking for /sys/fs/selinux
	if _, err := os.Stat("/sys/fs/selinux"); err == nil {
		// Check if SELinux is enforcing or permissive
		if data, err := os.ReadFile("/sys/fs/selinux/enforce"); err == nil {
			enforce := strings.TrimSpace(string(data))
			return enforce == "1" || enforce == "0" // Both enforcing and permissive count as enabled
		}
	}
	return false
}

// detectRootlessMode checks if we're running in rootless mode
func (vpm *VolumePermissionManager) detectRootlessMode() bool {
	// Check if we're running as root
	return os.Geteuid() != 0
}

// detectUserNamespaceMapping detects the user namespace mapping for rootless Podman
func (vpm *VolumePermissionManager) detectUserNamespaceMapping() (*UserNamespaceMapping, error) {
	if !vpm.rootlessMode {
		return nil, nil // No mapping needed for rootful mode
	}

	// Get current user
	currentUser, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}

	// Read /etc/subuid and /etc/subgid to get the mapping ranges
	uidMapStart, uidMapSize, err := vpm.readSubIDFile("/etc/subuid", currentUser.Username)
	if err != nil {
		return nil, fmt.Errorf("failed to read UID mapping: %w", err)
	}

	gidMapStart, gidMapSize, err := vpm.readSubIDFile("/etc/subgid", currentUser.Username)
	if err != nil {
		return nil, fmt.Errorf("failed to read GID mapping: %w", err)
	}

	// Use the smaller of the two ranges
	mapSize := uidMapSize
	if gidMapSize < mapSize {
		mapSize = gidMapSize
	}

	return &UserNamespaceMapping{
		UIDMapStart: uidMapStart,
		GIDMapStart: gidMapStart,
		MapSize:     mapSize,
	}, nil
}

// readSubIDFile reads /etc/subuid or /etc/subgid to get mapping information
func (vpm *VolumePermissionManager) readSubIDFile(filename, username string) (int64, int64, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to read %s: %w", filename, err)
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Split(line, ":")
		if len(parts) != 3 {
			continue
		}

		if parts[0] == username {
			start, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				continue
			}

			size, err := strconv.ParseInt(parts[2], 10, 64)
			if err != nil {
				continue
			}

			return start, size, nil
		}
	}

	return 0, 0, fmt.Errorf("no mapping found for user %s in %s", username, filename)
}

// DetermineSELinuxLabel determines the appropriate SELinux label for a volume mount
func (vpm *VolumePermissionManager) DetermineSELinuxLabel(volume *VolumeResource, mount *VolumeMount, sharedAccess bool) string {
	if !vpm.seLinuxEnabled {
		return ""
	}

	// Check explicit mount options first
	if mount.MountOptions != nil && mount.MountOptions.SELinuxLabel != "" {
		return mount.MountOptions.SELinuxLabel
	}

	// Check volume security context
	if volume.Spec.SecurityContext != nil && volume.Spec.SecurityContext.SELinuxOptions != nil {
		switch volume.Spec.SecurityContext.SELinuxOptions.Level {
		case "shared":
			return "z" // Shared label - multiple containers can access
		case "private":
			return "Z" // Private label - only this container can access
		}
	}

	// Default based on sharing requirements
	if sharedAccess {
		return "z"
	}
	return "Z"
}

// HandleUserNamespaceMapping handles user namespace mapping for rootless Podman
func (vpm *VolumePermissionManager) HandleUserNamespaceMapping(volume *VolumeResource, container *ContainerResource) (*UIDGIDMapping, *UIDGIDMapping, error) {
	if !vpm.rootlessMode || vpm.userNSMapping == nil {
		return nil, nil, nil // No mapping needed for rootful Podman
	}

	// Determine container user context
	containerUID := int64(0)
	containerGID := int64(0)

	if container.Spec.UID != nil {
		containerUID = *container.Spec.UID
	}
	if container.Spec.GID != nil {
		containerGID = *container.Spec.GID
	}

	// Map to host user namespace
	hostUID, hostGID, err := vpm.mapToHost(containerUID, containerGID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to map user namespace: %w", err)
	}

	uidMapping := &UIDGIDMapping{
		ContainerID: containerUID,
		HostID:      hostUID,
		Size:        1,
	}

	gidMapping := &UIDGIDMapping{
		ContainerID: containerGID,
		HostID:      hostGID,
		Size:        1,
	}

	return uidMapping, gidMapping, nil
}

// mapToHost maps container UID/GID to host UID/GID using user namespace mapping
func (vpm *VolumePermissionManager) mapToHost(containerUID, containerGID int64) (int64, int64, error) {
	if vpm.userNSMapping == nil {
		return containerUID, containerGID, nil
	}

	// Check if the container UID/GID is within the mapping range
	if containerUID >= vpm.userNSMapping.MapSize {
		return 0, 0, fmt.Errorf("container UID %d exceeds mapping range %d", containerUID, vpm.userNSMapping.MapSize)
	}

	if containerGID >= vpm.userNSMapping.MapSize {
		return 0, 0, fmt.Errorf("container GID %d exceeds mapping range %d", containerGID, vpm.userNSMapping.MapSize)
	}

	// Map to host namespace
	hostUID := vpm.userNSMapping.UIDMapStart + containerUID
	hostGID := vpm.userNSMapping.GIDMapStart + containerGID

	return hostUID, hostGID, nil
}

// ManageHostDirectoryOwnership manages ownership of host directories for volume mounts
func (vpm *VolumePermissionManager) ManageHostDirectoryOwnership(hostPath string, volume *VolumeResource) error {
	// Only handle ownership if security context specifies it
	if volume.Spec.SecurityContext == nil || volume.Spec.SecurityContext.Owner == nil {
		return nil
	}

	owner := volume.Spec.SecurityContext.Owner
	uid := -1
	gid := -1

	// Determine target UID/GID
	if owner.User != nil {
		if vpm.rootlessMode {
			// Map container UID to host UID
			hostUID, _, err := vpm.mapToHost(*owner.User, 0)
			if err != nil {
				return fmt.Errorf("failed to map UID for ownership: %w", err)
			}
			uid = int(hostUID)
		} else {
			uid = int(*owner.User)
		}
	}

	if owner.Group != nil {
		if vpm.rootlessMode {
			// Map container GID to host GID
			_, hostGID, err := vpm.mapToHost(0, *owner.Group)
			if err != nil {
				return fmt.Errorf("failed to map GID for ownership: %w", err)
			}
			gid = int(hostGID)
		} else {
			gid = int(*owner.Group)
		}
	}

	// Apply ownership if specified
	if uid != -1 || gid != -1 {
		if err := os.Chown(hostPath, uid, gid); err != nil {
			return fmt.Errorf("failed to set ownership on %s to %d:%d: %w", hostPath, uid, gid, err)
		}
	}

	return nil
}

// BuildPodmanMountOptions builds Podman mount options for a volume mount
func (vpm *VolumePermissionManager) BuildPodmanMountOptions(volume *VolumeResource, mount *VolumeMount, sharedAccess bool) ([]string, error) {
	var options []string

	// Base mount type
	switch volume.Spec.Type {
	case VolumeTypeHostPath:
		options = append(options, "bind")
	}

	// Read-only flag
	if mount.ReadOnly {
		options = append(options, "ro")
	}

	// SELinux labels
	seLinuxLabel := vpm.DetermineSELinuxLabel(volume, mount, sharedAccess)
	if seLinuxLabel != "" {
		options = append(options, seLinuxLabel)
	}

	// Note: UID/GID mapping for rootless is handled by Podman automatically
	// The actual chown happens in ManageHostDirectoryOwnership() before mounting

	return options, nil
}

// DiagnosePermissionError analyzes permission errors and provides diagnostic information
func (vpm *VolumePermissionManager) DiagnosePermissionError(err error, volume *VolumeResource, mount *VolumeMount, hostPath string) *VolumePermissionError {
	errStr := err.Error()

	switch {
	case strings.Contains(errStr, "permission denied") && vpm.seLinuxEnabled:
		return &VolumePermissionError{
			VolumeName:    volume.GetName(),
			MountPath:     mount.MountPath,
			HostPath:      hostPath,
			ErrorType:     SELinuxDenied,
			Suggestion:    "Try adding seLinuxLabel: 'z' to mountOptions for shared access, or 'Z' for private access",
			OriginalError: err,
		}
	case strings.Contains(errStr, "operation not permitted"):
		return &VolumePermissionError{
			VolumeName:    volume.GetName(),
			MountPath:     mount.MountPath,
			HostPath:      hostPath,
			ErrorType:     OwnershipMismatch,
			Suggestion:    "Check that the host path ownership matches the container's UID/GID or set appropriate securityContext",
			OriginalError: err,
		}
	case vpm.rootlessMode && strings.Contains(errStr, "user namespace"):
		return &VolumePermissionError{
			VolumeName:    volume.GetName(),
			MountPath:     mount.MountPath,
			HostPath:      hostPath,
			ErrorType:     UserNSMappingFail,
			Suggestion:    "Verify that the host path is accessible within the rootless user namespace",
			OriginalError: err,
		}
	default:
		return &VolumePermissionError{
			VolumeName:    volume.GetName(),
			MountPath:     mount.MountPath,
			HostPath:      hostPath,
			ErrorType:     PathNotAccessible,
			Suggestion:    "Ensure the host path exists and is accessible to the Podman process",
			OriginalError: err,
		}
	}
}

// IsRootlessMode returns whether the manager is operating in rootless mode
func (vpm *VolumePermissionManager) IsRootlessMode() bool {
	return vpm.rootlessMode
}

// IsSELinuxEnabled returns whether SELinux is enabled
func (vpm *VolumePermissionManager) IsSELinuxEnabled() bool {
	return vpm.seLinuxEnabled
}

// GetUserNamespaceMapping returns the user namespace mapping configuration
func (vpm *VolumePermissionManager) GetUserNamespaceMapping() *UserNamespaceMapping {
	return vpm.userNSMapping
}

// ValidateVolumePermissions validates that a volume can be mounted with the specified permissions
func (vpm *VolumePermissionManager) ValidateVolumePermissions(volume *VolumeResource, mount *VolumeMount, hostPath string) error {
	// Check if the host path exists and is accessible
	if _, err := os.Stat(hostPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("host path does not exist: %s", hostPath)
		}
		return fmt.Errorf("host path is not accessible: %w", err)
	}

	// Check if we can read the path
	if _, err := os.Open(hostPath); err != nil {
		return fmt.Errorf("cannot read host path %s: %w", hostPath, err)
	}

	// For rootless mode, check if the path is within accessible range
	if vpm.rootlessMode {
		if err := vpm.validateRootlessAccess(hostPath); err != nil {
			return fmt.Errorf("rootless access validation failed for %s: %w", hostPath, err)
		}
	}

	return nil
}

// validateRootlessAccess validates that a path is accessible in rootless mode
func (vpm *VolumePermissionManager) validateRootlessAccess(hostPath string) error {
	// Get file info
	fileInfo, err := os.Stat(hostPath)
	if err != nil {
		return err
	}

	// Get file system info
	stat, ok := fileInfo.Sys().(*syscall.Stat_t)
	if !ok {
		return fmt.Errorf("unable to get file system info")
	}

	// Check if the file is owned by the current user or accessible
	currentUID := uint32(os.Geteuid())
	currentGID := uint32(os.Getegid())

	// Check owner permissions
	if stat.Uid == currentUID {
		return nil // Owner can access
	}

	// Check group permissions
	if stat.Gid == currentGID && fileInfo.Mode()&0040 != 0 {
		return nil // Group has read access
	}

	// Check other permissions
	if fileInfo.Mode()&0004 != 0 {
		return nil // Others have read access
	}

	return fmt.Errorf("path %s is not accessible to current user in rootless mode", hostPath)
}
