package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/franelfers/omarchy-btrfs-raid-manager/internal/btrfs"
	"github.com/franelfers/omarchy-btrfs-raid-manager/internal/systemd"
	"github.com/franelfers/omarchy-btrfs-raid-manager/internal/udisks"
	"github.com/godbus/dbus/v5"
)

// Response represents standardized JSON output for admin commands.
type Response struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

// PrintSuccess prints a JSON success message to stdout.
func PrintSuccess(msg string) {
	resp := Response{
		Success: true,
		Message: msg,
	}
	_ = json.NewEncoder(os.Stdout).Encode(resp)
}

// PrintError prints a JSON error message to stderr.
func PrintError(err error) {
	resp := Response{
		Success: false,
		Error:   err.Error(),
	}
	_ = json.NewEncoder(os.Stderr).Encode(resp)
}

// Handler dispatches admin mutation operations.
type Handler struct {
	btrClient btrfs.Client
}

// NewHandler returns a new Handler instance.
func NewHandler(btr btrfs.Client) *Handler {
	if btr == nil {
		btr = btrfs.NewClient("")
	}
	return &Handler{btrClient: btr}
}

// Mount mounts a device via UDisks2 D-Bus.
func (h *Handler) Mount(ctx context.Context, device string) error {
	conn, err := dbus.SystemBus()
	if err != nil {
		return fmt.Errorf("failed to connect to system bus: %w", err)
	}
	defer conn.Close()

	uClient := udisks.NewClient(conn)
	mountPath, err := uClient.MountDevice(ctx, device)
	if err != nil {
		return err
	}
	PrintSuccess(fmt.Sprintf("Mounted %s at %s", device, mountPath))
	return nil
}

// Unmount unmounts a device via UDisks2 D-Bus.
func (h *Handler) Unmount(ctx context.Context, device string) error {
	conn, err := dbus.SystemBus()
	if err != nil {
		return fmt.Errorf("failed to connect to system bus: %w", err)
	}
	defer conn.Close()

	uClient := udisks.NewClient(conn)
	if err := uClient.UnmountDevice(ctx, device); err != nil {
		return err
	}
	PrintSuccess(fmt.Sprintf("Unmounted %s", device))
	return nil
}

// Scrub controls the scrub parametric service via systemd D-Bus.
func (h *Handler) Scrub(ctx context.Context, action, mountpoint string) error {
	conn, err := dbus.SystemBus()
	if err != nil {
		return fmt.Errorf("failed to connect to system bus: %w", err)
	}
	defer conn.Close()

	sysClient := systemd.NewClient(conn)
	unitName := systemd.ScrubServiceName(mountpoint)

	switch strings.ToLower(action) {
	case "start":
		if err := sysClient.StartUnit(ctx, unitName); err != nil {
			return err
		}
		PrintSuccess(fmt.Sprintf("Started scrub service %s", unitName))
	case "cancel", "stop":
		if err := sysClient.StopUnit(ctx, unitName); err != nil {
			return err
		}
		PrintSuccess(fmt.Sprintf("Stopped scrub service %s", unitName))
	default:
		return fmt.Errorf("unknown scrub action '%s', must be start or cancel", action)
	}
	return nil
}

// Balance controls the balance parametric service via systemd D-Bus.
func (h *Handler) Balance(ctx context.Context, action, mountpoint string) error {
	conn, err := dbus.SystemBus()
	if err != nil {
		return fmt.Errorf("failed to connect to system bus: %w", err)
	}
	defer conn.Close()

	sysClient := systemd.NewClient(conn)
	unitName := systemd.BalanceServiceName(mountpoint)

	switch strings.ToLower(action) {
	case "start":
		if err := sysClient.StartUnit(ctx, unitName); err != nil {
			return err
		}
		PrintSuccess(fmt.Sprintf("Started balance service %s", unitName))
	case "cancel", "stop":
		if err := sysClient.StopUnit(ctx, unitName); err != nil {
			return err
		}
		PrintSuccess(fmt.Sprintf("Stopped balance service %s", unitName))
	default:
		return fmt.Errorf("unknown balance action '%s', must be start or cancel", action)
	}
	return nil
}

