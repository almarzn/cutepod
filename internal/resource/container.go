package resource

import (
	"fmt"
	"strings"

	"github.com/goccy/go-yaml"
)

// ContainerResource represents a container resource that implements the Resource interface
type ContainerResource struct {
	BaseResource `json:",inline"`
	Spec         CuteContainerSpec `json:"spec"`
}

// CuteContainerSpec defines the specification for a container
type CuteContainerSpec struct {
	Image           string                `json:"image"`
	Command         []string              `json:"command,omitempty"`
	Args            []string              `json:"args,omitempty"`
	Env             []EnvVar              `json:"env,omitempty"`
	EnvFile         string                `json:"envFile,omitempty"`
	WorkingDir      string                `json:"workingDir,omitempty"`
	UID             *int64                `json:"uid,omitempty"`
	GID             *int64                `json:"gid,omitempty"`
	Pod             string                `json:"pod,omitempty"`
	Ports           []ContainerPort       `json:"ports,omitempty"`
	Volumes         []VolumeMount         `json:"volumes,omitempty"`
	Networks        []string              `json:"networks,omitempty"`
	Secrets         []SecretReference     `json:"secrets,omitempty"`
	Sysctl          map[string]string     `json:"sysctl,omitempty"`
	Health          *HealthCheck          `json:"health,omitempty"`
	SecurityContext *SecurityContext      `json:"securityContext,omitempty"`
	Resources       *ResourceRequirements `json:"resources,omitempty"`
	RestartPolicy   string                `json:"restartPolicy,omitempty"`
}

type EnvVar struct {
	Name  string `json:"name"`
	Value string `json:"value,omitempty"`
}

type ContainerPort struct {
	ContainerPort uint16 `json:"containerPort"`
	HostPort      uint16 `json:"hostPort,omitempty"`
	Protocol      string `json:"protocol,omitempty"` // TCP or UDP
}

type VolumeMount struct {
	Name          string              `json:"name"`                    // Volume name reference (required)
	ContainerPath string              `json:"containerPath,omitempty"` // Deprecated: use MountPath
	MountPath     string              `json:"mountPath"`               // Container mount path
	SubPath       string              `json:"subPath,omitempty"`       // Path within the volume from which to mount
	ReadOnly      bool                `json:"readOnly,omitempty"`
	MountOptions  *VolumeMountOptions `json:"mountOptions,omitempty"` // Podman-specific mount options
}

// VolumeMountOptions defines Podman-specific mount options
type VolumeMountOptions struct {
	SELinuxLabel string         `json:"seLinuxLabel,omitempty"` // "z", "Z", or custom SELinux label
	UIDMapping   *UIDGIDMapping `json:"uidMapping,omitempty"`   // UID mapping for rootless Podman
	GIDMapping   *UIDGIDMapping `json:"gidMapping,omitempty"`   // GID mapping for rootless Podman
}

// UIDGIDMapping defines user/group ID mapping for rootless containers
type UIDGIDMapping struct {
	ContainerID int64 `json:"containerID"` // ID inside the container
	HostID      int64 `json:"hostID"`      // ID on the host
	Size        int64 `json:"size"`        // Range size for the mapping
}

type SecretReference struct {
	Name string `json:"name"`           // Secret name reference
	Env  bool   `json:"env,omitempty"`  // Mount as environment variables
	Path string `json:"path,omitempty"` // Mount as file (optional)
}

type HealthCheck struct {
	Type               string     `json:"type"` // exec or http
	Command            []string   `json:"command,omitempty"`
	HTTPGet            *HTTPProbe `json:"httpGet,omitempty"`
	IntervalSeconds    int32      `json:"intervalSeconds,omitempty"`
	TimeoutSeconds     int32      `json:"timeoutSeconds,omitempty"`
	StartPeriodSeconds int32      `json:"startPeriodSeconds,omitempty"`
	Retries            int32      `json:"retries,omitempty"`
}

type HTTPProbe struct {
	Path string `json:"path"`
	Port int32  `json:"port"`
}

type SecurityContext struct {
	Privileged   *bool         `json:"privileged,omitempty"`
	Capabilities *Capabilities `json:"capabilities,omitempty"`
}

type Capabilities struct {
	Add  []string `json:"add,omitempty"`
	Drop []string `json:"drop,omitempty"`
}

type ResourceRequirements struct {
	Limits   ResourceList `json:"limits,omitempty"`
	Requests ResourceList `json:"requests,omitempty"`
}

type ResourceList struct {
	CPU    string `json:"cpu,omitempty"`
	Memory string `json:"memory,omitempty"`
}

// NewContainerResource creates a new ContainerResource
func NewContainerResource() *ContainerResource {
	return &ContainerResource{
		BaseResource: BaseResource{
			ResourceType: ResourceTypeContainer,
		},
	}
}

// GetDependencies returns the resources this container depends on
func (c *ContainerResource) GetDependencies() []ResourceReference {
	var deps []ResourceReference

	// Add network dependencies
	for _, network := range c.Spec.Networks {
		deps = append(deps, ResourceReference{
			Type: ResourceTypeNetwork,
			Name: network,
		})
	}

	// Add volume dependencies
	for _, volume := range c.Spec.Volumes {
		if volume.Name != "" {
			deps = append(deps, ResourceReference{
				Type: ResourceTypeVolume,
				Name: volume.Name,
			})
		}
	}

	// Add secret dependencies
	for _, secret := range c.Spec.Secrets {
		deps = append(deps, ResourceReference{
			Type: ResourceTypeSecret,
			Name: secret.Name,
		})
	}

	// Add pod dependency if specified
	if c.Spec.Pod != "" {
		deps = append(deps, ResourceReference{
			Type: ResourceTypePod,
			Name: c.Spec.Pod,
		})
	}

	return deps
}

