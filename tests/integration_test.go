package tests

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/franelfers/omarchy-btrfs-raid-manager/internal/parser"
	"github.com/franelfers/omarchy-btrfs-raid-manager/internal/sysfs"
	"github.com/franelfers/omarchy-btrfs-raid-manager/internal/udisks"
)

func TestMockHealthyFixture(t *testing.T) {
	fixtureDir := filepath.Join("fixtures", "sysfs", "healthy")
	reader := sysfs.NewReader(fixtureDir)

	uuids, err := reader.ListUUIDs()
	if err != nil {
		t.Fatalf("failed to list uuids: %v", err)
	}
	if len(uuids) != 1 {
		t.Fatalf("expected 1 uuid, got %d", len(uuids))
	}

	data, err := reader.ReadPool(uuids[0])
	if err != nil {
		t.Fatalf("failed to read pool: %v", err)
	}

	devices := map[string]udisks.DeviceInfo{
		"sda1": {
			DevNode:     "/dev/sda1",
			Model:       "Samsung 870 EVO",
			Serial:      "S5Y4NJ0R",
			SizeBytes:   5000000000,
			MountPoints: []string{"/mnt/backup"},
			SmartStatus: "passed",
		},
		"sdb1": {
			DevNode:     "/dev/sdb1",
			Model:       "Samsung 870 EVO",
			Serial:      "S5Y4NJ0S",
			SizeBytes:   5000000000,
			SmartStatus: "passed",
		},
	}

	pool := parser.AggregatePool(data, devices, nil, nil, nil, nil, nil, nil)

	if pool.Status != "healthy" {
		t.Errorf("expected status healthy, got %s", pool.Status)
	}
	if pool.IsDegraded {
		t.Errorf("expected is_degraded false")
	}
	if pool.FreeBytes != 6000000000 {
		t.Errorf("expected free_bytes 6000000000, got %d", pool.FreeBytes)
	}
	if pool.PercentUsed != 40.0 {
		t.Errorf("expected 40%% used, got %f", pool.PercentUsed)
	}
	if pool.RaidProfile != "RAID1" {
		t.Errorf("expected RAID1 profile, got %s", pool.RaidProfile)
	}
	if len(pool.Devices) != 2 {
		t.Errorf("expected 2 devices, got %d", len(pool.Devices))
	}

	state := &parser.State{
		Pools:     []parser.PoolInfo{pool},
		Timestamp: 1788390000,
	}
	ndjson, err := parser.MarshalNDJSON(state)
	if err != nil {
		t.Fatalf("failed to marshal NDJSON: %v", err)
	}

	var decoded parser.State
	if err := json.Unmarshal([]byte(strings.TrimSpace(string(ndjson))), &decoded); err != nil {
		t.Fatalf("failed to unmarshal NDJSON output: %v", err)
	}
	if decoded.Pools[0].UUID != uuids[0] {
		t.Errorf("decoded UUID %s != expected %s", decoded.Pools[0].UUID, uuids[0])
	}
}

func TestMockDegradedFixture(t *testing.T) {
	fixtureDir := filepath.Join("fixtures", "sysfs", "degraded")
	reader := sysfs.NewReader(fixtureDir)

	uuids, err := reader.ListUUIDs()
	if err != nil {
		t.Fatalf("failed to list uuids: %v", err)
	}
	if len(uuids) != 1 {
		t.Fatalf("expected 1 uuid, got %d", len(uuids))
	}

	data, err := reader.ReadPool(uuids[0])
	if err != nil {
		t.Fatalf("failed to read pool: %v", err)
	}

	devices := map[string]udisks.DeviceInfo{
		"sda1": {
			DevNode:     "/dev/sda1",
			MountPoints: []string{"/mnt/datos"},
			SmartStatus: "passed",
		},
	}

	pool := parser.AggregatePool(data, devices, nil, nil, nil, nil, nil, nil)

	if pool.Status != "degraded" {
		t.Errorf("expected status degraded, got %s", pool.Status)
	}
	if !pool.IsDegraded {
		t.Errorf("expected is_degraded true")
	}
	if pool.FreeBytes != 8000000000 {
		t.Errorf("expected free_bytes 8000000000, got %d", pool.FreeBytes)
	}
	if pool.PercentUsed != 60.0 {
		t.Errorf("expected 60%% used, got %f", pool.PercentUsed)
	}

	// Verify missing device is flagged
	var foundMissing bool
	for _, dev := range pool.Devices {
		if dev.Missing {
			foundMissing = true
			if dev.WriteErrs != 12 || dev.ReadErrs != 4 {
				t.Errorf("unexpected error counters on missing device: %+v", dev)
			}
		}
	}
	if !foundMissing {
		t.Errorf("expected to find missing device in pool devices")
	}
}
