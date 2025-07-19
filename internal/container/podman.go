package container

import (
	"context"
	"cutepod/internal/podman"
	"fmt"
	"time"
)

// RemoveContainer removes a container using the provided Podman client
func RemoveContainer(ctx context.Context, client podman.PodmanClient, name string) error {
	if err := client.Connect(ctx); err != nil {
		return fmt.Errorf("unable to connect to podman: %v", err)
	}
	defer client.Close()

	timeout, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	err := client.StopContainer(timeout, name, 15)
	if err != nil {
		return fmt.Errorf("unable to stop container: %v", err)
	}

	err = client.RemoveContainer(ctx, name)
	if err != nil {
		return fmt.Errorf("unable to remove container: %v", err)
	}

	return nil
}
