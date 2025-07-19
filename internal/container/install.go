package container

import (
	"context"
	"cutepod/internal/object"
	"cutepod/internal/podman"
	"fmt"
	"os"

	"github.com/containers/common/libnetwork/types"
	"github.com/containers/podman/v5/pkg/specgen"
	"github.com/opencontainers/runtime-spec/specs-go"
)

func (c *CuteContainer) Install(ctx context.Context, client podman.PodmanClient, target object.InstallTarget) error {
	if err := client.Connect(ctx); err != nil {
		return err
	}
	defer client.Close()

	err := c.pullImage(ctx, client)
	if err != nil {
		return fmt.Errorf("unable to pull image: %v", err)
	}

	spec, err := client.CreateContainer(ctx, c.buildSpec(target))
	if err != nil {
		return fmt.Errorf("unable to create container: %v", err)
	}

	err = client.StartContainer(ctx, spec.ID)
	if err != nil {
		return fmt.Errorf("unable to start container: %v", err)
	}

	return nil
}

func GetPodmanURI() string {
	if env, b := os.LookupEnv("PODMAN_SOCK"); b {
		return env
	}

	return "unix:/run/user/1000/podman/podman.sock"
}

func (c *CuteContainer) buildSpec(t object.InstallTarget) *specgen.SpecGenerator {
	generator := specgen.SpecGenerator{
		ContainerBasicConfig: specgen.ContainerBasicConfig{
			Name: t.GetContainerName(c),
			Env:  c.getEnv(),
			Labels: map[string]string{
				"cutepod.Namespace": t.GetNamespace(),
			},
		},
		ContainerNetworkConfig: specgen.ContainerNetworkConfig{
			PortMappings: c.getPortMappings(),
		},
		ContainerStorageConfig: specgen.ContainerStorageConfig{
			Image:   c.Spec.Image,
			Mounts:  c.getMounts(),
			WorkDir: c.Spec.WorkingDir,
		},
		ContainerHealthCheckConfig: specgen.ContainerHealthCheckConfig{
			HealthLogDestination: "/tmp",
		},
		ContainerSecurityConfig: specgen.ContainerSecurityConfig{
			CapAdd:     c.capabilities().Add,
			CapDrop:    c.capabilities().Drop,
			Privileged: c.securityContext().Privileged,
		},
	}
	return &generator
}

func (c *CuteContainer) securityContext() *SecurityContext {
	securityContext := c.Spec.SecurityContext
	if securityContext == nil {
		securityContext = &SecurityContext{}
	}
	return securityContext
}

func (c *CuteContainer) capabilities() *Capabilities {
	if c.Spec.SecurityContext == nil {
		return &Capabilities{}
	}
	capabilities := c.Spec.SecurityContext.Capabilities
	if capabilities == nil {
		return &Capabilities{}
	}
	return capabilities
}

func (c *CuteContainer) getEnv() map[string]string {
	m := make(map[string]string)
	for _, env := range c.Spec.Env {
		m[env.Name] = env.Value
	}
	return m
}

func (c *CuteContainer) getMounts() []specs.Mount {
	mounts := make([]specs.Mount, 0)

	for _, mount := range c.Spec.Volumes {
		mounts = append(mounts, specs.Mount{
			Destination: mount.ContainerPath,
			Source:      mount.HostPath,
		})
	}

	return mounts
}

func (c *CuteContainer) getPortMappings() []types.PortMapping {
	portMappings := make([]types.PortMapping, 0)

	for _, port := range c.Spec.Ports {
		portMappings = append(portMappings, types.PortMapping{
			HostIP:        "",
			ContainerPort: port.ContainerPort,
			HostPort:      port.HostPort,
			Protocol:      port.Protocol,
		})
	}

	return portMappings
}

func (c *CuteContainer) pullImage(ctx context.Context, client podman.PodmanClient) error {
	image, err := client.GetImage(ctx, c.Spec.Image)
	if err == nil && image != nil {
		return nil
	}

	err = client.PullImage(ctx, c.Spec.Image)
	if err != nil {
		return fmt.Errorf("unable to pull image: %v", err)
	}

	return nil
}

func (c *CuteContainer) GetName() string {
	return c.Name
}

func (c *CuteContainer) GetNamespace() string {
	return c.Namespace
}

func (c *CuteContainer) ComputeChanges(ctx context.Context, client podman.PodmanClient, t object.InstallTarget) ([]object.SpecChange, error) {
	if err := client.Connect(ctx); err != nil {
		return nil, fmt.Errorf("unable to connect to podman: %v", err)
	}
	defer client.Close()

	inspect, err := client.InspectContainer(ctx, t.GetContainerName(c))
	if err != nil {
		return nil, fmt.Errorf("unable to inspect container: %v", err)
	}

	image, err := client.GetImage(ctx, c.Spec.Image)
	if err != nil {
		return nil, fmt.Errorf("unable to check image: %v", err)
	}

	return Compare(t, c, inspect, image)
}

func (c *CuteContainer) Uninstall(ctx context.Context, client podman.PodmanClient, t object.InstallTarget) error {
	name := t.GetContainerName(c)

	return RemoveContainer(ctx, client, name)
}