// Timer enables or disables periodic maintenance timers via systemd D-Bus.
func (h *Handler) Timer(ctx context.Context, action, mountpoint, kind string) error {
	conn, err := dbus.SystemBus()
	if err != nil {
		return fmt.Errorf("failed to connect to system bus: %w", err)
	}
	defer conn.Close()

	sysClient := systemd.NewClient(conn)
	var timerName string

	switch strings.ToLower(kind) {
	case "scrub":
		timerName = systemd.ScrubTimerName(mountpoint)
	case "balance":
		timerName = systemd.BalanceTimerName(mountpoint)
	default:
		return fmt.Errorf("unknown timer kind '%s', must be 'scrub' or 'balance'", kind)
	}

	switch strings.ToLower(action) {
	case "enable":
		if err := sysClient.EnableUnit(ctx, timerName); err != nil {
			return err
		}
		_ = sysClient.StartUnit(ctx, timerName)
		PrintSuccess(fmt.Sprintf("Enabled timer %s", timerName))
	case "disable":
		if err := sysClient.DisableUnit(ctx, timerName); err != nil {
			return err
		}
		_ = sysClient.StopUnit(ctx, timerName)
		PrintSuccess(fmt.Sprintf("Disabled timer %s", timerName))
	default:
		return fmt.Errorf("unknown timer action '%s', must be enable or disable", action)
	}
	return nil
}

// DeviceAdd adds a block device guarded by Polkit.
func (h *Handler) DeviceAdd(ctx context.Context, device, mountpoint string) error {
	if os.Geteuid() != 0 {
		return runWithPolkit(ctx, "add", device, mountpoint)
	}
	if err := h.btrClient.DeviceAdd(ctx, device, mountpoint); err != nil {
		return err
	}
	PrintSuccess(fmt.Sprintf("Added device %s to %s", device, mountpoint))
	return nil
}

// DeviceRemove removes a device guarded by Polkit.
func (h *Handler) DeviceRemove(ctx context.Context, device, mountpoint string) error {
	if os.Geteuid() != 0 {
		return runWithPolkit(ctx, "remove", device, mountpoint)
	}
	if err := h.btrClient.DeviceRemove(ctx, device, mountpoint); err != nil {
		return err
	}
	PrintSuccess(fmt.Sprintf("Removed device %s from %s", device, mountpoint))
	return nil
}

// DeviceReplace replaces an existing device with a new device guarded by Polkit.
func (h *Handler) DeviceReplace(ctx context.Context, oldDev, newDev, mountpoint string) error {
	if os.Geteuid() != 0 {
		return runWithPolkit(ctx, "replace", oldDev, newDev, mountpoint)
	}
	if err := h.btrClient.DeviceReplace(ctx, oldDev, newDev, mountpoint); err != nil {
		return err
	}
	PrintSuccess(fmt.Sprintf("Replaced device %s with %s in %s", oldDev, newDev, mountpoint))
	return nil
}

func runWithPolkit(ctx context.Context, subcommand string, args ...string) error {
	execPath, err := os.Executable()
	if err != nil {
		execPath = "raid-manager"
	}

	cmdArgs := append([]string{execPath, "admin", subcommand}, args...)
	cmd := exec.CommandContext(ctx, "pkexec", cmdArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pkexec elevation failed: %w", err)
	}
	return nil
}

// Execute parses command line arguments and runs the respective admin subcommand.
func Execute(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("admin requires a subcommand (mount, unmount, scrub, balance, timer, add, remove, replace)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	h := NewHandler(nil)
	subcmd := args[0]

	switch subcmd {
	case "mount":
		if len(args) < 2 {
			return fmt.Errorf("usage: raid-manager admin mount <device>")
		}
		return h.Mount(ctx, args[1])
	case "unmount":
		if len(args) < 2 {
			return fmt.Errorf("usage: raid-manager admin unmount <device>")
		}
		return h.Unmount(ctx, args[1])
	case "scrub":
		if len(args) < 3 {
			return fmt.Errorf("usage: raid-manager admin scrub <start|cancel> <mountpoint>")
		}
		return h.Scrub(ctx, args[1], args[2])
	case "balance":
		if len(args) < 3 {
			return fmt.Errorf("usage: raid-manager admin balance <start|cancel> <mountpoint>")
		}
		return h.Balance(ctx, args[1], args[2])
	case "timer":
		if len(args) < 4 {
			return fmt.Errorf("usage: raid-manager admin timer <enable|disable> <mountpoint> <scrub|balance>")
		}
		return h.Timer(ctx, args[1], args[2], args[3])
	case "add":
		if len(args) < 3 {
			return fmt.Errorf("usage: raid-manager admin add <device> <mountpoint>")
		}
		return h.DeviceAdd(ctx, args[1], args[2])
	case "remove":
		if len(args) < 3 {
			return fmt.Errorf("usage: raid-manager admin remove <device> <mountpoint>")
		}
		return h.DeviceRemove(ctx, args[1], args[2])
	case "replace":
		if len(args) < 4 {
			return fmt.Errorf("usage: raid-manager admin replace <old_dev> <new_dev> <mountpoint>")
		}
		return h.DeviceReplace(ctx, args[1], args[2], args[3])
	default:
		return fmt.Errorf("unknown admin subcommand: %s", subcmd)
	}
}
