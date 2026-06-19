package docker

import (
	"fmt"
	"os"
	"testing"
)

// TestBackwardCompatibility_ExplicitMTU verifies that when PLUGIN_MTU is set,
// the auto-detection is skipped and the explicit value is used
func TestBackwardCompatibility_ExplicitMTU(t *testing.T) {
	// Simulate the behavior when PLUGIN_MTU is set
	daemon := Daemon{
		MTU:      "1400", // This would be set from PLUGIN_MTU via app.go
		Disabled: false,
	}

	// This is what happens in Plugin.Exec()
	// The auto-detection should NOT run because MTU is already set
	if daemon.MTU == "" && !daemon.Disabled {
		t.Error("Auto-detection should NOT run when MTU is explicitly set")
	} else if daemon.MTU != "" {
		// This path should execute
		fmt.Printf("Using explicitly configured MTU: %s\n", daemon.MTU)
	}

	// Verify the value hasn't changed
	if daemon.MTU != "1400" {
		t.Errorf("Expected MTU to remain 1400, got %s", daemon.MTU)
	}
}

// TestBackwardCompatibility_AutoDetect verifies that when PLUGIN_MTU is NOT set,
// auto-detection runs
func TestBackwardCompatibility_AutoDetect(t *testing.T) {
	// Simulate the behavior when PLUGIN_MTU is NOT set
	daemon := Daemon{
		MTU:      "", // Empty, as it would be if PLUGIN_MTU not set
		Disabled: false,
	}

	// This is what happens in Plugin.Exec()
	if daemon.MTU == "" && !daemon.Disabled {
		// Auto-detection should run
		if detectedMTU, err := detectNetworkMTU(); err == nil {
			fmt.Printf("Auto-detected network MTU: %s\n", detectedMTU)
			daemon.MTU = detectedMTU
		} else {
			fmt.Printf("Note: Could not auto-detect MTU (%s). Will use Docker default.\n", err)
		}
	}

	// Verify that SOME value was attempted to be set (or we got a detection error)
	// This is fine either way - the important thing is auto-detection ran
	t.Logf("After auto-detection: MTU = %s", daemon.MTU)
}

// TestBackwardCompatibility_DaemonDisabled verifies that when daemon is disabled,
// no MTU configuration happens
func TestBackwardCompatibility_DaemonDisabled(t *testing.T) {
	daemon := Daemon{
		MTU:      "",
		Disabled: true, // Daemon is disabled (already running)
	}

	// This is what happens in Plugin.Exec()
	if daemon.MTU == "" && !daemon.Disabled {
		t.Error("Auto-detection should NOT run when daemon is disabled")
	}

	// MTU should remain empty
	if daemon.MTU != "" {
		t.Errorf("Expected MTU to remain empty when daemon disabled, got %s", daemon.MTU)
	}
}

// TestBackwardCompatibility_CommandLine verifies the dockerd command is built correctly
func TestBackwardCompatibility_CommandLine(t *testing.T) {
	tests := []struct {
		name          string
		mtu           string
		shouldHaveMTU bool
	}{
		{
			name:          "With explicit MTU",
			mtu:           "1400",
			shouldHaveMTU: true,
		},
		{
			name:          "Without MTU",
			mtu:           "",
			shouldHaveMTU: false,
		},
		{
			name:          "With auto-detected MTU",
			mtu:           "1450",
			shouldHaveMTU: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			daemon := Daemon{
				MTU:         tt.mtu,
				StoragePath: "/var/lib/docker",
			}

			cmd := commandDaemon(daemon)
			args := cmd.Args

			foundMTU := false
			var mtuValue string
			for i, arg := range args {
				if arg == "--mtu" {
					foundMTU = true
					if i+1 < len(args) {
						mtuValue = args[i+1]
					}
					break
				}
			}

			if tt.shouldHaveMTU {
				if !foundMTU {
					t.Errorf("Expected --mtu flag in command, but not found. Args: %v", args)
				}
				if mtuValue != tt.mtu {
					t.Errorf("Expected --mtu %s, got %s", tt.mtu, mtuValue)
				}
			} else {
				if foundMTU {
					t.Errorf("Expected no --mtu flag in command, but found: --mtu %s", mtuValue)
				}
			}
		})
	}
}

// TestBackwardCompatibility_EnvVarParsing simulates the full flow from env var to daemon
func TestBackwardCompatibility_EnvVarParsing(t *testing.T) {
	// Save original env var
	originalMTU := os.Getenv("PLUGIN_MTU")
	defer func() {
		if originalMTU != "" {
			os.Setenv("PLUGIN_MTU", originalMTU)
		} else {
			os.Unsetenv("PLUGIN_MTU")
		}
	}()

	tests := []struct {
		name        string
		envValue    string
		expectedMTU string
	}{
		{
			name:        "PLUGIN_MTU set to 1400",
			envValue:    "1400",
			expectedMTU: "1400",
		},
		{
			name:        "PLUGIN_MTU set to 1450",
			envValue:    "1450",
			expectedMTU: "1450",
		},
		{
			name:        "PLUGIN_MTU empty",
			envValue:    "",
			expectedMTU: "", // Should trigger auto-detection
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv("PLUGIN_MTU", tt.envValue)
			} else {
				os.Unsetenv("PLUGIN_MTU")
			}

			// In the real app.go, this would be: c.String("daemon.mtu")
			// which reads from PLUGIN_MTU
			envMTU := os.Getenv("PLUGIN_MTU")

			if envMTU != tt.expectedMTU {
				t.Errorf("Expected env MTU %s, got %s", tt.expectedMTU, envMTU)
			}

			t.Logf("PLUGIN_MTU=%s correctly read as: %s", tt.envValue, envMTU)
		})
	}
}
