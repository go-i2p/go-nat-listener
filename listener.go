package nattraversal

import (
	"context"
	"fmt"
	"net"
)

// Listen creates a TCP listener with NAT traversal on the specified port.
// This is a convenience wrapper around ListenContext using context.Background().
func Listen(port int) (*NATListener, error) {
	return ListenContext(context.Background(), port)
}

// ListenContext creates a TCP listener with NAT traversal on the specified port.
// The context can be used to cancel the discovery and mapping operations.
// Once the listener is created, the context is no longer used - use Close() to stop the listener.
func ListenContext(ctx context.Context, port int) (*NATListener, error) {
	// Check context before starting
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("context cancelled before starting: %w", err)
	}

	mapper, externalPort, err := createTCPMappingContext(ctx, port)
	if err != nil {
		return nil, fmt.Errorf("failed to create port mapping: %w", err)
	}

	// Check context after mapping
	if err := ctx.Err(); err != nil {
		mapper.UnmapPort("TCP", externalPort)
		return nil, fmt.Errorf("context cancelled after mapping: %w", err)
	}

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		mapper.UnmapPort("TCP", externalPort)
		return nil, fmt.Errorf("failed to create listener: %w", err)
	}

	// Check context after creating listener
	if err := ctx.Err(); err != nil {
		listener.Close()
		mapper.UnmapPort("TCP", externalPort)
		return nil, fmt.Errorf("context cancelled after listener creation: %w", err)
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

	natListener := &NATListener{
		listener:     listener,
		renewal:      renewal,
		externalPort: externalPort,
		externalIP:   externalIP,
		addr:         addr,
	}

	// Set up callback to handle external port changes during renewal
	renewal.SetPortChangeCallback(natListener.updateExternalPort)
	renewal.Start()

	return natListener, nil
}
