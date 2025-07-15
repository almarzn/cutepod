package object

import (
	"context"
	"fmt"
)

type Change interface {
	// GetKind returns Remove, Add, or Update
	GetKind() string
	GetName() string
	GetNamespace() string
	Describe() string
	Execute(ctx context.Context) error
}

type Remove struct {
	name      string
	namespace string
	execute   func(ctx context.Context) error
}

func NewRemove(name string, namespace string, execute func(ctx context.Context) error) *Remove {
	return &Remove{name: name, namespace: namespace, execute: execute}
}

func (d *Remove) Describe() string {
	return "<removed>"
}

func (d *Remove) GetKind() string {
	return "Remove"
}

func (d *Remove) GetName() string {
	return d.name
}

func (d *Remove) GetNamespace() string {
	return d.namespace
}

func (d *Remove) Execute(ctx context.Context) error {
	return d.execute(ctx)
}

type Update struct {
	name      string
	namespace string
	changes   []SpecChange
	execute   func(ctx context.Context) error
}

func NewUpdate(name string, namespace string, changes []SpecChange, execute func(ctx context.Context) error) *Update {
	return &Update{name: name, namespace: namespace, changes: changes, execute: execute}
}

func (u *Update) Describe() string {
	first := u.changes[0].Path
	extra := ""
	if len(u.changes) > 1 {
		extra = fmt.Sprintf(", +%d more", len(u.changes)-1)
	}

	return fmt.Sprintf("%s%s", first, extra)
}

func (u *Update) GetKind() string {
	return "Update"
}

func (u *Update) GetName() string {
	return u.name
}

func (u *Update) GetNamespace() string {
	return u.namespace
}

func (u *Update) Execute(ctx context.Context) error {
	return u.execute(ctx)
}

type Add struct {
	name      string
	namespace string
	changes   []SpecChange
	execute   func(ctx context.Context) error
}

func NewAdd(name string, namespace string, execute func(ctx context.Context) error) *Add {
	return &Add{name: name, namespace: namespace, execute: execute}
}

func (a *Add) Describe() string {
	return "<added>"
}

func (a *Add) GetKind() string {
	return "Add"
}

func (a *Add) GetName() string {
	return a.name
}

func (a *Add) GetNamespace() string {
	return a.namespace
}

func (a *Add) Execute(ctx context.Context) error {
	return a.execute(ctx)
}

type None struct {
	name      string
	namespace string
}

func NewNone(name string, namespace string) *None {
	return &None{name: name, namespace: namespace}
}

func (n *None) Describe() string {
	return ""
}

func (n *None) GetKind() string {
	return "None"
}
func (n *None) GetName() string {
	return n.name
}
func (n *None) GetNamespace() string {
	return n.namespace
}
func (n *None) Execute(ctx context.Context) error {
	return nil
}
