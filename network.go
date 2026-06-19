package docker

import (
	"fmt"
	"net"
)

// detectNetworkMTU detects the MTU of the default network interface
// Returns the MTU value as a string, or an error if detection fails
// Uses Go standard library for cross-platform compatibility (Linux, macOS, Windows)
func detectNetworkMTU() (string, error) {
	// Determine the default interface by dialing out to a known external IP
	// This doesn't actually send data, just determines routing
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", fmt.Errorf("failed to determine default network interface: %w", err)
	}
	defer conn.Close()

	// Get the local address used for this connection
	localAddr := conn.LocalAddr().(*net.UDPAddr)

	// Get all network interfaces
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", fmt.Errorf("failed to get network interfaces: %w", err)
	}

	// Find the interface that has the local address
	for _, iface := range interfaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if ok && ipNet.IP.Equal(localAddr.IP) {
				// Found the interface - return its MTU
				if iface.MTU <= 0 {
					return "", fmt.Errorf("invalid MTU value: %d", iface.MTU)
				}
				return fmt.Sprintf("%d", iface.MTU), nil
			}
		}
	}

	return "", fmt.Errorf("could not determine MTU for default network interface")
}

// getEffectiveMTU returns the MTU value to use for BuildKit containers
// Priority: daemon.MTU (explicit) > auto-detected > fallback to empty (docker default)
func getEffectiveMTU(daemonMTU string) string {
	// If explicitly set via PLUGIN_MTU, use that
	if daemonMTU != "" {
		fmt.Printf("Using explicitly configured MTU: %s\n", daemonMTU)
		return daemonMTU
	}

	// Try to auto-detect
	detectedMTU, err := detectNetworkMTU()
	if err != nil {
		fmt.Printf("Warning: Could not auto-detect MTU (%s), BuildKit will use Docker default\n", err)
		return ""
	}

	fmt.Printf("Auto-detected network MTU: %s\n", detectedMTU)
	return detectedMTU
}
