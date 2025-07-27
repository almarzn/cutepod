# Design Document

## Overview

This design enhances Cutepod's volume management system by implementing Kubernetes-style volume types with subPath support and addressing Podman-specific mount permission challenges. The design separates volume definitions from container specifications, enabling better reusability and more flexible mount configurations while handling the complexities of rootless Podman operation, SELinux, and user namespace mapping.

## Architecture

### Volume Type System

The enhanced volume system introduces a hierarchical type system:

```go
type CuteVolumeSpec struct {
    Type        VolumeType                `json:"type"`
    HostPath    *HostPathVolumeSource    `json:"hostPath,omitempty"`
    EmptyDir    *EmptyDirVolumeSource    `json:"emptyDir,omitempty"`
    Volume      *VolumeVolumeSource      `json:"volume,omitempty"`
    
    // Permission and security settings
    SecurityContext *VolumeSecurityContext `json:"securityContext,omitempty"`
}

type VolumeSecurityContext struct {
    SELinuxOptions *SELinuxVolumeOptions `json:"seLinuxOptions,omitempty"`
    Owner          *VolumeOwnership      `json:"owner,omitempty"`
}

type VolumeOwnership struct {
    User  *int64 `json:"user,omitempty"`   // UID for host directory ownership
    Group *int64 `json:"group,omitempty"`  // GID for host directory ownership
}

type SELinuxVolumeOptions struct {
    Level string `json:"level,omitempty"` // "shared" (z flag) or "private" (Z flag)
}
```

### Enhanced VolumeMount Structure

```go
type VolumeMount struct {
    Name          string                    `json:"name"`
    MountPath     string                    `json:"mountPath"`
    SubPath       string                    `json:"subPath,omitempty"`
    ReadOnly      bool                      `json:"readOnly,omitempty"`
    
    // Podman-specific mount options
    MountOptions  *VolumeMountOptions      `json:"mountOptions,omitempty"`
}

type VolumeMountOptions struct {
    SELinuxLabel  string   `json:"seLinuxLabel,omitempty"`  // "z", "Z", or custom
    UIDMapping    *UIDGIDMapping `json:"uidMapping,omitempty"`
    GIDMapping    *UIDGIDMapping `json:"gidMapping,omitempty"`
}

type UIDGIDMapping struct {
    ContainerID int64 `json:"containerID"`
    HostID      int64 `json:"hostID"`
    Size        int64 `json:"size"`
}
```

## Podman Integration Details

### How VolumeSecurityContext Maps to Podman Commands

The `VolumeSecurityContext` affects both **host-side preparation** and **Podman mount options**:

#### Host-Side Effects (before `podman run`):
```go
// VolumeSecurityContext affects host directory preparation
spec:
  securityContext:
    runAsUser: 1000      # chown 1000 /host/path (before mounting)
    runAsGroup: 1000     # chgrp 1000 /host/path (before mounting)
    seLinuxOptions:
      level: shared      # Determines mount option (z vs Z)
```

#### Podman Mount Options (during `podman run`):
```bash
# This VolumeSecurityContext:
securityContext:
  seLinuxOptions:
    level: shared

# Becomes this Podman command:
podman run -v /host/path:/container/path:bind,z nginx:latest

# While this:
securityContext:
  seLinuxOptions:
    level: private

# Becomes:
podman run -v /host/path:/container/path:bind,Z nginx:latest
```

#### Complete Translation Example:
```yaml
# Cutepod Volume Definition
apiVersion: cutepod.io/v1
kind: CuteVolume
metadata:
  name: web-content
spec:
  type: hostPath
  hostPath:
    path: /home/user/website
  securityContext:
    owner:
      user: 1000
      group: 1000
    seLinuxOptions:
      level: shared

# Container Mount
volumes:
  - name: web-content
    mountPath: /usr/share/nginx/html
    subPath: dist
    readOnly: true
```

**Results in:**
1. **Host preparation**: `chown 1000:1000 /home/user/website/dist`
2. **Podman command**: `podman run -v /home/user/website/dist:/usr/share/nginx/html:bind,z,ro nginx:latest`

### Rootless vs Rootful Behavior

#### Rootful Podman:
- `owner.user: 1000` → Direct `chown 1000` on host
- Container sees files as owned by UID 1000
- SELinux labels applied directly

#### Rootless Podman:
- `owner.user: 1000` → `chown` to mapped host UID (e.g., 100000 + 1000)
- Podman automatically maps back to UID 1000 inside container
- SELinux labels still work the same way

## Components and Interfaces

### Volume Manager Enhancement

The existing VolumeManager will be enhanced to handle the new volume types:

