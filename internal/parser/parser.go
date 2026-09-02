package parser

import (
	"bytes"
	"context"
	"encoding/json"
	"time"

	"github.com/franelfers/omarchy-btrfs-raid-manager/internal/btrfs"
	"github.com/franelfers/omarchy-btrfs-raid-manager/internal/sysfs"
	"github.com/franelfers/omarchy-btrfs-raid-manager/internal/systemd"
	"github.com/franelfers/omarchy-btrfs-raid-manager/internal/udisks"
)

// PoolDevice represents a disk member within a Btrfs pool.
type PoolDevice struct {
	DevNode           string  `json:"dev_node"`
	DevID             int     `json:"dev_id"`
	Missing           bool    `json:"missing"`
	Model             string  `json:"model"`
	Serial            string  `json:"serial"`
	SizeBytes         uint64  `json:"size_bytes"`
	SmartStatus       string  `json:"smart_status"` // "passed", "failing", "warning", "unknown"
	SmartTemperatureC float64 `json:"smart_temperature_c"`
	SmartBadSectors   int64   `json:"smart_bad_sectors"`
	WriteErrs         uint64  `json:"write_errs"`
	ReadErrs          uint64  `json:"read_errs"`
	CorruptionErrs    uint64  `json:"corruption_errs"`
}

// PoolScrub aggregates scrub execution and timer information.
type PoolScrub struct {
	Active          bool    `json:"active"`
	ProgressPercent float64 `json:"progress_percent"`
	StatusText      string  `json:"status_text"`
	ScrubbedBytes   uint64  `json:"scrubbed_bytes"`
	TotalBytes      uint64  `json:"total_bytes"`
	Errors          uint64  `json:"errors"`
	TimerEnabled    bool    `json:"timer_enabled"`
	TimerActive     bool    `json:"timer_active"`
}

// PoolBalance aggregates balance execution and timer information.
type PoolBalance struct {
	Active          bool    `json:"active"`
	ProgressPercent float64 `json:"progress_percent"`
	StatusText      string  `json:"status_text"`
	TimerEnabled    bool    `json:"timer_enabled"`
	TimerActive     bool    `json:"timer_active"`
}

// PoolInfo contains the unified telemetry and status for a single Btrfs RAID pool.
type PoolInfo struct {
	UUID              string       `json:"uuid"`
	Label             string       `json:"label"`
	Mountpoint        string       `json:"mountpoint"`
	IsMounted         bool         `json:"is_mounted"`
	TotalBytes        uint64       `json:"total_bytes"`
	UsedBytes         uint64       `json:"used_bytes"`
	FreeBytes         uint64       `json:"free_bytes"`
	PercentUsed       float64      `json:"percent_used"`
	RaidProfile       string       `json:"raid_profile"`
	MetaProfile       string       `json:"meta_profile"`
	Status            string       `json:"status"` // "healthy", "degraded", "working"
	IsDegraded        bool         `json:"is_degraded"`
	ActiveOperation   string       `json:"active_operation"` // "none", "scrub", "balance"
	OperationProgress float64      `json:"operation_progress"`
	Devices           []PoolDevice `json:"devices"`
	Scrub             PoolScrub    `json:"scrub"`
	Balance           PoolBalance  `json:"balance"`
}

// State represents the complete root object emitted via NDJSON.
type State struct {
	Pools     []PoolInfo `json:"pools"`
	Timestamp int64      `json:"timestamp"`
}

