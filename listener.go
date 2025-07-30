package nattraversal

import (
	"fmt"
	"net"
)

// Listen creates a TCP listener with NAT traversal on the specified port.
func Listen(port int) (*NATListener, error) {
	mapper, externalPort, err := createTCPMapping(port)
	if err != nil {
		return nil, fmt.Errorf("failed to create port mapping: %w", err)
	}

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		mapper.UnmapPort("TCP", externalPort)
		return nil, fmt.Errorf("failed to create listener: %w", err)
	}

	// Get addresses for NATAddr
	internalAddr := listener.Addr().String()
	externalIP, err := mapper.GetExternalIP()
	if err != nil {
		listener.Close()
		mapper.UnmapPort("TCP", externalPort)
		return nil, fmt.Errorf("failed to get external IP: %w", err)
	}

	externalAddr := fmt.Sprintf("%s:%d", externalIP, externalPort)
	addr := NewNATAddr("tcp", internalAddr, externalAddr)

	renewal := NewRenewalManager(mapper, "TCP", port, externalPort)
	renewal.Start()

	return &NATListener{
		listener:     listener,
		renewal:      renewal,
		externalPort: externalPort,
		addr:         addr,
	}, nil
}
