package nattraversal

import (
	"fmt"
	"net"
	"time"

	"github.com/huin/goupnp/dcps/internetgateway2"
)

// upnpClient defines the interface for UPnP IGD client operations.
// This is satisfied by WANIPConnection1, WANIPConnection2, and WANPPPConnection1.
type upnpClient interface {
	AddPortMapping(
		NewRemoteHost string,
		NewExternalPort uint16,
		NewProtocol string,
		NewInternalPort uint16,
		NewInternalClient string,
		NewEnabled bool,
		NewPortMappingDescription string,
		NewLeaseDuration uint32,
	) error
	DeletePortMapping(
		NewRemoteHost string,
		NewExternalPort uint16,
		NewProtocol string,
	) error
	GetExternalIPAddress() (string, error)
}

// UPnPMapper implements PortMapper using UPnP IGD protocol.
// Supports WANIPConnection1, WANIPConnection2, and WANPPPConnection1 services.
type UPnPMapper struct {
	client upnpClient
}

// NewUPnPMapper discovers and creates a UPnP mapper.
// It attempts discovery in order of preference: WANIPConnection2, WANIPConnection1,
// then WANPPPConnection1, using the first service that responds with available devices.
func NewUPnPMapper() (*UPnPMapper, error) {
	// Try WANIPConnection2 first (newest, most feature-rich)
	if client, err := discoverWANIPConnection2(); err == nil {
		return &UPnPMapper{client: client}, nil
	}

	// Try WANIPConnection1 (common on cable/fiber routers)
	if client, err := discoverWANIPConnection1(); err == nil {
		return &UPnPMapper{client: client}, nil
	}

	// Try WANPPPConnection1 (PPPoE routers like DSL)
	if client, err := discoverWANPPPConnection1(); err == nil {
		return &UPnPMapper{client: client}, nil
	}

	return nil, fmt.Errorf("no UPnP IGD devices found (tried WANIPConnection2, WANIPConnection1, WANPPPConnection1)")
}

// discoverWANIPConnection2 attempts to find WANIPConnection2 clients.
func discoverWANIPConnection2() (upnpClient, error) {
	clients, _, err := internetgateway2.NewWANIPConnection2Clients()
	if err != nil {
		return nil, err
	}
	if len(clients) == 0 {
		return nil, fmt.Errorf("no WANIPConnection2 devices found")
	}
	return clients[0], nil
}

// discoverWANIPConnection1 attempts to find WANIPConnection1 clients.
func discoverWANIPConnection1() (upnpClient, error) {
	clients, _, err := internetgateway2.NewWANIPConnection1Clients()
	if err != nil {
		return nil, err
	}
	if len(clients) == 0 {
		return nil, fmt.Errorf("no WANIPConnection1 devices found")
	}
	return clients[0], nil
}

// discoverWANPPPConnection1 attempts to find WANPPPConnection1 clients.
func discoverWANPPPConnection1() (upnpClient, error) {
	clients, _, err := internetgateway2.NewWANPPPConnection1Clients()
	if err != nil {
		return nil, err
	}
	if len(clients) == 0 {
		return nil, fmt.Errorf("no WANPPPConnection1 devices found")
	}
	return clients[0], nil
}

// MapPort creates a port mapping via UPnP.
func (u *UPnPMapper) MapPort(protocol string, internalPort int, duration time.Duration) (int, error) {
	localIP, err := u.getLocalIP()
	if err != nil {
		return 0, fmt.Errorf("failed to get local IP: %w", err)
	}

	leaseDuration := uint32(duration.Seconds())

	err = u.client.AddPortMapping(
		"",                   // remote host (any)
		uint16(internalPort), // external port (same as internal)
		protocol,             // TCP or UDP
		uint16(internalPort), // internal port
		localIP,              // internal client
		true,                 // enabled
		"nattraversal",       // description
		leaseDuration,        // lease duration
	)

	if err != nil {
		return 0, fmt.Errorf("UPnP port mapping failed: %w", err)
	}

	return internalPort, nil
}

// UnmapPort removes a port mapping via UPnP.
func (u *UPnPMapper) UnmapPort(protocol string, externalPort int) error {
	err := u.client.DeletePortMapping("", uint16(externalPort), protocol)
	if err != nil {
		return fmt.Errorf("UPnP port unmapping failed: %w", err)
	}
	return nil
}

// GetExternalIP returns the external IP address via UPnP.
func (u *UPnPMapper) GetExternalIP() (string, error) {
	ip, err := u.client.GetExternalIPAddress()
	if err != nil {
		return "", fmt.Errorf("UPnP external IP lookup failed: %w", err)
	}
	return ip, nil
}

// getLocalIP discovers the local IP address for port mapping.
func (u *UPnPMapper) getLocalIP() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String(), nil
}
