package btrfs

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ScrubStatus represents the current state and progress of a btrfs scrub.
type ScrubStatus struct {
	Active          bool    `json:"active"`
	StatusText      string  `json:"status_text"`
	TotalBytes      uint64  `json:"total_bytes"`
	ScrubbedBytes   uint64  `json:"scrubbed_bytes"`
	ProgressPercent float64 `json:"progress_percent"`
	RateBytesPerSec uint64  `json:"rate_bytes_per_sec"`
	Errors          uint64  `json:"errors"`
	DurationSeconds uint64  `json:"duration_seconds"`
}

// BalanceStatus represents the current state and progress of a btrfs balance.
type BalanceStatus struct {
	Active          bool    `json:"active"`
	StatusText      string  `json:"status_text"`
	ProgressPercent float64 `json:"progress_percent"`
	TotalChunks     int     `json:"total_chunks"`
	BalancedChunks  int     `json:"balanced_chunks"`
}

// Client defines methods to query and control btrfs operations.
type Client interface {
	GetScrubStatus(ctx context.Context, target string) (*ScrubStatus, error)
	GetBalanceStatus(ctx context.Context, target string) (*BalanceStatus, error)
	DeviceAdd(ctx context.Context, device, mountpoint string) error
	DeviceRemove(ctx context.Context, device, mountpoint string) error
	DeviceReplace(ctx context.Context, oldDev, newDev, mountpoint string) error
}

// RealClient executes real btrfs binaries via os/exec.
type RealClient struct {
	BtrfsPath string
}

// NewClient returns a new Client pointing to the btrfs executable.
func NewClient(binPath string) *RealClient {
	if binPath == "" {
		binPath = "btrfs"
	}
	return &RealClient{BtrfsPath: binPath}
}

// GetScrubStatus queries scrub progress on a mountpoint or device.
func (c *RealClient) GetScrubStatus(ctx context.Context, target string) (*ScrubStatus, error) {
	// First attempt JSON if supported by btrfs-progs version
	cmdJSON := exec.CommandContext(ctx, c.BtrfsPath, "--format", "json", "scrub", "status", target)
	outJSON, errJSON := cmdJSON.Output()
	if errJSON == nil && len(outJSON) > 0 {
		if status, err := parseScrubJSON(outJSON); err == nil {
			return status, nil
		}
	}

	// Fallback to raw text output
	cmdRaw := exec.CommandContext(ctx, c.BtrfsPath, "scrub", "status", "--raw", target)
	outRaw, err := cmdRaw.CombinedOutput()
	if err != nil {
		// If scrub has never been run or target is invalid
		if strings.Contains(string(outRaw), "no stats available") {
			return &ScrubStatus{
				Active:     false,
				StatusText: "idle",
			}, nil
		}
		return nil, fmt.Errorf("scrub status failed: %w (output: %s)", err, strings.TrimSpace(string(outRaw)))
	}

	return ParseScrubRaw(string(outRaw)), nil
}

// GetBalanceStatus queries balance progress on a mountpoint.
func (c *RealClient) GetBalanceStatus(ctx context.Context, target string) (*BalanceStatus, error) {
	cmdJSON := exec.CommandContext(ctx, c.BtrfsPath, "--format", "json", "balance", "status", target)
	outJSON, errJSON := cmdJSON.Output()
	if errJSON == nil && len(outJSON) > 0 {
		if status, err := parseBalanceJSON(outJSON); err == nil {
			return status, nil
		}
	}

	cmdRaw := exec.CommandContext(ctx, c.BtrfsPath, "balance", "status", target)
	outRaw, _ := cmdRaw.CombinedOutput()
	return ParseBalanceRaw(string(outRaw)), nil
}

