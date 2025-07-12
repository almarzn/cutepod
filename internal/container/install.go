package container

import (
	"context"
	"cutepod/internal/target"
	"fmt"
	"os"

	"github.com/containers/podman/v5/pkg/bindings"
	"github.com/containers/podman/v5/pkg/bindings/containers"
	"github.com/containers/podman/v5/pkg/bindings/images"
	"github.com/containers/podman/v5/pkg/specgen"
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

func (c *CuteContainer) buildSpec(target target.InstallTarget) *specgen.SpecGenerator {
	generator := specgen.SpecGenerator{
		ContainerBasicConfig: specgen.ContainerBasicConfig{
			Name: target.GetContainerName(c.Namespace, c.Name),
		},
		ContainerStorageConfig: specgen.ContainerStorageConfig{
			Image: c.Spec.Image,
		},
		ContainerHealthCheckConfig: specgen.ContainerHealthCheckConfig{
			HealthConfig:               nil,
			HealthCheckOnFailureAction: 0,
			StartupHealthConfig:        nil,
			HealthLogDestination:       "/tmp",
			HealthMaxLogCount:          0,
			HealthMaxLogSize:           0,
		},
	}
	return &generator
}

func (c *CuteContainer) pullImage(ctx context.Context) error {
	options := new(images.PullOptions)
	_, err := images.Pull(ctx, c.Spec.Image, options)
	return err
}
