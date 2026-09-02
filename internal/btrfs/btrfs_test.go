package btrfs

import (
	"testing"
)

func TestParseScrubRaw(t *testing.T) {
	raw := `UUID:             b295f839-ce80-4f55-8ce2-5798fe2d816c
Scrub started:    Wed Sep  2 20:00:00 2026
Status:           running
Duration:         0:02:30
Total to scrub:   1000000000
Bytes scrubbed:   420000000
Rate:             2800000/s
Error summary:    read_errors: 1  verify_errors: 2`

	st := ParseScrubRaw(raw)
	if !st.Active {
		t.Errorf("expected active to be true")
	}
	if st.StatusText != "running" {
		t.Errorf("expected status running, got %s", st.StatusText)
	}
	if st.TotalBytes != 1000000000 {
		t.Errorf("expected TotalBytes 1000000000, got %d", st.TotalBytes)
	}
	if st.ScrubbedBytes != 420000000 {
		t.Errorf("expected ScrubbedBytes 420000000, got %d", st.ScrubbedBytes)
	}
	if st.ProgressPercent < 41.9 || st.ProgressPercent > 42.1 {
		t.Errorf("expected ProgressPercent ~42%%, got %f", st.ProgressPercent)
	}
	if st.RateBytesPerSec != 2800000 {
		t.Errorf("expected RateBytesPerSec 2800000, got %d", st.RateBytesPerSec)
	}
	if st.Errors != 3 {
		t.Errorf("expected Errors 3 (1+2), got %d", st.Errors)
	}
	if st.DurationSeconds != 150 {
		t.Errorf("expected DurationSeconds 150, got %d", st.DurationSeconds)
	}
}

func TestParseScrubRawNoStats(t *testing.T) {
	raw := `UUID:             b295f839-ce80-4f55-8ce2-5798fe2d816c
	no stats available
Total to scrub:   0
Rate:             0/s
Error summary:    no errors found`

	st := ParseScrubRaw(raw)
	if st.Active {
		t.Errorf("expected active to be false")
	}
	if st.StatusText != "idle" {
		t.Errorf("expected idle status, got %s", st.StatusText)
	}
	if st.Errors != 0 {
		t.Errorf("expected 0 errors, got %d", st.Errors)
	}
}

func TestParseBalanceRaw(t *testing.T) {
	raw := `Balance on '/mnt/datos' is running
2 out of 10 chunks balanced (20% considered), 80% left`

	st := ParseBalanceRaw(raw)
	if !st.Active {
		t.Errorf("expected active to be true")
	}
	if st.StatusText != "running" {
		t.Errorf("expected running status, got %s", st.StatusText)
	}
	if st.BalancedChunks != 2 || st.TotalChunks != 10 {
		t.Errorf("expected 2/10 chunks, got %d/%d", st.BalancedChunks, st.TotalChunks)
	}
	if st.ProgressPercent != 20.0 {
		t.Errorf("expected ProgressPercent 20%%, got %f", st.ProgressPercent)
	}
}