// DeviceAdd adds a new block device to a mounted btrfs pool.
func (c *RealClient) DeviceAdd(ctx context.Context, device, mountpoint string) error {
	cmd := exec.CommandContext(ctx, c.BtrfsPath, "device", "add", "-f", device, mountpoint)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("btrfs device add failed: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// DeviceRemove removes a block device from a mounted btrfs pool.
func (c *RealClient) DeviceRemove(ctx context.Context, device, mountpoint string) error {
	cmd := exec.CommandContext(ctx, c.BtrfsPath, "device", "remove", device, mountpoint)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("btrfs device remove failed: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// DeviceReplace initiates replacing an existing device with a new one.
func (c *RealClient) DeviceReplace(ctx context.Context, oldDev, newDev, mountpoint string) error {
	cmd := exec.CommandContext(ctx, c.BtrfsPath, "replace", "start", "-B", "-f", oldDev, newDev, mountpoint)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("btrfs device replace failed: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// ParseScrubRaw parses btrfs scrub status --raw output.
func ParseScrubRaw(output string) *ScrubStatus {
	status := &ScrubStatus{
		StatusText: "idle",
	}

	if strings.Contains(output, "no stats available") {
		return status
	}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		parts := strings.SplitN(line, ":", 2)
		if len(parts) < 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		switch key {
		case "Status":
			status.StatusText = strings.ToLower(val)
			if strings.Contains(status.StatusText, "running") {
				status.Active = true
			}
		case "Total to scrub":
			status.TotalBytes, _ = strconv.ParseUint(val, 10, 64)
		case "Bytes scrubbed":
			status.ScrubbedBytes, _ = strconv.ParseUint(val, 10, 64)
		case "Rate":
			rateVal := strings.TrimSuffix(val, "/s")
			status.RateBytesPerSec, _ = strconv.ParseUint(strings.TrimSpace(rateVal), 10, 64)
		case "Duration":
			status.DurationSeconds = parseDuration(val)
		case "Error summary":
			if !strings.Contains(val, "no errors found") {
				// Parse numbers if present
				re := regexp.MustCompile(`\d+`)
				matches := re.FindAllString(val, -1)
				var errCount uint64
				for _, m := range matches {
					n, _ := strconv.ParseUint(m, 10, 64)
					errCount += n
				}
				status.Errors = errCount
			}
		}
	}

	if status.TotalBytes > 0 && status.ScrubbedBytes > 0 {
		pct := (float64(status.ScrubbedBytes) / float64(status.TotalBytes)) * 100.0
		if pct > 100.0 {
			pct = 100.0
		}
		status.ProgressPercent = pct
	}

	return status
}

// ParseBalanceRaw parses btrfs balance status output.
func ParseBalanceRaw(output string) *BalanceStatus {
	status := &BalanceStatus{
		StatusText: "idle",
	}

	if strings.Contains(output, "No balance found") || strings.TrimSpace(output) == "" {
		return status
	}

	if strings.Contains(output, "is running") || strings.Contains(output, "running") {
		status.Active = true
		status.StatusText = "running"
	} else if strings.Contains(output, "paused") {
		status.Active = false
		status.StatusText = "paused"
	}

	// Example: "2 out of 10 chunks balanced (20% considered), 80% left"
	re := regexp.MustCompile(`(\d+)\s+out of\s+(\d+)\s+chunks balanced.*?\((\d+)%`)
	matches := re.FindStringSubmatch(output)
	if len(matches) >= 4 {
		status.BalancedChunks, _ = strconv.Atoi(matches[1])
		status.TotalChunks, _ = strconv.Atoi(matches[2])
		pct, _ := strconv.ParseFloat(matches[3], 64)
		status.ProgressPercent = pct
	}

	return status
}

func parseScrubJSON(data []byte) (*ScrubStatus, error) {
	var payload struct {
		ScrubStatus struct {
			Status        string `json:"status"`
			BytesScrubbed uint64 `json:"data_bytes_scrubbed"`
			TreeScrubbed  uint64 `json:"tree_bytes_scrubbed"`
			TotalBytes    uint64 `json:"total_bytes"`
			Errors        uint64 `json:"read_errors"`
		} `json:"scrub-status"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	st := &ScrubStatus{
		StatusText:    payload.ScrubStatus.Status,
		Active:        strings.EqualFold(payload.ScrubStatus.Status, "running"),
		TotalBytes:    payload.ScrubStatus.TotalBytes,
		ScrubbedBytes: payload.ScrubStatus.BytesScrubbed + payload.ScrubStatus.TreeScrubbed,
		Errors:        payload.ScrubStatus.Errors,
	}
	if st.TotalBytes > 0 && st.ScrubbedBytes > 0 {
		st.ProgressPercent = (float64(st.ScrubbedBytes) / float64(st.TotalBytes)) * 100.0
	}
	return st, nil
}

func parseBalanceJSON(data []byte) (*BalanceStatus, error) {
	var payload struct {
		BalanceStatus struct {
			Status    string  `json:"status"`
			Completed int     `json:"completed"`
			Total     int     `json:"total"`
			Percent   float64 `json:"percent"`
		} `json:"balance-status"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	return &BalanceStatus{
		Active:          strings.EqualFold(payload.BalanceStatus.Status, "running"),
		StatusText:      payload.BalanceStatus.Status,
		ProgressPercent: payload.BalanceStatus.Percent,
		BalancedChunks:  payload.BalanceStatus.Completed,
		TotalChunks:     payload.BalanceStatus.Total,
	}, nil
}

func parseDuration(val string) uint64 {
	// e.g. 0:01:23
	parts := strings.Split(val, ":")
	if len(parts) == 3 {
		h, _ := strconv.ParseUint(parts[0], 10, 64)
		m, _ := strconv.ParseUint(parts[1], 10, 64)
		s, _ := strconv.ParseUint(parts[2], 10, 64)
		return h*3600 + m*60 + s
	}
	d, err := time.ParseDuration(val)
	if err == nil {
		return uint64(d.Seconds())
	}
	return 0
}
