package nattraversal

import (
	"fmt"
	"net"

	"github.com/go-i2p/logger"
)

// discoverGateway finds the default gateway for NAT-PMP.
// It uses platform-specific methods to read the system routing table,
// falling back to a heuristic if the routing table cannot be read.
func discoverGateway() (net.IP, error) {
	log.Debug("discovering default gateway")

	// Try to read the actual gateway from the routing table (platform-specific)
	gateway, err := readDefaultGateway()
	if err == nil && gateway != nil {
		log.WithField("gateway", gateway.String()).Debug("gateway found via routing table")
		return gateway, nil
	}

	if err != nil {
		log.WithError(err).Debug("routing table read failed, trying fallback gateway discovery")
	} else {
		log.Debug("no gateway in routing table, trying fallback gateway discovery")
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
	log.Debug("using fallback gateway discovery (assuming .1 gateway)")

	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.WithError(err).Error("failed to determine local IP for gateway fallback")
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
	log.WithFields(logger.Fields{
		"localIP": localAddr.IP.String(),
		"gateway": gateway.String(),
	}).Debug("fallback gateway determined")
	return gateway, nil
}
