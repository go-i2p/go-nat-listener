package nattraversal

import (
	"fmt"
	"net"
)

// discoverGateway finds the default gateway for NAT-PMP.
// It uses platform-specific methods to read the system routing table,
// falling back to a heuristic if the routing table cannot be read.
func discoverGateway() (net.IP, error) {
	// Try to read the actual gateway from the routing table (platform-specific)
	gateway, err := readDefaultGateway()
	if err == nil && gateway != nil {
		return gateway, nil
	}

	// Fallback: assume gateway is .1 in the same subnet as local IP
	return discoverGatewayFallback()
}

// discoverGatewayFallback uses the heuristic of assuming .1 gateway.
// This is used when platform-specific gateway detection fails or is unavailable.
// The heuristic works by:
// 1. Opening a UDP "connection" to a known external IP (no actual packets sent)
// 2. Determining which local IP would be used for that route
// 3. Assuming the gateway is at .1 in that subnet
//
// This works for most home/office networks where the router is at x.x.x.1
func discoverGatewayFallback() (net.IP, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return nil, fmt.Errorf("failed to determine local IP: %w", err)
	}
	defer conn.Close()

	// Use safe type assertion to prevent potential panic
	localAddr, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok {
		return nil, fmt.Errorf("unexpected local address type: %T", conn.LocalAddr())
	}
	ip := localAddr.IP.To4()
	if ip == nil {
		return nil, fmt.Errorf("not IPv4 address")
	}

	// Assume gateway is .1 in the same subnet (common convention)
	gateway := net.IPv4(ip[0], ip[1], ip[2], 1)
	return gateway, nil
}
