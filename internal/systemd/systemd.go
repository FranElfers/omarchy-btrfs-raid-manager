package systemd

import (
	"context"
	"fmt"
	"strings"

	"github.com/godbus/dbus/v5"
)

const (
	SystemdDest    = "org.freedesktop.systemd1"
	SystemdPath    = "/org/freedesktop/systemd1"
	ManagerIf      = "org.freedesktop.systemd1.Manager"
	UnitIf         = "org.freedesktop.systemd1.Unit"
	PropertiesIf   = "org.freedesktop.DBus.Properties"
)

// UnitStatus encapsulates active and load states of a systemd unit.
type UnitStatus struct {
	Name          string `json:"name"`
	ActiveState   string `json:"active_state"`   // e.g. "active", "inactive", "failed", "activating"
	SubState      string `json:"sub_state"`      // e.g. "running", "dead", "waiting"
	UnitFileState string `json:"unit_file_state"` // e.g. "enabled", "disabled", "static"
	IsActive      bool   `json:"is_active"`
	IsEnabled     bool   `json:"is_enabled"`
}

// Client manages maintenance services and timers via systemd D-Bus.
type Client struct {
	conn *dbus.Conn
}

// NewClient returns a new systemd manager client.
func NewClient(conn *dbus.Conn) *Client {
	return &Client{conn: conn}
}

// EscapePath converts a filesystem path to a systemd instance name (equivalent to systemd-escape -p).
func EscapePath(path string) string {
	clean := strings.Trim(path, "/")
	if clean == "" {
		return "-"
	}
	var b strings.Builder
	for i := 0; i < len(clean); i++ {
		c := clean[i]
		if c == '/' {
			b.WriteByte('-')
		} else if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' {
			b.WriteByte(c)
		} else {
			// Hex encode special characters
			b.WriteString(fmt.Sprintf("\\x%02x", c))
		}
	}
	return b.String()
}

// ScrubServiceName returns the parametric service name for a given mountpoint.
func ScrubServiceName(mountpoint string) string {
	return fmt.Sprintf("btrpool-scrub@%s.service", EscapePath(mountpoint))
}

// ScrubTimerName returns the parametric timer name for a given mountpoint.
func ScrubTimerName(mountpoint string) string {
	return fmt.Sprintf("btrpool-scrub@%s.timer", EscapePath(mountpoint))
}

// BalanceServiceName returns the parametric service name for a given mountpoint.
func BalanceServiceName(mountpoint string) string {
	return fmt.Sprintf("btrpool-balance@%s.service", EscapePath(mountpoint))
}

// BalanceTimerName returns the parametric timer name for a given mountpoint.
func BalanceTimerName(mountpoint string) string {
	return fmt.Sprintf("btrpool-balance@%s.timer", EscapePath(mountpoint))
}

// GetUnitStatus fetches the live active and unit file state for a unit.
func (c *Client) GetUnitStatus(ctx context.Context, unitName string) (*UnitStatus, error) {
	if c.conn == nil {
		return nil, fmt.Errorf("dbus connection is nil")
	}

	st := &UnitStatus{
		Name:          unitName,
		ActiveState:   "inactive",
		SubState:      "dead",
		UnitFileState: "disabled",
	}

	mgrObj := c.conn.Object(SystemdDest, SystemdPath)

	// Fetch unit file state (enabled/disabled)
	var fileState string
	errFileState := mgrObj.CallWithContext(ctx, ManagerIf+".GetUnitFileState", 0, unitName).Store(&fileState)
	if errFileState == nil {
		st.UnitFileState = fileState
		st.IsEnabled = fileState == "enabled" || fileState == "enabled-runtime"
	}

	// Fetch unit runtime object
	var unitPath dbus.ObjectPath
	err := mgrObj.CallWithContext(ctx, ManagerIf+".GetUnit", 0, unitName).Store(&unitPath)
	if err != nil {
		// Unit is not currently loaded in memory
		return st, nil
	}

	unitObj := c.conn.Object(SystemdDest, unitPath)
	actVal, errAct := unitObj.GetProperty(UnitIf + ".ActiveState")
	if errAct == nil {
		if s, ok := actVal.Value().(string); ok {
			st.ActiveState = s
			st.IsActive = (s == "active" || s == "activating")
		}
	}

	subVal, errSub := unitObj.GetProperty(UnitIf + ".SubState")
	if errSub == nil {
		if s, ok := subVal.Value().(string); ok {
			st.SubState = s
		}
	}

	return st, nil
}

