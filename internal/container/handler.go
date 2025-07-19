package container

import (
	"context"
	"cutepod/internal/object"
	"cutepod/internal/podman"
	"fmt"
	"slices"

	"github.com/containers/podman/v5/pkg/domain/entities/types"
)

func GetChanges(ctx context.Context, client podman.PodmanClient, t object.InstallTarget, specs []CuteContainer) ([]object.Change, error) {
	ret := make([]object.Change, 0)

	// Use connected client wrapper for automatic connection management
	connectedClient := podman.NewConnectedClient(client)
	defer connectedClient.Close()

	podmanClient, err := connectedClient.GetClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to podman: %v", err)
	}

	list, err := podmanClient.ListContainers(
		ctx,
		map[string][]string{
			"label": {"cutepod.Namespace=" + t.GetNamespace()},
		},
		true,
	)
	if err != nil {
		return nil, fmt.Errorf("unable to list containers: %v", err)
	}

	for _, actualContainer := range list {
		ret = appendRemoveIfStale(ret, t, actualContainer, specs, podmanClient)
	}

	for _, expectedContainer := range specs {
		update, err := appendAddOrUpdate(ctx, ret, t, expectedContainer, list, podmanClient)
		if err != nil {
			return nil, err
		}
		ret = update
	}

	return ret, nil
}

func appendRemoveIfStale(ret []object.Change, t object.InstallTarget, container types.ListContainer, specs []CuteContainer, client podman.PodmanClient) []object.Change {
	for _, spec := range specs {
		if slices.Contains(container.Names, t.GetContainerName(&spec)) {
			return ret
		}
	}

	return append(ret, object.NewRemove(container.Names[0], t.GetNamespace(), func(ctx context.Context) error {
		err := RemoveContainer(ctx, client, container.Names[0])
		if err != nil {
			return fmt.Errorf("unable to remove container: %v", err)
		}

		return nil
	}))
}

func appendAddOrUpdate(ctx context.Context, ret []object.Change, t object.InstallTarget, c CuteContainer, list []types.ListContainer, client podman.PodmanClient) ([]object.Change, error) {
	for _, spec := range list {
		if slices.Contains(spec.Names, t.GetContainerName(&c)) {
			changes, err := c.ComputeChanges(ctx, client, t)
			if err != nil {
				return nil, fmt.Errorf("unable to compute changes: %v", err)
			}
			if len(changes) == 0 {
				return append(ret, object.NewNone(c.Name, t.GetNamespace())), nil
			}
			return append(ret, object.NewUpdate(c.Name, t.GetNamespace(), changes, func(ctx context.Context) error {
				err := c.Uninstall(ctx, client, t)
				if err != nil {
					return err
				}
				return c.Install(ctx, client, t)
			})), nil
		}
	}

	return append(ret, object.NewAdd(c.Name, t.GetNamespace(), func(ctx context.Context) error {
		return c.Install(ctx, client, t)
	})), nil
}
