package container

import (
	"context"
	"cutepod/internal/target"
	"fmt"
	"os"

	"github.com/containers/common/libnetwork/types"
	"github.com/containers/podman/v5/pkg/bindings"
	"github.com/containers/podman/v5/pkg/bindings/containers"
	"github.com/containers/podman/v5/pkg/bindings/images"
	"github.com/containers/podman/v5/pkg/specgen"
	"github.com/opencontainers/runtime-spec/specs-go"
)

func (c *CuteContainer) Install(context context.Context, target target.InstallTarget) error {
	ctx, err := bindings.NewConnection(context, GetPodmanURI())
	if err != nil {
		return err
	}

	err = c.pullImage(ctx)
	if err != nil {
		return fmt.Errorf("unable to pull image: %v", err)
	}

	options := &containers.CreateOptions{}
	spec, err := containers.CreateWithSpec(ctx, c.buildSpec(target), options)
	if err != nil {
		return fmt.Errorf("unable to create container: %v", err)
	}

	err = containers.Start(ctx, spec.ID, &containers.StartOptions{})
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

func (c *CuteContainer) buildSpec(t target.InstallTarget) *specgen.SpecGenerator {
	generator := specgen.SpecGenerator{
		ContainerBasicConfig: specgen.ContainerBasicConfig{
			Name: t.GetContainerName(c.Namespace, c.Name),
			Env:  c.getEnv(),
			Labels: map[string]string{
				"cutepod.Namespace": t.GetNamespace(c.Namespace),
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

func (c *CuteContainer) pullImage(ctx context.Context) error {
	image, err := images.GetImage(ctx, c.Spec.Image, &images.GetOptions{})
	if err != nil {
		return fmt.Errorf("unable to check image: %v", err)
	}

	if image != nil {
		return nil
	}

	options := new(images.PullOptions)
	_, err = images.Pull(ctx, c.Spec.Image, options)

	return err
}
