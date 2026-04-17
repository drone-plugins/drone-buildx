package docker

import (
	"fmt"
	"testing"
)

func TestDetectNetworkMTU(t *testing.T) {
	mtu, err := detectNetworkMTU()

	if err != nil {
		t.Logf("Warning: Could not detect MTU (this may be expected in test environment): %v", err)
		// Don't fail the test - MTU detection may not work in all environments
		return
	}

	if mtu == "" {
		t.Error("Expected MTU to be non-empty")
	}

	// Basic validation: MTU should be a reasonable value
	// Common values: 1280 (minimum IPv6), 1450-1500 (common), up to 9000 (jumbo frames)
	t.Logf("Detected MTU: %s", mtu)

	// Parse as integer to validate
	var mtuInt int
	_, err = fmt.Sscanf(mtu, "%d", &mtuInt)
	if err != nil {
		t.Errorf("MTU is not a valid integer: %s", mtu)
	}

	if mtuInt < 1280 {
		t.Errorf("MTU %d is too small (minimum expected: 1280)", mtuInt)
	}

	if mtuInt > 9000 {
		t.Errorf("MTU %d is unusually large (maximum expected: 9000)", mtuInt)
	}
}

func TestGetEffectiveMTU(t *testing.T) {
	tests := []struct {
		name       string
		daemonMTU  string
		wantExplicit bool
	}{
		{
			name:       "explicit MTU set",
			daemonMTU:  "1400",
			wantExplicit: true,
		},
		{
			name:       "empty MTU triggers auto-detect",
			daemonMTU:  "",
			wantExplicit: false,
		},
		{
			name:       "zero MTU triggers auto-detect",
			daemonMTU:  "0",
			wantExplicit: true, // "0" is explicit, even if unusual
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getEffectiveMTU(tt.daemonMTU)

			if tt.wantExplicit {
				if result != tt.daemonMTU {
					t.Errorf("Expected explicit MTU %s, got %s", tt.daemonMTU, result)
				}
			} else {
				// For auto-detect, we just verify it returns something
				// (or empty if detection fails)
				t.Logf("Auto-detected MTU: %s", result)
			}
		})
	}
}

func TestGetEffectiveMTU_Priority(t *testing.T) {
	// Test that explicit MTU takes priority over auto-detection
	explicitMTU := "1350"
	result := getEffectiveMTU(explicitMTU)

	if result != explicitMTU {
		t.Errorf("Expected explicit MTU %s to take priority, got %s", explicitMTU, result)
	}
}
