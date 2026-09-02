package sysfs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadPool(t *testing.T) {
	tempDir := t.TempDir()
	uuid := "b295f839-ce80-4f55-8ce2-5798fe2d816c"
	poolDir := filepath.Join(tempDir, uuid)

	// Create directories
	dirs := []string{
		filepath.Join(poolDir, "allocation", "data", "raid1"),
		filepath.Join(poolDir, "allocation", "metadata", "raid1"),
		filepath.Join(poolDir, "devices"),
		filepath.Join(poolDir, "devinfo", "1"),
		filepath.Join(poolDir, "devinfo", "2"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatalf("failed to create dir %s: %v", d, err)
		}
	}

	// Write label
	_ = os.WriteFile(filepath.Join(poolDir, "label"), []byte("data_pool\n"), 0644)

	// Write allocation data
	_ = os.WriteFile(filepath.Join(poolDir, "allocation", "data", "total_bytes"), []byte("24696061952\n"), 0644)
	_ = os.WriteFile(filepath.Join(poolDir, "allocation", "data", "bytes_used"), []byte("22971301888\n"), 0644)
	_ = os.WriteFile(filepath.Join(poolDir, "allocation", "data", "disk_total"), []byte("49392123904\n"), 0644)
	_ = os.WriteFile(filepath.Join(poolDir, "allocation", "data", "disk_used"), []byte("45942603776\n"), 0644)

	// Write allocation metadata
	_ = os.WriteFile(filepath.Join(poolDir, "allocation", "metadata", "total_bytes"), []byte("1073741824\n"), 0644)
	_ = os.WriteFile(filepath.Join(poolDir, "allocation", "metadata", "bytes_used"), []byte("524288000\n"), 0644)

	// Write devices
	_ = os.WriteFile(filepath.Join(poolDir, "devices", "sda1"), []byte(""), 0644)
	_ = os.WriteFile(filepath.Join(poolDir, "devices", "sdb1"), []byte(""), 0644)

	// Write devinfo 1
	_ = os.WriteFile(filepath.Join(poolDir, "devinfo", "1", "missing"), []byte("0\n"), 0644)
	_ = os.WriteFile(filepath.Join(poolDir, "devinfo", "1", "in_fs_metadata"), []byte("1\n"), 0644)
	_ = os.WriteFile(filepath.Join(poolDir, "devinfo", "1", "writeable"), []byte("1\n"), 0644)
	_ = os.WriteFile(filepath.Join(poolDir, "devinfo", "1", "error_stats"), []byte("write_errs 0\nread_errs 0\nflush_errs 0\ncorruption_errs 0\ngeneration_errs 0\n"), 0644)

	// Write devinfo 2 (missing device)
	_ = os.WriteFile(filepath.Join(poolDir, "devinfo", "2", "missing"), []byte("1\n"), 0644)
	_ = os.WriteFile(filepath.Join(poolDir, "devinfo", "2", "in_fs_metadata"), []byte("1\n"), 0644)
	_ = os.WriteFile(filepath.Join(poolDir, "devinfo", "2", "writeable"), []byte("0\n"), 0644)
	_ = os.WriteFile(filepath.Join(poolDir, "devinfo", "2", "error_stats"), []byte("write_errs 5\nread_errs 2\nflush_errs 0\ncorruption_errs 1\ngeneration_errs 0\n"), 0644)

	reader := NewReader(tempDir)
	uuids, err := reader.ListUUIDs()
	if err != nil {
		t.Fatalf("unexpected error listing uuids: %v", err)
	}
	if len(uuids) != 1 || uuids[0] != uuid {
		t.Fatalf("expected [%s], got %v", uuid, uuids)
	}

	data, err := reader.ReadPool(uuid)
	if err != nil {
		t.Fatalf("unexpected error reading pool: %v", err)
	}

	if data.Label != "data_pool" {
		t.Errorf("expected label data_pool, got %s", data.Label)
	}
	if data.DataAlloc.Profile != "RAID1" {
		t.Errorf("expected RAID1 profile, got %s", data.DataAlloc.Profile)
	}
	if data.DataAlloc.TotalBytes != 24696061952 {
		t.Errorf("expected TotalBytes 24696061952, got %d", data.DataAlloc.TotalBytes)
	}
	if len(data.Devices) != 2 {
		t.Errorf("expected 2 devices, got %d", len(data.Devices))
	}
	if len(data.DevInfos) != 2 {
		t.Errorf("expected 2 devinfos, got %d", len(data.DevInfos))
	}
	if !data.DevInfos[2].Missing {
		t.Errorf("expected dev 2 to be missing")
	}
	if data.DevInfos[2].ErrorStats.WriteErrs != 5 {
		t.Errorf("expected dev 2 write_errs = 5, got %d", data.DevInfos[2].ErrorStats.WriteErrs)
	}
}