// MarshalNDJSON serializes a State struct into a single-line JSON byte array with a newline terminator.
func MarshalNDJSON(state *State) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(state); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// AggregatePool builds a unified PoolInfo from disparate subsystem inputs.
func AggregatePool(
	sysData *sysfs.PoolSysfsData,
	devices map[string]udisks.DeviceInfo,
	scrubStatus *btrfs.ScrubStatus,
	balanceStatus *btrfs.BalanceStatus,
	scrubServiceStatus *systemd.UnitStatus,
	scrubTimerStatus *systemd.UnitStatus,
	balanceServiceStatus *systemd.UnitStatus,
	balanceTimerStatus *systemd.UnitStatus,
) PoolInfo {
	info := PoolInfo{
		UUID:        sysData.UUID,
		Label:       sysData.Label,
		RaidProfile: sysData.DataAlloc.Profile,
		MetaProfile: sysData.MetaAlloc.Profile,
		Status:      "healthy",
		Scrub: PoolScrub{
			StatusText: "idle",
		},
		Balance: PoolBalance{
			StatusText: "idle",
		},
	}

	if info.Label == "" {
		info.Label = "btrfs-" + sysData.UUID[:8]
	}

	// Calculate storage usage
	info.TotalBytes = sysData.DataAlloc.TotalBytes
	info.UsedBytes = sysData.DataAlloc.BytesUsed
	if info.TotalBytes > info.UsedBytes {
		info.FreeBytes = info.TotalBytes - info.UsedBytes
	}
	if info.TotalBytes > 0 {
		info.PercentUsed = (float64(info.UsedBytes) / float64(info.TotalBytes)) * 100.0
	}

	// Match devices
	var hasMissing bool
	var hasFailing bool

	// Build map of devices by node/name
	for devID, dInfo := range sysData.DevInfos {
		pDev := PoolDevice{
			DevID:          devID,
			Missing:        dInfo.Missing,
			WriteErrs:      dInfo.ErrorStats.WriteErrs,
			ReadErrs:       dInfo.ErrorStats.ReadErrs,
			CorruptionErrs: dInfo.ErrorStats.CorruptionErrs,
			SmartStatus:    "unknown",
		}

		if dInfo.Missing {
			pDev.DevNode = "MISSING"
			pDev.Model = "Missing Device"
			hasMissing = true
		} else {
			// Find matching block device name from sysData.Devices
			for _, devName := range sysData.Devices {
				if uInfo, ok := devices[devName]; ok {
					pDev.DevNode = uInfo.DevNode
					pDev.Model = uInfo.Model
					pDev.Serial = uInfo.Serial
					pDev.SizeBytes = uInfo.SizeBytes
					pDev.SmartStatus = uInfo.SmartStatus
					pDev.SmartTemperatureC = uInfo.SmartTemperatureC
					pDev.SmartBadSectors = uInfo.SmartBadSectors

					if len(uInfo.MountPoints) > 0 && info.Mountpoint == "" {
						info.Mountpoint = uInfo.MountPoints[0]
						info.IsMounted = true
					}
					if uInfo.SmartFailing {
						hasFailing = true
					}
					break
				}
			}
			if pDev.DevNode == "" && len(sysData.Devices) > 0 {
				pDev.DevNode = "/dev/" + sysData.Devices[0]
			}
		}

		info.Devices = append(info.Devices, pDev)
	}

	// If no mountpoint discovered from device mountpoints, check any device in pool
	if !info.IsMounted {
		for _, devName := range sysData.Devices {
			if uInfo, ok := devices[devName]; ok && len(uInfo.MountPoints) > 0 {
				info.Mountpoint = uInfo.MountPoints[0]
				info.IsMounted = true
				break
			}
		}
	}

	// Incorporate Scrub telemetry
	if scrubStatus != nil {
		info.Scrub.Active = scrubStatus.Active
		info.Scrub.StatusText = scrubStatus.StatusText
		info.Scrub.TotalBytes = scrubStatus.TotalBytes
		info.Scrub.ScrubbedBytes = scrubStatus.ScrubbedBytes
		info.Scrub.ProgressPercent = scrubStatus.ProgressPercent
		info.Scrub.Errors = scrubStatus.Errors
	}
	if scrubServiceStatus != nil && scrubServiceStatus.IsActive {
		info.Scrub.Active = true
		if info.Scrub.StatusText == "idle" {
			info.Scrub.StatusText = "running"
		}
	}
	if scrubTimerStatus != nil {
		info.Scrub.TimerEnabled = scrubTimerStatus.IsEnabled
		info.Scrub.TimerActive = scrubTimerStatus.IsActive
	}

	// Incorporate Balance telemetry
	if balanceStatus != nil {
		info.Balance.Active = balanceStatus.Active
		info.Balance.StatusText = balanceStatus.StatusText
		info.Balance.ProgressPercent = balanceStatus.ProgressPercent
	}
	if balanceServiceStatus != nil && balanceServiceStatus.IsActive {
		info.Balance.Active = true
		if info.Balance.StatusText == "idle" {
			info.Balance.StatusText = "running"
		}
	}
	if balanceTimerStatus != nil {
		info.Balance.TimerEnabled = balanceTimerStatus.IsEnabled
		info.Balance.TimerActive = balanceTimerStatus.IsActive
	}

	// Determine overall operational status
	if hasMissing || hasFailing {
		info.Status = "degraded"
		info.IsDegraded = true
	} else if info.Scrub.Active {
		info.Status = "working"
		info.ActiveOperation = "scrub"
		info.OperationProgress = info.Scrub.ProgressPercent
	} else if info.Balance.Active {
		info.Status = "working"
		info.ActiveOperation = "balance"
		info.OperationProgress = info.Balance.ProgressPercent
	} else {
		info.Status = "healthy"
		info.ActiveOperation = "none"
	}

	return info
}