```go
type EnhancedVolumeManager struct {
    podmanClient    podman.PodmanClient
    pathManager     *VolumePathManager
    permissionMgr   *VolumePermissionManager
}

type VolumePathManager struct {
    tempDirBase     string
    hostPathValidator *HostPathValidator
}

type VolumePermissionManager struct {
    seLinuxEnabled  bool
    rootlessMode    bool
    userNSMapping   *UserNamespaceMapping
}
```

### Permission Handling Strategy

#### 1. SELinux Label Management
```go
func (vpm *VolumePermissionManager) determineSELinuxLabel(volume *VolumeResource, mount *VolumeMount, sharedAccess bool) string {
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
            return "z"  // Shared label - multiple containers can access
        case "private":
            return "Z"  // Private label - only this container can access
        }
    }
    
    // Default based on sharing requirements
    if sharedAccess {
        return "z"
    }
    return "Z"
}
```

#### 2. User Namespace Mapping
```go
func (vpm *VolumePermissionManager) handleUserNamespaceMapping(volume *VolumeResource, container *ContainerResource) (*UIDGIDMapping, error) {
    if !vpm.rootlessMode {
        return nil, nil // No mapping needed for rootful Podman
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
    hostUID, hostGID, err := vpm.userNSMapping.MapToHost(containerUID, containerGID)
    if err != nil {
        return nil, fmt.Errorf("failed to map user namespace: %w", err)
    }
    
    return &UIDGIDMapping{
        ContainerID: containerUID,
        HostID:      hostUID,
        Size:        1,
    }, nil
}
```

#### 3. Podman Mount Option Translation

The `VolumeSecurityContext` translates directly to Podman mount options:

```go
func (vpm *VolumePermissionManager) buildPodmanMountOptions(volume *VolumeResource, mount *VolumeMount) ([]string, error) {
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
    seLinuxLabel := vpm.determineSELinuxLabel(volume, mount, false)
    if seLinuxLabel != "" {
        options = append(options, seLinuxLabel)
    }
    
    // UID/GID mapping for rootless
    if vpm.rootlessMode {
        if volume.Spec.SecurityContext != nil && volume.Spec.SecurityContext.Owner != nil {
            // This affects the host directory ownership, not container runtime
            // Podman handles user namespace mapping automatically
            // The actual chown happens in ensureHostPath() before mounting
        }
    }
    
    return options, nil
}

// Example Podman command generation:
// podman run -v /host/path:/container/path:bind,z,ro nginx:latest
func (cm *ContainerManager) buildMountSpec(volume *VolumeResource, mount *VolumeMount) (specs.Mount, error) {
    options, err := cm.permissionMgr.buildPodmanMountOptions(volume, mount)
    if err != nil {
        return specs.Mount{}, err
    }
    
    sourcePath := volume.Spec.HostPath.Path
    if mount.SubPath != "" {
        sourcePath = filepath.Join(sourcePath, mount.SubPath)
    }
    
    return specs.Mount{
        Source:      sourcePath,
        Destination: mount.MountPath,
        Type:        "bind",
        Options:     options, // ["bind", "z", "ro"] becomes "bind,z,ro"
    }, nil
}
```

#### 4. Directory Creation with Proper Permissions
```go
func (vpm *VolumePathManager) ensureHostPath(hostPath string, securityContext *VolumeSecurityContext) error {
    // Create directory if it doesn't exist
    if err := os.MkdirAll(hostPath, 0755); err != nil {
        return fmt.Errorf("failed to create host path %s: %w", hostPath, err)
    }
    
    // Apply ownership if specified - this is done on the HOST side before mounting
    if securityContext != nil && securityContext.Owner != nil {
        uid := -1
        gid := -1
        
        if securityContext.Owner.User != nil {
            uid = int(*securityContext.Owner.User)
        }
        if securityContext.Owner.Group != nil {
            gid = int(*securityContext.Owner.Group)
        }
        
        // This changes the actual host directory ownership
        // Podman will then map this through user namespaces if running rootless
        if err := os.Chown(hostPath, uid, gid); err != nil {
            return fmt.Errorf("failed to set ownership on %s: %w", hostPath, err)
        }
    }
    
    return nil
}
```

## Data Models

### Volume Resource Definition
```yaml
apiVersion: cutepod.io/v1
kind: CuteVolume
metadata:
  name: project-source
spec:
  type: hostPath
  hostPath:
    path: /home/user/myproject
    type: Directory
  securityContext:
    owner:
      user: 1000
      group: 1000
    seLinuxOptions:
      level: shared  # Use 'z' flag for shared access
---
apiVersion: cutepod.io/v1
kind: CuteVolume
metadata:
  name: temp-storage
spec:
  type: emptyDir
  emptyDir:
    sizeLimit: 1Gi
  securityContext:
    owner:
      group: 1000  # Set group ownership on temp directory
```

