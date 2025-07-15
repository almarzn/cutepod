package container

import (
	"context"
	"fmt"
	"time"

	"github.com/containers/podman/v5/pkg/bindings"
	"github.com/containers/podman/v5/pkg/bindings/containers"
)

func RemoveContainer(ctx context.Context, name string) error {
	ctx, err := bindings.NewConnection(ctx, GetPodmanURI())
	if err != nil {
		return fmt.Errorf("unable to connect to podman: %v", err)
	}

	timeout, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	u := uint(15)
	err = containers.Stop(timeout, name, &containers.StopOptions{Timeout: &u})
	if err != nil {
		return fmt.Errorf("unable to stop container: %v", err)
	}

	_, err = containers.Remove(ctx, name, &containers.RemoveOptions{})
	if err != nil {
		return fmt.Errorf("unable to remove container: %v", err)
	}

	return nil
}
