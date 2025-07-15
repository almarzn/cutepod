package container

import (
	"context"
	"cutepod/internal/object"
	"fmt"
	"slices"

	"github.com/containers/podman/v5/pkg/bindings"
	"github.com/containers/podman/v5/pkg/bindings/containers"
	"github.com/containers/podman/v5/pkg/domain/entities/types"
)

func GetChanges(ctx context.Context, t object.InstallTarget, specs []CuteContainer) ([]object.Change, error) {
	ret := make([]object.Change, 0)

	ctx, err := bindings.NewConnection(ctx, GetPodmanURI())
	if err != nil {
		return nil, fmt.Errorf("unable to connect to podman: %v", err)
	}

	all := true
	list, err := containers.List(
		ctx,
		&containers.ListOptions{
			All: &all,
			Filters: map[string][]string{
				"label": {"cutepod.Namespace=" + t.GetNamespace()},
			},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("unable to list containers: %v", err)
	}

	for _, actualContainer := range list {
		ret = appendRemoveIfStale(ret, t, actualContainer, specs)
	}

	for _, expectedContainer := range specs {
		update, err := appendAddOrUpdate(ctx, ret, t, expectedContainer, list)
		if err != nil {
			return nil, err
		}
		ret = update
	}

	return ret, nil
}

func appendRemoveIfStale(ret []object.Change, t object.InstallTarget, container types.ListContainer, specs []CuteContainer) []object.Change {
	for _, spec := range specs {
		if slices.Contains(container.Names, t.GetContainerName(&spec)) {
			return ret
		}
	}

	return append(ret, object.NewRemove(container.Names[0], t.GetNamespace(), func(ctx context.Context) error {
		ctx, err := bindings.NewConnection(ctx, GetPodmanURI())
		if err != nil {
			return fmt.Errorf("unable to connect to podman: %v", err)
		}

		err = RemoveContainer(ctx, container.Names[0])
		if err != nil {
			return fmt.Errorf("unable to remove container: %v", err)
		}

		return nil
	}))
}

func appendAddOrUpdate(ctx context.Context, ret []object.Change, t object.InstallTarget, c CuteContainer, list []types.ListContainer) ([]object.Change, error) {
	for _, spec := range list {
		if slices.Contains(spec.Names, t.GetContainerName(&c)) {
			changes, err := c.ComputeChanges(ctx, t)
			if err != nil {
				return nil, fmt.Errorf("unable to compute changes: %v", err)
			}
			if len(changes) == 0 {
				return append(ret, object.NewNone(c.Name, t.GetNamespace())), nil
			}
			return append(ret, object.NewUpdate(c.Name, t.GetNamespace(), changes, func(ctx context.Context) error {
				err := c.Uninstall(ctx, t)
				if err != nil {
					return err
				}
				return c.Install(ctx, t)
			})), nil
		}
	}

	return append(ret, object.NewAdd(c.Name, t.GetNamespace(), func(ctx context.Context) error {
		return c.Install(ctx, t)
	})), nil
}