// Validate validates the container specification
func (c *ContainerResource) Validate(yml string) []error {
	var errs []error

	addErr := func(jsonPath, msg string) {
		p, err := yaml.PathString(jsonPath)
		if err != nil {
			errs = append(errs, fmt.Errorf("invalid JSONPath: %s (%v)", jsonPath, err))
			return
		}
		annotated, err := p.AnnotateSource([]byte(yml), true)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to annotate YAML at %s: %v", jsonPath, err))
			return
		}
		errs = append(errs, fmt.Errorf("%s:\n%s", msg, string(annotated)))
	}

	if c.Spec.Image == "" {
		addErr("$.spec.image", "image must not be empty")
	}
	if c.Spec.UID != nil && *c.Spec.UID < 0 {
		addErr("$.spec.uid", "uid must be >= 0")
	}
	if c.Spec.GID != nil && *c.Spec.GID < 0 {
		addErr("$.spec.gid", "gid must be >= 0")
	}

	validRestart := map[string]bool{
		"no": true, "on-failure": true, "always": true, "unless-stopped": true,
		// Also accept capitalized versions for backward compatibility
		"Always": true, "OnFailure": true, "Never": true,
	}
	if c.Spec.RestartPolicy != "" && !validRestart[c.Spec.RestartPolicy] {
		addErr("$.spec.restartPolicy", "invalid restartPolicy: must be no, on-failure, always, unless-stopped, Always, OnFailure, or Never")
	}

	for i, env := range c.Spec.Env {
		if strings.TrimSpace(env.Name) == "" {
			addErr(fmt.Sprintf("$.spec.env[%d].name", i), "env name must not be empty")
		}
	}

	for i, port := range c.Spec.Ports {
		if port.ContainerPort < 1 || port.ContainerPort > 65535 {
			addErr(fmt.Sprintf("$.spec.ports[%d].containerPort", i), "containerPort must be between 1 and 65535")
		}
		if port.Protocol != "" && port.Protocol != "TCP" && port.Protocol != "UDP" {
			addErr(fmt.Sprintf("$.spec.ports[%d].protocol", i), "protocol must be TCP or UDP")
		}
	}

	if c.Spec.Health != nil {
		if c.Spec.Health.Type == "exec" && len(c.Spec.Health.Command) == 0 {
			addErr("$.spec.health.command", "exec health check requires non-empty command")
		}
		if c.Spec.Health.Type != "exec" && c.Spec.Health.Type != "http" {
			addErr("$.spec.health.type", "health.type must be 'exec' or 'http'")
		}
	}

	// Validate volume mounts
	for i, volume := range c.Spec.Volumes {
		if strings.TrimSpace(volume.Name) == "" {
			addErr(fmt.Sprintf("$.spec.volumes[%d].name", i), "volume name must not be empty")
		}

		// Validate mount path
		mountPath := volume.MountPath
		if mountPath == "" && volume.ContainerPath != "" {
			// Support deprecated ContainerPath for backward compatibility
			mountPath = volume.ContainerPath
		}
		if strings.TrimSpace(mountPath) == "" {
			addErr(fmt.Sprintf("$.spec.volumes[%d].mountPath", i), "mountPath must not be empty")
		} else if !strings.HasPrefix(mountPath, "/") {
			addErr(fmt.Sprintf("$.spec.volumes[%d].mountPath", i), "mountPath must be an absolute path starting with '/'")
		}

		// Validate subPath for security (prevent path traversal)
		if volume.SubPath != "" {
			if strings.Contains(volume.SubPath, "..") {
				addErr(fmt.Sprintf("$.spec.volumes[%d].subPath", i), "subPath must not contain '..' (path traversal not allowed)")
			}
			if strings.HasPrefix(volume.SubPath, "/") {
				addErr(fmt.Sprintf("$.spec.volumes[%d].subPath", i), "subPath must be a relative path (cannot start with '/')")
			}
			if strings.Contains(volume.SubPath, "//") {
				addErr(fmt.Sprintf("$.spec.volumes[%d].subPath", i), "subPath must not contain consecutive slashes")
			}
		}

		// Validate mount options if specified
		if volume.MountOptions != nil {
			// Validate SELinux label
			if volume.MountOptions.SELinuxLabel != "" {
				validSELinuxLabels := map[string]bool{
					"z": true, "Z": true, "shared": true, "private": true,
				}
				if !validSELinuxLabels[volume.MountOptions.SELinuxLabel] {
					addErr(fmt.Sprintf("$.spec.volumes[%d].mountOptions.seLinuxLabel", i),
						"seLinuxLabel must be one of: z, Z, shared, private")
				}
			}

			// Validate UID mapping
			if volume.MountOptions.UIDMapping != nil {
				if volume.MountOptions.UIDMapping.Size <= 0 {
					addErr(fmt.Sprintf("$.spec.volumes[%d].mountOptions.uidMapping.size", i),
						"uidMapping.size must be greater than 0")
				}
			}

			// Validate GID mapping
			if volume.MountOptions.GIDMapping != nil {
				if volume.MountOptions.GIDMapping.Size <= 0 {
					addErr(fmt.Sprintf("$.spec.volumes[%d].mountOptions.gidMapping.size", i),
						"gidMapping.size must be greater than 0")
				}
			}
		}
	}

	return errs
}
