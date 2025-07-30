package nattraversal

import (
	"fmt"
	"net"
	"strings"
	"time"

	natpmp "github.com/jackpal/go-nat-pmp"
)

// NATAddr represents a network address with NAT traversal information.
// Moved from: original implementation (missing type)
type NATAddr struct {
	network      string
	internalAddr string
	externalAddr string
}

// NewNATAddr creates a new NATAddr with internal and external addresses.
func NewNATAddr(network, internalAddr, externalAddr string) *NATAddr {
	return &NATAddr{
		network:      network,
		internalAddr: internalAddr,
		externalAddr: externalAddr,
	}
}

// Network returns the network type (tcp/udp).
func (a *NATAddr) Network() string {
	return a.network
}

// String returns the external address for external connections.
func (a *NATAddr) String() string {
	return a.externalAddr
}

// InternalAddr returns the internal network address.
func (a *NATAddr) InternalAddr() string {
	return a.internalAddr
}

// ExternalAddr returns the external network address.
func (a *NATAddr) ExternalAddr() string {
	return a.externalAddr
}

// NATPMPMapper implements PortMapper using NAT-PMP protocol.
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

// discoverGateway finds the default gateway for NAT-PMP.
func discoverGateway() (net.IP, error) {
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

	// Assume gateway is .1 in the same subnet
	gateway := net.IPv4(ip[0], ip[1], ip[2], 1)
	return gateway, nil
}
