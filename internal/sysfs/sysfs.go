package sysfs

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// DevErrorStats represents kernel-reported error counters for a btrfs device.
type DevErrorStats struct {
	WriteErrs      uint64 `json:"write_errs"`
	ReadErrs       uint64 `json:"read_errs"`
	FlushErrs      uint64 `json:"flush_errs"`
	CorruptionErrs uint64 `json:"corruption_errs"`
	GenerationErrs uint64 `json:"generation_errs"`
}

// DevInfo represents kernel devinfo statistics for a device within a pool.
type DevInfo struct {
	DevID         int           `json:"dev_id"`
	Missing       bool          `json:"missing"`
	InFSMetadata  bool          `json:"in_fs_metadata"`
	Writeable     bool          `json:"writeable"`
	ReplaceTarget bool          `json:"replace_target"`
	ErrorStats    DevErrorStats `json:"error_stats"`
}

// AllocationStats represents bytes allocated and used for a btrfs allocation group (data/metadata).
type AllocationStats struct {
	TotalBytes uint64 `json:"total_bytes"`
	BytesUsed  uint64 `json:"bytes_used"`
	DiskTotal  uint64 `json:"disk_total"`
	DiskUsed   uint64 `json:"disk_used"`
	Profile    string `json:"profile"`
}

// PoolSysfsData holds raw parsed sysfs information for a specific Btrfs pool UUID.
type PoolSysfsData struct {
	UUID       string           `json:"uuid"`
	Label      string           `json:"label"`
	DataAlloc  AllocationStats  `json:"data_alloc"`
	MetaAlloc  AllocationStats  `json:"meta_alloc"`
	Devices    []string         `json:"devices"`
	DevInfos   map[int]DevInfo  `json:"dev_infos"`
}

// Reader provides methods to inspect the btrfs sysfs hierarchy.
type Reader struct {
	RootPath string
}

// NewReader creates a Reader rooted at the standard sysfs path or a custom mock path.
func NewReader(rootPath string) *Reader {
	if rootPath == "" {
		rootPath = "/sys/fs/btrfs"
	}
	return &Reader{RootPath: rootPath}
}

// KnownProfiles lists recognized btrfs profile directory names under allocation.
var KnownProfiles = []string{
	"raid0", "raid1", "raid1c3", "raid1c4", "raid10", "raid5", "raid6", "dup", "single",
}

// ReadPool reads sysfs telemetry for a given UUID.
func (r *Reader) ReadPool(uuid string) (*PoolSysfsData, error) {
	poolDir := filepath.Join(r.RootPath, uuid)
	info, err := os.Stat(poolDir)
	if err != nil || !info.IsDir() {
		return nil, fmt.Errorf("btrfs pool directory not found for uuid %s: %w", uuid, err)
	}

	label := readFirstLine(filepath.Join(poolDir, "label"))

	dataAlloc := r.readAllocation(filepath.Join(poolDir, "allocation", "data"))
	metaAlloc := r.readAllocation(filepath.Join(poolDir, "allocation", "metadata"))

	devices := r.readDevices(filepath.Join(poolDir, "devices"))
	devInfos := r.readDevInfos(filepath.Join(poolDir, "devinfo"))

	return &PoolSysfsData{
		UUID:      uuid,
		Label:     label,
		DataAlloc: dataAlloc,
		MetaAlloc: metaAlloc,
		Devices:   devices,
		DevInfos:  devInfos,
	}, nil
}

// ListUUIDs discovers all btrfs filesystem UUIDs currently registered in sysfs.
func (r *Reader) ListUUIDs() ([]string, error) {
	entries, err := os.ReadDir(r.RootPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var uuids []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if name == "features" {
			continue
		}
		// Btrfs UUID format typically contains hyphens or is 36 chars.
		if strings.Contains(name, "-") || len(name) >= 32 {
			uuids = append(uuids, name)
		}
	}
	return uuids, nil
}

func (r *Reader) readAllocation(allocDir string) AllocationStats {
	var stats AllocationStats

	stats.TotalBytes = readUint64File(filepath.Join(allocDir, "total_bytes"))
	stats.BytesUsed = readUint64File(filepath.Join(allocDir, "bytes_used"))
	stats.DiskTotal = readUint64File(filepath.Join(allocDir, "disk_total"))
	stats.DiskUsed = readUint64File(filepath.Join(allocDir, "disk_used"))

	for _, prof := range KnownProfiles {
		profDir := filepath.Join(allocDir, prof)
		if fi, err := os.Stat(profDir); err == nil && fi.IsDir() {
			stats.Profile = strings.ToUpper(prof)
			break
		}
	}
	if stats.Profile == "" {
		stats.Profile = "SINGLE"
	}
	return stats
}

func (r *Reader) readDevices(devicesDir string) []string {
	entries, err := os.ReadDir(devicesDir)
	if err != nil {
		return nil
	}
	var devs []string
	for _, entry := range entries {
		devs = append(devs, entry.Name())
	}
	return devs
}

func (r *Reader) readDevInfos(devinfoDir string) map[int]DevInfo {
	res := make(map[int]DevInfo)
	entries, err := os.ReadDir(devinfoDir)
	if err != nil {
		return res
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		devID, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}

		infoPath := filepath.Join(devinfoDir, entry.Name())
		missingVal := readUint64File(filepath.Join(infoPath, "missing"))
		inMetaVal := readUint64File(filepath.Join(infoPath, "in_fs_metadata"))
		writeableVal := readUint64File(filepath.Join(infoPath, "writeable"))
		replaceVal := readUint64File(filepath.Join(infoPath, "replace_target"))

		errStats := r.readErrorStats(filepath.Join(infoPath, "error_stats"))

		res[devID] = DevInfo{
			DevID:         devID,
			Missing:       missingVal > 0,
			InFSMetadata:  inMetaVal > 0,
			Writeable:     writeableVal > 0,
			ReplaceTarget: replaceVal > 0,
			ErrorStats:    errStats,
		}
	}
	return res
}

func (r *Reader) readErrorStats(filePath string) DevErrorStats {
	var stats DevErrorStats
	file, err := os.Open(filePath)
	if err != nil {
		return stats
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		val, _ := strconv.ParseUint(parts[1], 10, 64)
		switch parts[0] {
		case "write_errs":
			stats.WriteErrs = val
		case "read_errs":
			stats.ReadErrs = val
		case "flush_errs":
			stats.FlushErrs = val
		case "corruption_errs":
			stats.CorruptionErrs = val
		case "generation_errs":
			stats.GenerationErrs = val
		}
	}
	return stats
}

func readUint64File(path string) uint64 {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	val, err := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		return 0
	}
	return val
}

func readFirstLine(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	lines := strings.Split(string(data), "\n")
	if len(lines) > 0 {
		return strings.TrimSpace(lines[0])
	}
	return ""
}
