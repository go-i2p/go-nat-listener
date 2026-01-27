package nattraversal

import (
	"fmt"
	"net"
	"strings"
	"time"

	natpmp "github.com/jackpal/go-nat-pmp"
)

// NATPMPMapper implements PortMapper using NAT-PMP protocol.
// Moved from: addr.go
type NATPMPMapper struct {
	client *natpmp.Client
}

// NewNATPMPMapper discovers and creates a NAT-PMP mapper.
func NewNATPMPMapper() (*NATPMPMapper, error) {
	gateway, err := discoverGateway()
	if err != nil {
		return nil, fmt.Errorf("NAT-PMP gateway discovery failed: %w", err)
	}

	client := natpmp.NewClient(gateway)

	// Test connectivity
	_, err = client.GetExternalAddress()
	if err != nil {
		return nil, fmt.Errorf("NAT-PMP connectivity test failed: %w", err)
	}

	return &NATPMPMapper{client: client}, nil
}

// MapPort creates a port mapping via NAT-PMP.
func (n *NATPMPMapper) MapPort(protocol string, internalPort int, duration time.Duration) (int, error) {
	// Validate port range to prevent invalid mappings
	if internalPort < 1 || internalPort > 65535 {
		return 0, fmt.Errorf("invalid port number: %d (must be 1-65535)", internalPort)
	}

	protocolStr := strings.ToUpper(protocol)
	if protocolStr != "TCP" && protocolStr != "UDP" {
		return 0, fmt.Errorf("unsupported protocol: %s", protocol)
	}

	result, err := n.client.AddPortMapping(
		protocolStr,
		internalPort,
		internalPort,
		int(duration.Seconds()),
	)

	if err != nil {
		return 0, fmt.Errorf("NAT-PMP port mapping failed: %w", err)
	}

	return int(result.MappedExternalPort), nil
}

// UnmapPort removes a port mapping via NAT-PMP.
func (n *NATPMPMapper) UnmapPort(protocol string, externalPort int) error {
	// Validate port range to prevent invalid unmappings
	if externalPort < 1 || externalPort > 65535 {
		return fmt.Errorf("invalid port number: %d (must be 1-65535)", externalPort)
	}

	protocolStr := strings.ToUpper(protocol)
	if protocolStr != "TCP" && protocolStr != "UDP" {
		return fmt.Errorf("unsupported protocol: %s", protocol)
	}

	_, err := n.client.AddPortMapping(protocolStr, externalPort, 0, 0)
	if err != nil {
		return fmt.Errorf("NAT-PMP port unmapping failed: %w", err)
	}

	return nil
}

// GetExternalIP returns the external IP address via NAT-PMP.
func (n *NATPMPMapper) GetExternalIP() (string, error) {
	result, err := n.client.GetExternalAddress()
	if err != nil {
		return "", fmt.Errorf("NAT-PMP external IP lookup failed: %w", err)
	}
	ip := net.IPv4(result.ExternalIPAddress[0], result.ExternalIPAddress[1],
		result.ExternalIPAddress[2], result.ExternalIPAddress[3])
	return ip.String(), nil
}
