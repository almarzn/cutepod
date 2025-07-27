package resource

import (
	"fmt"
	"path/filepath"
	"strings"
)

// VolumeResource represents a volume resource that implements the Resource interface
type VolumeResource struct {
	BaseResource `json:",inline"`
	Spec         CuteVolumeSpec `json:"spec"`
}

// CuteVolumeSpec defines the specification for a volume with Kubernetes-style volume types
type CuteVolumeSpec struct {
	Type            VolumeType             `json:"type"`
	HostPath        *HostPathVolumeSource  `json:"hostPath,omitempty"`
	EmptyDir        *EmptyDirVolumeSource  `json:"emptyDir,omitempty"`
	Volume          *VolumeVolumeSource    `json:"volume,omitempty"`
	SecurityContext *VolumeSecurityContext `json:"securityContext,omitempty"`

	// Legacy fields for backward compatibility (deprecated)
	Driver  string            `json:"driver,omitempty"`
	Options map[string]string `json:"options,omitempty"`
}

// VolumeType represents the type of volume
type VolumeType string

const (
	// New Kubernetes-style volume types
	VolumeTypeHostPath VolumeType = "hostPath"
	VolumeTypeEmptyDir VolumeType = "emptyDir"
	VolumeTypeVolume   VolumeType = "volume"

	// Legacy types for backward compatibility (deprecated)
	VolumeTypeBind VolumeType = "bind"
)

// HostPathVolumeSource represents a host path mapped into a pod
type HostPathVolumeSource struct {
	Path string        `json:"path"`
	Type *HostPathType `json:"type,omitempty"`
}

// HostPathType represents the type of host path
type HostPathType string

const (
	HostPathUnset             HostPathType = ""
	HostPathDirectoryOrCreate HostPathType = "DirectoryOrCreate"
	HostPathDirectory         HostPathType = "Directory"
	HostPathFileOrCreate      HostPathType = "FileOrCreate"
	HostPathFile              HostPathType = "File"
	HostPathSocket            HostPathType = "Socket"
	HostPathCharDevice        HostPathType = "CharDevice"
	HostPathBlockDevice       HostPathType = "BlockDevice"
)

// EmptyDirVolumeSource represents a temporary directory that shares a pod's lifetime
type EmptyDirVolumeSource struct {
	Medium    StorageMedium `json:"medium,omitempty"`
	SizeLimit *string       `json:"sizeLimit,omitempty"`
}

// StorageMedium defines ways that storage can be allocated to a volume
type StorageMedium string

const (
	StorageMediumDefault StorageMedium = ""       // Use default storage medium
	StorageMediumMemory  StorageMedium = "Memory" // Use tmpfs (RAM-backed filesystem)
)

// VolumeVolumeSource represents a named Podman volume
type VolumeVolumeSource struct {
	Driver  string            `json:"driver,omitempty"`
	Options map[string]string `json:"options,omitempty"`
}

// VolumeSecurityContext holds security configuration for volumes
type VolumeSecurityContext struct {
	SELinuxOptions *SELinuxVolumeOptions `json:"seLinuxOptions,omitempty"`
	Owner          *VolumeOwnership      `json:"owner,omitempty"`
}

// SELinuxVolumeOptions defines SELinux options for volume mounts
type SELinuxVolumeOptions struct {
	Level string `json:"level,omitempty"` // "shared" (z flag) or "private" (Z flag)
}

// VolumeOwnership defines ownership settings for volumes
type VolumeOwnership struct {
	User  *int64 `json:"user,omitempty"`  // UID for host directory ownership
	Group *int64 `json:"group,omitempty"` // GID for host directory ownership
}

// NewVolumeResource creates a new VolumeResource
func NewVolumeResource() *VolumeResource {
	return &VolumeResource{
		BaseResource: BaseResource{
			ResourceType: ResourceTypeVolume,
		},
	}
}

// GetDependencies returns the resources this volume depends on
// Volumes typically don't depend on other resources
func (v *VolumeResource) GetDependencies() []ResourceReference {
	return []ResourceReference{}
}

// Validate validates the volume specification
func (v *VolumeResource) Validate() []error {
	var errs []error

	// Validate volume type is specified
	if v.Spec.Type == "" {
		errs = append(errs, fmt.Errorf("volume type must be specified"))
		return errs
	}

	// Validate volume type is supported
	switch v.Spec.Type {
	case VolumeTypeHostPath:
		errs = append(errs, v.validateHostPath()...)
	case VolumeTypeEmptyDir:
		errs = append(errs, v.validateEmptyDir()...)
	case VolumeTypeVolume:
		errs = append(errs, v.validateVolume()...)
	case VolumeTypeBind:
		// Legacy support - treat as hostPath
		errs = append(errs, v.validateLegacyBind()...)
	default:
		errs = append(errs, fmt.Errorf("unsupported volume type: %s (supported types: hostPath, emptyDir, volume)", v.Spec.Type))
	}

	// Validate security context if specified
	if v.Spec.SecurityContext != nil {
		errs = append(errs, v.validateSecurityContext()...)
	}

	return errs
}

