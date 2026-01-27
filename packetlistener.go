package nattraversal

import (
	"context"
	"fmt"
	"net"
)

// ListenPacket creates a UDP packet listener with NAT traversal on the specified port.
// This is a convenience wrapper around ListenPacketContext using context.Background().
func ListenPacket(port int) (*NATPacketListener, error) {
	return ListenPacketContext(context.Background(), port)
}

// ListenPacketContext creates a UDP packet listener with NAT traversal on the specified port.
// The context can be used to cancel the discovery and mapping operations.
// Once the listener is created, the context is no longer used - use Close() to stop the listener.
func ListenPacketContext(ctx context.Context, port int) (*NATPacketListener, error) {
	// Check context before starting
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("context cancelled before starting: %w", err)
	}

	mapper, externalPort, err := createUDPMappingContext(ctx, port)
	if err != nil {
		return nil, fmt.Errorf("failed to create port mapping: %w", err)
	}

	// Check context after mapping
	if err := ctx.Err(); err != nil {
		mapper.UnmapPort("UDP", externalPort)
		return nil, fmt.Errorf("context cancelled after mapping: %w", err)
	}

	conn, err := net.ListenPacket("udp", fmt.Sprintf(":%d", port))
	if err != nil {
		mapper.UnmapPort("UDP", externalPort)
		return nil, fmt.Errorf("failed to create packet conn: %w", err)
	}

	// Check context after creating connection
	if err := ctx.Err(); err != nil {
		conn.Close()
		mapper.UnmapPort("UDP", externalPort)
		return nil, fmt.Errorf("context cancelled after connection creation: %w", err)
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

	packetListener := &NATPacketListener{
		conn:         conn,
		renewal:      renewal,
		externalPort: externalPort,
		externalIP:   externalIP,
		addr:         addr,
	}

	// Set up callback to handle external port changes during renewal
	renewal.SetPortChangeCallback(packetListener.updateExternalPort)
	renewal.Start()

	return packetListener, nil
}
