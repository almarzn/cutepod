package container

import (
	"cutepod/internal/resource"
)

// CuteContainer is a compatibility wrapper around ContainerResource
type CuteContainer struct {
	*resource.ContainerResource
}

// NewCuteContainer creates a new CuteContainer
func NewCuteContainer() *CuteContainer {
	return &CuteContainer{
		ContainerResource: resource.NewContainerResource(),
	}
}

// Legacy type aliases for backward compatibility
type CuteContainerSpec = resource.CuteContainerSpec
type EnvVar = resource.EnvVar
type ContainerPort = resource.ContainerPort
type VolumeMount = resource.VolumeMount
type HealthCheck = resource.HealthCheck
type HTTPProbe = resource.HTTPProbe
type SecurityContext = resource.SecurityContext
type Capabilities = resource.Capabilities
type ResourceRequirements = resource.ResourceRequirements
type ResourceList = resource.ResourceList
type SecretReference = resource.SecretReference

func validateWithAnnotation(yml string, cc CuteContainer) []error {
	return cc.ContainerResource.Validate(yml)
}
