package nattraversal

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"strings"
)

// createTCPMapping establishes a TCP port mapping.
// Moved from: listener.go
func createTCPMapping(port int) (PortMapper, int, error) {
	mapper, err := NewPortMapper()
	if err != nil {
		return nil, 0, err
	}

	externalPort, err := mapper.MapPort("TCP", port, mappingDuration)
	if err != nil {
		return nil, 0, err
	}

	return mapper, externalPort, nil
}

// createUDPMapping establishes a UDP port mapping.
// Moved from: packetlistener.go
func createUDPMapping(port int) (PortMapper, int, error) {
	mapper, err := NewPortMapper()
	if err != nil {
		return nil, 0, err
	}

	externalPort, err := mapper.MapPort("UDP", port, mappingDuration)
	if err != nil {
		return nil, 0, err
	}

	return mapper, externalPort, nil
}

// discoverGateway finds the default gateway for NAT-PMP.
// It reads the system routing table on Linux (/proc/net/route) to find
// the actual default gateway. Falls back to assuming .1 suffix if the
// routing table cannot be read.
func discoverGateway() (net.IP, error) {
	// Try to read the actual gateway from the routing table
	gateway, err := readDefaultGateway()
	if err == nil && gateway != nil {
		return gateway, nil
	}

	// Fallback: assume gateway is .1 in the same subnet as local IP
	return discoverGatewayFallback()
}

// readDefaultGateway reads the default gateway from /proc/net/route on Linux.
// Returns nil, nil if the file doesn't exist (non-Linux systems).
// Returns nil, error if the file exists but cannot be parsed.
func readDefaultGateway() (net.IP, error) {
	file, err := os.Open("/proc/net/route")
	if err != nil {
		// File doesn't exist (non-Linux) - not an error, use fallback
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

// discoverGatewayFallback uses the old heuristic of assuming .1 gateway.
// This is used when the routing table cannot be read (non-Linux systems).
func discoverGatewayFallback() (net.IP, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	ip := localAddr.IP.To4()
	if ip == nil {
		return nil, fmt.Errorf("not IPv4 address")
	}

	// Assume gateway is .1 in the same subnet (common convention)
	gateway := net.IPv4(ip[0], ip[1], ip[2], 1)
	return gateway, nil
}
