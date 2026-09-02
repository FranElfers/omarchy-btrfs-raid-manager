package parser

import (
	"strings"
	"testing"

	"github.com/franelfers/omarchy-btrfs-raid-manager/internal/btrfs"
	"github.com/franelfers/omarchy-btrfs-raid-manager/internal/sysfs"
	"github.com/franelfers/omarchy-btrfs-raid-manager/internal/systemd"
	"github.com/franelfers/omarchy-btrfs-raid-manager/internal/udisks"
)

func TestAggregatePoolHealthy(t *testing.T) {
	sysData := &sysfs.PoolSysfsData{
		UUID:  "1111-2222-3333-4444",
		Label: "tank",
		DataAlloc: sysfs.AllocationStats{
			TotalBytes: 20000000000,
			BytesUsed:  10000000000,
			Profile:    "RAID1",
		},
		Devices: []string{"sda1", "sdb1"},
		DevInfos: map[int]sysfs.DevInfo{
			1: {DevID: 1, Missing: false},
			2: {DevID: 2, Missing: false},
		},
	}

	devices := map[string]udisks.DeviceInfo{
		"sda1": {
			DevNode:     "/dev/sda1",
			Model:       "Crucial SSD",
			Serial:      "SN123",
			SizeBytes:   10000000000,
			MountPoints: []string{"/mnt/tank"},
			SmartStatus: "passed",
		},
		"sdb1": {
			DevNode:     "/dev/sdb1",
			Model:       "Crucial SSD",
			Serial:      "SN456",
			SizeBytes:   10000000000,
			SmartStatus: "passed",
		},
	}

	info := AggregatePool(sysData, devices, nil, nil, nil, nil, nil, nil)

	if info.Status != "healthy" {
		t.Errorf("expected healthy status, got %s", info.Status)
	}
	if info.IsDegraded {
		t.Errorf("expected is_degraded false")
	}
	if info.FreeBytes != 10000000000 {
		t.Errorf("expected 10GB free, got %d", info.FreeBytes)
	}
	if info.PercentUsed != 50.0 {
		t.Errorf("expected 50%% used, got %f", info.PercentUsed)
	}
	if !info.IsMounted || info.Mountpoint != "/mnt/tank" {
		t.Errorf("expected mounted at /mnt/tank, got %v (%s)", info.IsMounted, info.Mountpoint)
	}

	// Test NDJSON marshaling
	state := &State{
		Pools:     []PoolInfo{info},
		Timestamp: 1700000000,
	}
	data, err := MarshalNDJSON(state)
	if err != nil {
		t.Fatalf("unexpected marshal error: %v", err)
	}
	if !strings.HasSuffix(string(data), "\n") {
		t.Errorf("expected trailing newline in NDJSON")
	}
	if strings.Count(strings.TrimSpace(string(data)), "\n") != 0 {
		t.Errorf("expected single line JSON without internal newlines")
	}
}

func TestAggregatePoolDegraded(t *testing.T) {
	sysData := &sysfs.PoolSysfsData{
		UUID:  "1111-2222-3333-4444",
		Label: "tank",
		DataAlloc: sysfs.AllocationStats{
			TotalBytes: 20000000000,
			BytesUsed:  10000000000,
			Profile:    "RAID1",
		},
		Devices: []string{"sda1"},
		DevInfos: map[int]sysfs.DevInfo{
			1: {DevID: 1, Missing: false},
			2: {DevID: 2, Missing: true}, // Dev 2 is missing!
		},
	}

	devices := map[string]udisks.DeviceInfo{
		"sda1": {
			DevNode:     "/dev/sda1",
			MountPoints: []string{"/mnt/tank"},
			SmartStatus: "passed",
		},
	}

	info := AggregatePool(sysData, devices, nil, nil, nil, nil, nil, nil)

	if info.Status != "degraded" {
		t.Errorf("expected degraded status, got %s", info.Status)
	}
	if !info.IsDegraded {
		t.Errorf("expected is_degraded to be true")
	}
}

func TestAggregatePoolWorking(t *testing.T) {
	sysData := &sysfs.PoolSysfsData{
		UUID:  "1111-2222-3333-4444",
		Label: "tank",
		DataAlloc: sysfs.AllocationStats{
			TotalBytes: 20000000000,
			BytesUsed:  10000000000,
			Profile:    "RAID1",
		},
		Devices: []string{"sda1", "sdb1"},
		DevInfos: map[int]sysfs.DevInfo{
			1: {DevID: 1, Missing: false},
			2: {DevID: 2, Missing: false},
		},
	}

	devices := map[string]udisks.DeviceInfo{
		"sda1": {
			DevNode:     "/dev/sda1",
			MountPoints: []string{"/mnt/tank"},
			SmartStatus: "passed",
		},
		"sdb1": {
			DevNode:     "/dev/sdb1",
			SmartStatus: "passed",
		},
	}

	scrubStatus := &btrfs.ScrubStatus{
		Active:          true,
		ProgressPercent: 42.5,
		StatusText:      "running",
	}

	scrubService := &systemd.UnitStatus{
		IsActive: true,
	}

	info := AggregatePool(sysData, devices, scrubStatus, nil, scrubService, nil, nil, nil)

	if info.Status != "working" {
		t.Errorf("expected working status, got %s", info.Status)
	}
	if info.ActiveOperation != "scrub" {
		t.Errorf("expected active operation scrub, got %s", info.ActiveOperation)
	}
	if info.OperationProgress != 42.5 {
		t.Errorf("expected operation progress 42.5, got %f", info.OperationProgress)
	}
}
