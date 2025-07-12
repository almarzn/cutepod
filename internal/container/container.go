package container

import (
	"fmt"
	"strings"

	"github.com/goccy/go-yaml"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type CuteContainer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              CuteContainerSpec `json:"spec"`
}

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
	ContainerPort int32  `json:"containerPort"`
	HostPort      int32  `json:"hostPort,omitempty"`
	Protocol      string `json:"protocol,omitempty"` // TCP or UDP
}

type VolumeMount struct {
	HostPath      string `json:"hostPath"`
	ContainerPath string `json:"containerPath"`
	ReadOnly      bool   `json:"readOnly,omitempty"`
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

func validateWithAnnotation(yml string, cc CuteContainer) []error {
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

	if cc.Spec.Image == "" {
		addErr("$.spec.image", "image must not be empty")
	}
	if cc.Spec.UID != nil && *cc.Spec.UID < 0 {
		addErr("$.spec.uid", "uid must be >= 0")
	}
	if cc.Spec.GID != nil && *cc.Spec.GID < 0 {
		addErr("$.spec.gid", "gid must be >= 0")
	}

	validRestart := map[string]bool{"Always": true, "OnFailure": true, "Never": true}
	if cc.Spec.RestartPolicy != "" && !validRestart[cc.Spec.RestartPolicy] {
		addErr("$.spec.restartPolicy", "invalid restartPolicy: must be Always, OnFailure, or Never")
	}

	for i, env := range cc.Spec.Env {
		if strings.TrimSpace(env.Name) == "" {
			addErr(fmt.Sprintf("$.spec.env[%d].name", i), "env name must not be empty")
		}
	}

	for i, port := range cc.Spec.Ports {
		if port.ContainerPort < 1 || port.ContainerPort > 65535 {
			addErr(fmt.Sprintf("$.spec.ports[%d].containerPort", i), "containerPort must be between 1 and 65535")
		}
		if port.Protocol != "" && port.Protocol != "TCP" && port.Protocol != "UDP" {
			addErr(fmt.Sprintf("$.spec.ports[%d].protocol", i), "protocol must be TCP or UDP")
		}
	}

	if cc.Spec.Health != nil {
		if cc.Spec.Health.Type == "exec" && len(cc.Spec.Health.Command) == 0 {
			addErr("$.spec.health.command", "exec health check requires non-empty command")
		}
		if cc.Spec.Health.Type != "exec" && cc.Spec.Health.Type != "http" {
			addErr("$.spec.health.type", "health.type must be 'exec' or 'http'")
		}
	}

	return errs
}