// StartUnit triggers starting a systemd unit.
func (c *Client) StartUnit(ctx context.Context, unitName string) error {
	if c.conn == nil {
		return fmt.Errorf("dbus connection is nil")
	}
	mgrObj := c.conn.Object(SystemdDest, SystemdPath)
	var jobPath dbus.ObjectPath
	err := mgrObj.CallWithContext(ctx, ManagerIf+".StartUnit", 0, unitName, "replace").Store(&jobPath)
	if err != nil {
		return fmt.Errorf("failed to start unit %s: %w", unitName, err)
	}
	return nil
}

// StopUnit triggers stopping an active systemd unit.
func (c *Client) StopUnit(ctx context.Context, unitName string) error {
	if c.conn == nil {
		return fmt.Errorf("dbus connection is nil")
	}
	mgrObj := c.conn.Object(SystemdDest, SystemdPath)
	var jobPath dbus.ObjectPath
	err := mgrObj.CallWithContext(ctx, ManagerIf+".StopUnit", 0, unitName, "replace").Store(&jobPath)
	if err != nil {
		return fmt.Errorf("failed to stop unit %s: %w", unitName, err)
	}
	return nil
}

// EnableUnit enables a unit file to run automatically.
func (c *Client) EnableUnit(ctx context.Context, unitName string) error {
	if c.conn == nil {
		return fmt.Errorf("dbus connection is nil")
	}
	mgrObj := c.conn.Object(SystemdDest, SystemdPath)
	var carriesInstallInfo bool
	var changes []struct {
		Type        string
		Filename    string
		Destination string
	}
	err := mgrObj.CallWithContext(ctx, ManagerIf+".EnableUnitFiles", 0, []string{unitName}, false, true).Store(&carriesInstallInfo, &changes)
	if err != nil {
		return fmt.Errorf("failed to enable unit %s: %w", unitName, err)
	}
	// Reload manager configuration
	_ = mgrObj.CallWithContext(ctx, ManagerIf+".Reload", 0).Err
	return nil
}

// DisableUnit disables a unit file.
func (c *Client) DisableUnit(ctx context.Context, unitName string) error {
	if c.conn == nil {
		return fmt.Errorf("dbus connection is nil")
	}
	mgrObj := c.conn.Object(SystemdDest, SystemdPath)
	var changes []struct {
		Type        string
		Filename    string
		Destination string
	}
	err := mgrObj.CallWithContext(ctx, ManagerIf+".DisableUnitFiles", 0, []string{unitName}, false).Store(&changes)
	if err != nil {
		return fmt.Errorf("failed to disable unit %s: %w", unitName, err)
	}
	_ = mgrObj.CallWithContext(ctx, ManagerIf+".Reload", 0).Err
	return nil
}

// SubscribeEvents registers systemd event match rules and calls Manager.Subscribe.
func (c *Client) SubscribeEvents(conn *dbus.Conn) error {
	mgrObj := conn.Object(SystemdDest, SystemdPath)
	// Must call Subscribe() method so systemd begins emitting job and unit signals
	call := mgrObj.Call(ManagerIf+".Subscribe", 0)
	if call.Err != nil {
		return fmt.Errorf("failed to call systemd Subscribe: %w", call.Err)
	}

	rules := []string{
		fmt.Sprintf("type='signal',sender='%s',path='%s',interface='%s'", SystemdDest, SystemdPath, ManagerIf),
		fmt.Sprintf("type='signal',sender='%s',interface='%s'", SystemdDest, PropertiesIf),
	}
	for _, rule := range rules {
		cRule := conn.BusObject().Call("org.freedesktop.DBus.AddMatch", 0, rule)
		if cRule.Err != nil {
			return fmt.Errorf("failed to add match rule %q: %w", rule, cRule.Err)
		}
	}
	return nil
}
