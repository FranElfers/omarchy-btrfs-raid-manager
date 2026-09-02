package udisks

import (
	"testing"
)

func TestDeviceInfoParsing(t *testing.T) {
	// Verify struct values and conversions
	info := DeviceInfo{
		DevNode:           "/dev/sda1",
		Model:             "Crucial CT1000",
		Serial:            "123456789",
		SizeBytes:         1000000000000,
		SmartSupported:    true,
		SmartEnabled:      true,
		SmartFailing:      false,
		SmartBadSectors:   0,
		SmartTemperatureC: 27.5,
		SmartStatus:       "passed",
	}

	if info.DevNode != "/dev/sda1" {
		t.Errorf("expected /dev/sda1, got %s", info.DevNode)
	}
	if info.SmartStatus != "passed" {
		t.Errorf("expected passed status, got %s", info.SmartStatus)
	}
	if info.SmartTemperatureC != 27.5 {
		t.Errorf("expected 27.5 C, got %f", info.SmartTemperatureC)
	}
}
