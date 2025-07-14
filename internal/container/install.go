package container

import (
	"context"
	"cutepod/internal/object"
	"fmt"
	"os"
	"time"

	"github.com/containers/common/libnetwork/types"
	"github.com/containers/podman/v5/pkg/bindings"
	"github.com/containers/podman/v5/pkg/bindings/containers"
	"github.com/containers/podman/v5/pkg/bindings/images"
	"github.com/containers/podman/v5/pkg/specgen"
	"github.com/opencontainers/runtime-spec/specs-go"
)

func (c *CuteContainer) Install(context context.Context, target object.InstallTarget) error {
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

func (c *CuteContainer) buildSpec(t object.InstallTarget) *specgen.SpecGenerator {
	generator := specgen.SpecGenerator{
		ContainerBasicConfig: specgen.ContainerBasicConfig{
			Name: t.GetContainerName(c),
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

func (c *CuteContainer) GetName() string {
	return c.Name
}

func (c *CuteContainer) GetNamespace() string {
	return c.Namespace
}

func (c *CuteContainer) ComputeChanges(ctx context.Context, t object.InstallTarget) ([]object.ConfigChange, error) {
	ctx, err := bindings.NewConnection(ctx, GetPodmanURI())
	if err != nil {
		return nil, fmt.Errorf("unable to connect to podman: %v", err)
	}

	inspect, err := containers.Inspect(ctx, t.GetContainerName(c), &containers.InspectOptions{})
	if err != nil {
		return nil, fmt.Errorf("unable to inspect container: %v", err)
	}

	image, err := images.GetImage(ctx, c.Spec.Image, &images.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("unable to check image: %v", err)
	}

	return Compare(t, c, inspect, image.ImageData)
}

func (c *CuteContainer) Uninstall(ctx context.Context, t object.InstallTarget) error {
	ctx, err := bindings.NewConnection(ctx, GetPodmanURI())
	if err != nil {
		return fmt.Errorf("unable to connect to podman: %v", err)
	}

	timeout, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	u := uint(15)
	err = containers.Stop(timeout, t.GetContainerName(c), &containers.StopOptions{Timeout: &u})
	if err != nil {
		return fmt.Errorf("unable to stop container: %v", err)
	}

	_, err = containers.Remove(ctx, t.GetContainerName(c), &containers.RemoveOptions{})
	if err != nil {
		return fmt.Errorf("unable to remove container: %v", err)
	}

	return nil
}
