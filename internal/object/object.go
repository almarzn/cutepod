package object

import (
	"context"
)

type SpecChange struct {
	Path     string
	Expected string
	Actual   string
}

type Actions interface {
	Install(ctx context.Context, t InstallTarget) error
	Uninstall(ctx context.Context, t InstallTarget) error
	ComputeChanges(ctx context.Context, t InstallTarget) ([]SpecChange, error)
}

type Describe interface {
	GetName() string
	GetNamespace() string
}