// BuildState collects and aggregates all active pools into a single State object.
func BuildState(
	ctx context.Context,
	sysReader *sysfs.Reader,
	uClient *udisks.Client,
	sysClient *systemd.Client,
	btrClient btrfs.Client,
) (*State, error) {
	uuids, err := sysReader.ListUUIDs()
	if err != nil {
		return nil, err
	}

	var blockDevices map[string]udisks.DeviceInfo
	if uClient != nil {
		blockDevices, _ = uClient.InspectBlockDevices(ctx)
	}

	state := &State{
		Pools:     make([]PoolInfo, 0, len(uuids)),
		Timestamp: time.Now().Unix(),
	}

	for _, uuid := range uuids {
		sysData, err := sysReader.ReadPool(uuid)
		if err != nil {
			continue
		}

		var scrubStatus *btrfs.ScrubStatus
		var balanceStatus *btrfs.BalanceStatus
		var scrubService *systemd.UnitStatus
		var scrubTimer *systemd.UnitStatus
		var balanceService *systemd.UnitStatus
		var balanceTimer *systemd.UnitStatus

		// Discover mountpoint if possible
		var mountpoint string
		for _, dev := range sysData.Devices {
			if uInfo, ok := blockDevices[dev]; ok && len(uInfo.MountPoints) > 0 {
				mountpoint = uInfo.MountPoints[0]
				break
			}
		}

		if mountpoint != "" {
			if btrClient != nil {
				scrubStatus, _ = btrClient.GetScrubStatus(ctx, mountpoint)
				balanceStatus, _ = btrClient.GetBalanceStatus(ctx, mountpoint)
			}
			if sysClient != nil {
				scrubService, _ = sysClient.GetUnitStatus(ctx, systemd.ScrubServiceName(mountpoint))
				scrubTimer, _ = sysClient.GetUnitStatus(ctx, systemd.ScrubTimerName(mountpoint))
				balanceService, _ = sysClient.GetUnitStatus(ctx, systemd.BalanceServiceName(mountpoint))
				balanceTimer, _ = sysClient.GetUnitStatus(ctx, systemd.BalanceTimerName(mountpoint))
			}
		}

		poolInfo := AggregatePool(
			sysData,
			blockDevices,
			scrubStatus,
			balanceStatus,
			scrubService,
			scrubTimer,
			balanceService,
			balanceTimer,
		)
		state.Pools = append(state.Pools, poolInfo)
	}

	return state, nil
}