// validateHostPath validates hostPath volume specifications
func (v *VolumeResource) validateHostPath() []error {
	var errs []error

	if v.Spec.HostPath == nil {
		errs = append(errs, fmt.Errorf("hostPath specification is required for hostPath volume type"))
		return errs
	}

	if v.Spec.HostPath.Path == "" {
		errs = append(errs, fmt.Errorf("hostPath.path must be specified"))
	} else {
		// Validate path is absolute
		if !filepath.IsAbs(v.Spec.HostPath.Path) {
			errs = append(errs, fmt.Errorf("hostPath.path must be an absolute path, got: %s", v.Spec.HostPath.Path))
		}

		// Validate path doesn't contain dangerous patterns
		if strings.Contains(v.Spec.HostPath.Path, "..") {
			errs = append(errs, fmt.Errorf("hostPath.path cannot contain '..' for security reasons"))
		}
	}

	// Validate hostPath type if specified
	if v.Spec.HostPath.Type != nil {
		switch *v.Spec.HostPath.Type {
		case HostPathDirectoryOrCreate, HostPathDirectory, HostPathFileOrCreate,
			HostPathFile, HostPathSocket, HostPathCharDevice, HostPathBlockDevice:
			// Valid types
		default:
			errs = append(errs, fmt.Errorf("invalid hostPath.type: %s", *v.Spec.HostPath.Type))
		}
	}

	return errs
}

// validateEmptyDir validates emptyDir volume specifications
func (v *VolumeResource) validateEmptyDir() []error {
	var errs []error

	if v.Spec.EmptyDir == nil {
		errs = append(errs, fmt.Errorf("emptyDir specification is required for emptyDir volume type"))
		return errs
	}

	// Validate storage medium if specified
	if v.Spec.EmptyDir.Medium != "" {
		switch v.Spec.EmptyDir.Medium {
		case StorageMediumDefault, StorageMediumMemory:
			// Valid mediums
		default:
			errs = append(errs, fmt.Errorf("invalid emptyDir.medium: %s (supported: '', 'Memory')", v.Spec.EmptyDir.Medium))
		}
	}

	// Validate size limit format if specified
	if v.Spec.EmptyDir.SizeLimit != nil {
		sizeLimit := *v.Spec.EmptyDir.SizeLimit
		if sizeLimit != "" {
			// Basic validation for Kubernetes resource quantity format
			if !isValidResourceQuantity(sizeLimit) {
				errs = append(errs, fmt.Errorf("invalid emptyDir.sizeLimit format: %s (examples: '1Gi', '500Mi', '1000000000')", sizeLimit))
			}
		}
	}

	return errs
}

// validateVolume validates named volume specifications
func (v *VolumeResource) validateVolume() []error {
	var errs []error

	if v.Spec.Volume == nil {
		errs = append(errs, fmt.Errorf("volume specification is required for volume type"))
		return errs
	}

	// Driver is optional, defaults to "local"
	// Options are optional

	return errs
}

// validateLegacyBind validates legacy bind mount specifications
func (v *VolumeResource) validateLegacyBind() []error {
	var errs []error

	// For legacy bind mounts, we expect the old hostPath field
	if v.Spec.Driver == "" && v.Spec.Options == nil {
		errs = append(errs, fmt.Errorf("legacy bind volume type is deprecated, use hostPath volume type instead"))
	}

	return errs
}

// validateSecurityContext validates volume security context
func (v *VolumeResource) validateSecurityContext() []error {
	var errs []error

	sc := v.Spec.SecurityContext

	// Validate SELinux options
	if sc.SELinuxOptions != nil {
		if sc.SELinuxOptions.Level != "" {
			switch sc.SELinuxOptions.Level {
			case "shared", "private":
				// Valid levels
			default:
				errs = append(errs, fmt.Errorf("invalid seLinuxOptions.level: %s (supported: 'shared', 'private')", sc.SELinuxOptions.Level))
			}
		}
	}

	// Validate ownership
	if sc.Owner != nil {
		if sc.Owner.User != nil && *sc.Owner.User < 0 {
			errs = append(errs, fmt.Errorf("owner.user must be >= 0, got: %d", *sc.Owner.User))
		}
		if sc.Owner.Group != nil && *sc.Owner.Group < 0 {
			errs = append(errs, fmt.Errorf("owner.group must be >= 0, got: %d", *sc.Owner.Group))
		}
	}

	return errs
}

// isValidResourceQuantity performs basic validation of Kubernetes resource quantity format
func isValidResourceQuantity(quantity string) bool {
	if quantity == "" {
		return false
	}

	// Simple regex-like validation for common formats
	// This is a basic check - in production, you might want to use a proper parser
	// Check longer suffixes first to avoid empty string matching everything
	validSuffixes := []string{"Ki", "Mi", "Gi", "Ti", "Pi", "Ei", "k", "M", "G", "T", "P", "E", ""}

	for _, suffix := range validSuffixes {
		if strings.HasSuffix(quantity, suffix) {
			numberPart := strings.TrimSuffix(quantity, suffix)
			if numberPart == "" {
				return false
			}
			// Check if the remaining part is a valid number (basic check)
			dotCount := 0
			for _, char := range numberPart {
				if char >= '0' && char <= '9' {
					continue
				} else if char == '.' {
					dotCount++
					if dotCount > 1 {
						return false // Multiple dots not allowed
					}
				} else {
					return false // Invalid character
				}
			}
			return true
		}
	}

	return false
}
