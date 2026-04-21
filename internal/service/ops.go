package service

import (
	"context"
	"fmt"

	"github.com/flexphere/lctl/internal/plist"
)

// Ops groups per-job operations exposed to the TUI.
type Ops interface {
	Start(ctx context.Context, label string) error
	Stop(ctx context.Context, label string) error
	Restart(ctx context.Context, label string) error
	Enable(ctx context.Context, label string) error
	Disable(ctx context.Context, label string) error
	Delete(ctx context.Context, label string) error
}

// PathStore is the subset of plist store required for operations that
// need to resolve or remove the on-disk plist.
type PathStore interface {
	Load(label string) (*plist.Agent, error)
	Delete(label string) error
}

// opClient widens the Client interface used by the service for
// operations to keep fake injection ergonomic in tests.
type opClient interface {
	Bootout(ctx context.Context, label string) error
	Kickstart(ctx context.Context, label string) error
	Kill(ctx context.Context, label, signal string) error
	Enable(ctx context.Context, label string) error
	Disable(ctx context.Context, label string) error
}

// opService implements Ops by composing a PathStore and an opClient.
type opService struct {
	store  PathStore
	client opClient
}

// NewOps builds an Ops backed by the given store and client. The
// arguments are the same collaborators used by JobService, so callers
// typically pass a single concrete store and client into both.
func NewOps(store PathStore, client opClient) Ops {
	return &opService{store: store, client: client}
}

func (o *opService) Start(ctx context.Context, label string) error {
	return o.client.Kickstart(ctx, label)
}

func (o *opService) Stop(ctx context.Context, label string) error {
	return o.client.Kill(ctx, label, "TERM")
}

func (o *opService) Restart(ctx context.Context, label string) error {
	return o.client.Kickstart(ctx, label)
}

func (o *opService) Enable(ctx context.Context, label string) error {
	return o.client.Enable(ctx, label)
}

func (o *opService) Disable(ctx context.Context, label string) error {
	return o.client.Disable(ctx, label)
}

// Delete unregisters the service and removes the plist file from the
// store. Either step failing aborts the whole operation.
func (o *opService) Delete(ctx context.Context, label string) error {
	if err := o.client.Bootout(ctx, label); err != nil {
		return fmt.Errorf("bootout: %w", err)
	}
	if err := o.store.Delete(label); err != nil {
		return fmt.Errorf("delete plist: %w", err)
	}
	return nil
}
