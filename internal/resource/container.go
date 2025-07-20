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
	Name          string `json:"name"`                    // Volume name reference (required)
	ContainerPath string `json:"containerPath,omitempty"` // Deprecated: use MountPath
	MountPath     string `json:"mountPath"`               // Container mount path
	ReadOnly      bool   `json:"readOnly,omitempty"`
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

	return errs
}
