//go:build linux

package nattraversal

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"strings"
)

// readDefaultGateway reads the default gateway from /proc/net/route on Linux.
// Returns nil, nil if the file doesn't exist or no default route is found.
// Returns nil, error if the file exists but cannot be parsed.
func readDefaultGateway() (net.IP, error) {
	file, err := os.Open("/proc/net/route")
	if err != nil {
		// File doesn't exist - not an error, use fallback
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to open routing table: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	// Skip header line
	if !scanner.Scan() {
		return nil, fmt.Errorf("empty routing table")
	}

	// Find the default route (Destination == 00000000)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 3 {
			continue
		}

		destination := fields[1]
		gatewayHex := fields[2]

		// Default route has destination 00000000
		if destination == "00000000" {
			gateway, err := parseHexIP(gatewayHex)
			if err != nil {
				return nil, fmt.Errorf("failed to parse gateway: %w", err)
			}
			// Skip if gateway is 0.0.0.0 (local route)
			if !gateway.Equal(net.IPv4zero) {
				return gateway, nil
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading routing table: %w", err)
	}

	return nil, nil // No default gateway found, use fallback
}

// parseHexIP converts a hex-encoded IP address from /proc/net/route to net.IP.
// The format is little-endian hex (e.g., "0101A8C0" = 192.168.1.1).
func parseHexIP(hexIP string) (net.IP, error) {
	if len(hexIP) != 8 {
		return nil, fmt.Errorf("invalid hex IP length: %d", len(hexIP))
	}

	bytes, err := hex.DecodeString(hexIP)
	if err != nil {
		return nil, fmt.Errorf("invalid hex IP: %w", err)
	}

	// Reverse bytes (little-endian to big-endian)
	return net.IPv4(bytes[3], bytes[2], bytes[1], bytes[0]), nil
}
