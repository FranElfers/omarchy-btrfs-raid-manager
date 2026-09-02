package systemd

import (
	"testing"
)

func TestEscapePath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/mnt/datos", "mnt-datos"},
		{"/mnt/storage/pool1", "mnt-storage-pool1"},
		{"/", "-"},
		{"mnt/data", "mnt-data"},
	}

	for _, tt := range tests {
		got := EscapePath(tt.input)
		if got != tt.expected {
			t.Errorf("EscapePath(%q) = %q; expected %q", tt.input, got, tt.expected)
		}
	}
}

func TestUnitNames(t *testing.T) {
	mp := "/mnt/datos"
	if s := ScrubServiceName(mp); s != "btrpool-scrub@mnt-datos.service" {
		t.Errorf("expected btrpool-scrub@mnt-datos.service, got %s", s)
	}
	if tm := ScrubTimerName(mp); tm != "btrpool-scrub@mnt-datos.timer" {
		t.Errorf("expected btrpool-scrub@mnt-datos.timer, got %s", tm)
	}
	if b := BalanceServiceName(mp); b != "btrpool-balance@mnt-datos.service" {
		t.Errorf("expected btrpool-balance@mnt-datos.service, got %s", b)
	}
	if btm := BalanceTimerName(mp); btm != "btrpool-balance@mnt-datos.timer" {
		t.Errorf("expected btrpool-balance@mnt-datos.timer, got %s", btm)
	}
}
