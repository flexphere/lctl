package service

import (
	"context"
	"errors"
	"testing"

	"github.com/flexphere/lctl/internal/plist"
)

type fakeOpStore struct {
	deleteErr error
	deleted   []string
	loadResp  *plist.Agent
	loadErr   error
}

func (f *fakeOpStore) Load(label string) (*plist.Agent, error) { return f.loadResp, f.loadErr }
func (f *fakeOpStore) Delete(label string) error {
	f.deleted = append(f.deleted, label)
	return f.deleteErr
}

type fakeOpClient struct {
	calls []string
	err   error
	// per-method overrides
	bootoutErr    error
	kickstartErr  error
	kickstartCall int
}

func (f *fakeOpClient) Bootout(_ context.Context, label string) error {
	f.calls = append(f.calls, "bootout "+label)
	if f.bootoutErr != nil {
		return f.bootoutErr
	}
	return f.err
}
func (f *fakeOpClient) Kickstart(_ context.Context, label string) error {
	f.kickstartCall++
	f.calls = append(f.calls, "kickstart "+label)
	if f.kickstartErr != nil {
		return f.kickstartErr
	}
	return f.err
}
func (f *fakeOpClient) Kill(_ context.Context, label, signal string) error {
	f.calls = append(f.calls, "kill "+signal+" "+label)
	return f.err
}
func (f *fakeOpClient) Enable(_ context.Context, label string) error {
	f.calls = append(f.calls, "enable "+label)
	return f.err
}
func (f *fakeOpClient) Disable(_ context.Context, label string) error {
	f.calls = append(f.calls, "disable "+label)
	return f.err
}

func TestOpsStart(t *testing.T) {
	c := &fakeOpClient{}
	ops := NewOps(&fakeOpStore{}, c)
	if err := ops.Start(context.Background(), "com.x"); err != nil {
		t.Fatal(err)
	}
	if len(c.calls) != 1 || c.calls[0] != "kickstart com.x" {
		t.Errorf("unexpected calls: %v", c.calls)
	}
}

func TestOpsStop(t *testing.T) {
	c := &fakeOpClient{}
	ops := NewOps(&fakeOpStore{}, c)
	if err := ops.Stop(context.Background(), "com.x"); err != nil {
		t.Fatal(err)
	}
	if c.calls[0] != "kill TERM com.x" {
		t.Errorf("unexpected: %v", c.calls)
	}
}

func TestOpsEnableDisable(t *testing.T) {
	c := &fakeOpClient{}
	ops := NewOps(&fakeOpStore{}, c)
	if err := ops.Enable(context.Background(), "x"); err != nil {
		t.Fatal(err)
	}
	if err := ops.Disable(context.Background(), "x"); err != nil {
		t.Fatal(err)
	}
	if c.calls[0] != "enable x" || c.calls[1] != "disable x" {
		t.Errorf("unexpected: %v", c.calls)
	}
}

func TestOpsDeleteRemovesPlist(t *testing.T) {
	c := &fakeOpClient{}
	s := &fakeOpStore{}
	ops := NewOps(s, c)
	if err := ops.Delete(context.Background(), "com.flexphere.d"); err != nil {
		t.Fatal(err)
	}
	if len(s.deleted) != 1 || s.deleted[0] != "com.flexphere.d" {
		t.Errorf("delete not called: %v", s.deleted)
	}
	if len(c.calls) != 1 || c.calls[0] != "bootout com.flexphere.d" {
		t.Errorf("bootout not called: %v", c.calls)
	}
}

func TestOpsDeleteAbortsOnBootoutError(t *testing.T) {
	c := &fakeOpClient{bootoutErr: errors.New("boom")}
	s := &fakeOpStore{}
	ops := NewOps(s, c)
	if err := ops.Delete(context.Background(), "com.x"); err == nil {
		t.Error("expected error")
	}
	if len(s.deleted) != 0 {
		t.Errorf("plist should not be removed on bootout failure: %v", s.deleted)
	}
}

func TestOpsDeleteSurfacesStoreError(t *testing.T) {
	c := &fakeOpClient{}
	s := &fakeOpStore{deleteErr: errors.New("io")}
	ops := NewOps(s, c)
	if err := ops.Delete(context.Background(), "com.x"); err == nil {
		t.Error("expected error")
	}
}