### Container with Enhanced Volume Mounts
```yaml
apiVersion: cutepod.io/v1
kind: CuteContainer
metadata:
  name: web-server
spec:
  image: nginx:latest
  uid: 1000
  gid: 1000
  volumes:
    - name: project-source
      mountPath: /usr/share/nginx/html
      subPath: frontend/dist
      readOnly: true
      mountOptions:
        seLinuxLabel: z  # Override volume default
    - name: project-source
      mountPath: /etc/nginx/nginx.conf
      subPath: configs/nginx.conf
      readOnly: true
    - name: temp-storage
      mountPath: /var/cache/nginx
      subPath: nginx-cache
```

## Error Handling

### Permission Error Detection and Resolution
```go
type VolumePermissionError struct {
    VolumeName    string
    MountPath     string
    HostPath      string
    ErrorType     PermissionErrorType
    Suggestion    string
}

type PermissionErrorType string

const (
    SELinuxDenied     PermissionErrorType = "selinux_denied"
    OwnershipMismatch PermissionErrorType = "ownership_mismatch"
    PathNotAccessible PermissionErrorType = "path_not_accessible"
    UserNSMappingFail PermissionErrorType = "user_namespace_mapping_failed"
)

func (vpm *VolumePermissionManager) diagnosePermissionError(err error, volume *VolumeResource, mount *VolumeMount) *VolumePermissionError {
    errStr := err.Error()
    
    switch {
    case strings.Contains(errStr, "permission denied") && vpm.seLinuxEnabled:
        return &VolumePermissionError{
            VolumeName: volume.GetName(),
            ErrorType:  SELinuxDenied,
            Suggestion: "Try adding seLinuxLabel: 'z' to mountOptions for shared access, or 'Z' for private access",
        }
    case strings.Contains(errStr, "operation not permitted"):
        return &VolumePermissionError{
            VolumeName: volume.GetName(),
            ErrorType:  OwnershipMismatch,
            Suggestion: "Check that the host path ownership matches the container's UID/GID or set appropriate securityContext",
        }
    case vpm.rootlessMode && strings.Contains(errStr, "user namespace"):
        return &VolumePermissionError{
            VolumeName: volume.GetName(),
            ErrorType:  UserNSMappingFail,
            Suggestion: "Verify that the host path is accessible within the rootless user namespace",
        }
    default:
        return &VolumePermissionError{
            VolumeName: volume.GetName(),
            ErrorType:  PathNotAccessible,
            Suggestion: "Ensure the host path exists and is accessible to the Podman process",
        }
    }
}
```

## Testing Strategy

### Unit Tests
- Volume type validation and creation
- SubPath resolution and validation
- Permission mapping logic
- SELinux label determination
- User namespace mapping

### Integration Tests
- End-to-end volume mounting with different types
- Permission handling in rootless vs rootful mode
- SELinux integration testing
- Multi-container volume sharing

### Permission Testing Matrix
```go
type PermissionTestCase struct {
    Name           string
    RootlessMode   bool
    SELinuxEnabled bool
    VolumeType     VolumeType
    ContainerUID   *int64
    ExpectedResult PermissionTestResult
}

var permissionTestCases = []PermissionTestCase{
    {
        Name:           "rootless_selinux_hostpath",
        RootlessMode:   true,
        SELinuxEnabled: true,
        VolumeType:     VolumeTypeHostPath,
        ContainerUID:   &[]int64{1000}[0],
        ExpectedResult: PermissionTestResult{ShouldSucceed: true, ExpectedSELinuxLabel: "z"},
    },
    // ... more test cases
}
```

## Migration Strategy

Since backward compatibility is not required, the implementation will:

1. **Replace existing volume mount structure** with the enhanced version
2. **Update all existing examples** to use the new syntax
3. **Provide clear migration documentation** showing before/after examples
4. **Include validation** that rejects old syntax with helpful error messages

## Performance Considerations

- **Path Resolution Caching**: Cache resolved subPath calculations to avoid repeated filesystem operations
- **Permission Check Optimization**: Cache SELinux and user namespace detection results
- **Lazy Directory Creation**: Only create directories when containers are actually started
- **Concurrent Mount Handling**: Handle multiple containers mounting the same volume efficiently

## Security Considerations

- **Path Traversal Prevention**: Validate subPath doesn't contain ".." or other traversal attempts
- **SELinux Label Isolation**: Ensure proper labeling prevents unauthorized cross-container access
- **User Namespace Boundaries**: Respect rootless Podman's user namespace isolation
- **Host Path Validation**: Restrict hostPath to allowed directories based on configuration