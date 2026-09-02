package udisks

import (
	"context"
	"fmt"
	"strings"

	"github.com/godbus/dbus/v5"
)

const (
	UDisks2Dest      = "org.freedesktop.UDisks2"
	UDisks2Path      = "/org/freedesktop/UDisks2"
	BlockInterface   = "org.freedesktop.UDisks2.Block"
	FSInterface      = "org.freedesktop.UDisks2.Filesystem"
	DriveInterface   = "org.freedesktop.UDisks2.Drive"
	AtaInterface     = "org.freedesktop.UDisks2.Drive.Ata"
	ObjectManagerIf  = "org.freedesktop.DBus.ObjectManager"
	PropertiesIf     = "org.freedesktop.DBus.Properties"
)

// DeviceInfo represents hardware details and SMART telemetry for a block device.
type DeviceInfo struct {
	DevNode           string  `json:"dev_node"`           // e.g. "/dev/sda1"
	DrivePath         string  `json:"drive_path"`         // e.g. "/org/freedesktop/UDisks2/drives/..."
	Model             string  `json:"model"`              // e.g. "KINGSTON SV300S37A240G"
	Serial            string  `json:"serial"`             // e.g. "50026B7754093F9B"
	SizeBytes         uint64  `json:"size_bytes"`
	MountPoints       []string `json:"mount_points"`
	SmartSupported    bool    `json:"smart_supported"`
	SmartEnabled      bool    `json:"smart_enabled"`
	SmartFailing      bool    `json:"smart_failing"`
	SmartBadSectors   int64   `json:"smart_bad_sectors"`
	SmartTemperatureC float64 `json:"smart_temperature_c"`
	SmartStatus       string  `json:"smart_status"`       // e.g. "passed", "failing", "unknown"
}

// Client interacts with UDisks2 over the system D-Bus.
type Client struct {
	conn *dbus.Conn
}

// NewClient establishes a connection to UDisks2 on the system bus.
func NewClient(conn *dbus.Conn) *Client {
	return &Client{conn: conn}
}

// GetManagedObjects queries the UDisks2 ObjectManager for all registered objects.
func (c *Client) GetManagedObjects(ctx context.Context) (map[dbus.ObjectPath]map[string]map[string]dbus.Variant, error) {
	if c.conn == nil {
		return nil, fmt.Errorf("dbus connection is nil")
	}
	obj := c.conn.Object(UDisks2Dest, UDisks2Path)
	var managed map[dbus.ObjectPath]map[string]map[string]dbus.Variant
	err := obj.CallWithContext(ctx, ObjectManagerIf+".GetManagedObjects", 0).Store(&managed)
	if err != nil {
		return nil, fmt.Errorf("failed to call GetManagedObjects: %w", err)
	}
	return managed, nil
}

// InspectBlockDevices gathers DeviceInfo indexed by device node path (e.g. "/dev/sda1" or "sda1").
func (c *Client) InspectBlockDevices(ctx context.Context) (map[string]DeviceInfo, error) {
	managed, err := c.GetManagedObjects(ctx)
	if err != nil {
		return nil, err
	}

	drives := make(map[dbus.ObjectPath]map[string]dbus.Variant)
	ataDrives := make(map[dbus.ObjectPath]map[string]dbus.Variant)

	// First pass: collect drives and ATA SMART data
	for path, ifaces := range managed {
		if dProps, ok := ifaces[DriveInterface]; ok {
			drives[path] = dProps
		}
		if ataProps, ok := ifaces[AtaInterface]; ok {
			ataDrives[path] = ataProps
		}
	}

	result := make(map[string]DeviceInfo)

	// Second pass: collect block devices
	for _, ifaces := range managed {
		blockProps, hasBlock := ifaces[BlockInterface]
		if !hasBlock {
			continue
		}

		devBytes, ok := blockProps["Device"].Value().([]byte)
		if !ok || len(devBytes) == 0 {
			continue
		}
		devNode := strings.TrimRight(string(devBytes), "\x00")
		baseName := devNode
		if idx := strings.LastIndex(devNode, "/"); idx != -1 {
			baseName = devNode[idx+1:]
		}

		info := DeviceInfo{
			DevNode:     devNode,
			SmartStatus: "unknown",
		}

		// Mount points from Filesystem interface
		if fsProps, ok := ifaces[FSInterface]; ok {
			if mpVariant, ok := fsProps["MountPoints"]; ok {
				if mps, ok := mpVariant.Value().([][]byte); ok {
					for _, mpBytes := range mps {
						info.MountPoints = append(info.MountPoints, strings.TrimRight(string(mpBytes), "\x00"))
					}
				}
			}
		}

		// Drive association
		if driveObjPath, ok := blockProps["Drive"].Value().(dbus.ObjectPath); ok && driveObjPath != "/" {
			info.DrivePath = string(driveObjPath)

			if dProps, ok := drives[driveObjPath]; ok {
				if model, ok := dProps["Model"].Value().(string); ok {
					info.Model = model
				}
				if serial, ok := dProps["Serial"].Value().(string); ok {
					info.Serial = serial
				}
				if size, ok := dProps["Size"].Value().(uint64); ok {
					info.SizeBytes = size
				}
			}

			if ataProps, ok := ataDrives[driveObjPath]; ok {
				if supp, ok := ataProps["SmartSupported"].Value().(bool); ok {
					info.SmartSupported = supp
				}
				if en, ok := ataProps["SmartEnabled"].Value().(bool); ok {
					info.SmartEnabled = en
				}
				if failing, ok := ataProps["SmartFailing"].Value().(bool); ok {
					info.SmartFailing = failing
					if failing {
						info.SmartStatus = "failing"
					} else if info.SmartSupported {
						info.SmartStatus = "passed"
					}
				}
				if bad, ok := ataProps["SmartNumBadSectors"].Value().(int64); ok {
					info.SmartBadSectors = bad
					if bad > 0 && info.SmartStatus != "failing" {
						info.SmartStatus = "warning"
					}
				}
				if tempK, ok := ataProps["SmartTemperature"].Value().(float64); ok && tempK > 0 {
					info.SmartTemperatureC = tempK - 273.15
				}
			}
		}

		result[devNode] = info
		result[baseName] = info
	}

	return result, nil
}

