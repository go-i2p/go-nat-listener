package nattraversal

import (
	"fmt"
	"net"
)

// ListenPacket creates a UDP packet listener with NAT traversal on the specified port.
func ListenPacket(port int) (*NATPacketListener, error) {
	mapper, externalPort, err := createUDPMapping(port)
	if err != nil {
		return nil, fmt.Errorf("failed to create port mapping: %w", err)
	}

	conn, err := net.ListenPacket("udp", fmt.Sprintf(":%d", port))
	if err != nil {
		mapper.UnmapPort("UDP", externalPort)
		return nil, fmt.Errorf("failed to create packet conn: %w", err)
	}

	// Get addresses for NATAddr
	internalAddr := conn.LocalAddr().String()
	externalIP, err := mapper.GetExternalIP()
	if err != nil {
		conn.Close()
		mapper.UnmapPort("UDP", externalPort)
		return nil, fmt.Errorf("failed to get external IP: %w", err)
	}

	externalAddr := fmt.Sprintf("%s:%d", externalIP, externalPort)
	addr := NewNATAddr("udp", internalAddr, externalAddr)

	renewal := NewRenewalManager(mapper, "UDP", port, externalPort)
	renewal.Start()

	return &NATPacketListener{
		conn:         conn,
		renewal:      renewal,
		externalPort: externalPort,
		addr:         addr,
	}, nil
}
