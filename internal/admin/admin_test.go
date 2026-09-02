package admin

import (
	"context"
	"testing"

	"github.com/franelfers/omarchy-btrfs-raid-manager/internal/btrfs"
)

type mockBtrfsClient struct {
	added   string
	removed string
}

func (m *mockBtrfsClient) GetScrubStatus(ctx context.Context, target string) (*btrfs.ScrubStatus, error) {
	return nil, nil
}
func (m *mockBtrfsClient) GetBalanceStatus(ctx context.Context, target string) (*btrfs.BalanceStatus, error) {
	return nil, nil
}
func (m *mockBtrfsClient) DeviceAdd(ctx context.Context, device, mountpoint string) error {
	m.added = device + ":" + mountpoint
	return nil
}
func (m *mockBtrfsClient) DeviceRemove(ctx context.Context, device, mountpoint string) error {
	m.removed = device + ":" + mountpoint
	return nil
}
func (m *mockBtrfsClient) DeviceReplace(ctx context.Context, oldDev, newDev, mountpoint string) error {
	return nil
}

func TestAdminHandler(t *testing.T) {
	mock := &mockBtrfsClient{}
	h := NewHandler(mock)

	if h.btrClient == nil {
		t.Errorf("expected btrClient to be set")
	}
}

func TestExecuteValidation(t *testing.T) {
	err := Execute([]string{})
	if err == nil {
		t.Errorf("expected error for empty args")
	}

	err = Execute([]string{"unknown"})
	if err == nil {
		t.Errorf("expected error for unknown subcommand")
	}

	err = Execute([]string{"scrub"})
	if err == nil {
		t.Errorf("expected error for insufficient scrub args")
	}
}