// MountDevice mounts a block device using UDisks2 Filesystem.Mount.
func (c *Client) MountDevice(ctx context.Context, devNode string) (string, error) {
	blockPath, err := c.findBlockObject(ctx, devNode)
	if err != nil {
		return "", err
	}

	obj := c.conn.Object(UDisks2Dest, blockPath)
	var mountPath string
	options := make(map[string]dbus.Variant)
	err = obj.CallWithContext(ctx, FSInterface+".Mount", 0, options).Store(&mountPath)
	if err != nil {
		return "", fmt.Errorf("udisks mount failed for %s: %w", devNode, err)
	}
	return mountPath, nil
}

// UnmountDevice unmounts a block device using UDisks2 Filesystem.Unmount.
func (c *Client) UnmountDevice(ctx context.Context, devNode string) error {
	blockPath, err := c.findBlockObject(ctx, devNode)
	if err != nil {
		return err
	}

	obj := c.conn.Object(UDisks2Dest, blockPath)
	options := make(map[string]dbus.Variant)
	err = obj.CallWithContext(ctx, FSInterface+".Unmount", 0, options).Err
	if err != nil {
		return fmt.Errorf("udisks unmount failed for %s: %w", devNode, err)
	}
	return nil
}

func (c *Client) findBlockObject(ctx context.Context, target string) (dbus.ObjectPath, error) {
	managed, err := c.GetManagedObjects(ctx)
	if err != nil {
		return "", err
	}

	targetNode := target
	if !strings.HasPrefix(targetNode, "/dev/") && !strings.Contains(targetNode, "/") {
		targetNode = "/dev/" + target
	}

	for path, ifaces := range managed {
		blockProps, hasBlock := ifaces[BlockInterface]
		if !hasBlock {
			continue
		}
		if devBytes, ok := blockProps["Device"].Value().([]byte); ok {
			node := strings.TrimRight(string(devBytes), "\x00")
			if node == targetNode {
				return path, nil
			}
		}
		// Also match by filesystem UUID or symlinks
		if uuid, ok := blockProps["IdUUID"].Value().(string); ok && uuid == target {
			return path, nil
		}
	}
	return "", fmt.Errorf("block device %s not found in UDisks2", target)
}

// SubscribeEvents adds match rules for UDisks2 properties changed and object lifecycle signals.
func (c *Client) SubscribeEvents(conn *dbus.Conn) error {
	rules := []string{
		fmt.Sprintf("type='signal',sender='%s',path_namespace='%s',interface='%s'", UDisks2Dest, UDisks2Path, PropertiesIf),
		fmt.Sprintf("type='signal',sender='%s',path='%s',interface='%s'", UDisks2Dest, UDisks2Path, ObjectManagerIf),
	}
	for _, rule := range rules {
		call := conn.BusObject().Call("org.freedesktop.DBus.AddMatch", 0, rule)
		if call.Err != nil {
			return fmt.Errorf("failed to add dbus match rule %q: %w", rule, call.Err)
		}
	}
	return nil
}
