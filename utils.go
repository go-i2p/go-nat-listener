package nattraversal

import (
	"context"
)

// createTCPMapping establishes a TCP port mapping.
// Moved from: listener.go
func createTCPMapping(port int) (PortMapper, int, error) {
	return createTCPMappingContext(context.Background(), port)
}

// createTCPMappingContext establishes a TCP port mapping with context support.
// The context is checked before and after the discovery and mapping operations.
func createTCPMappingContext(ctx context.Context, port int) (PortMapper, int, error) {
	if err := ctx.Err(); err != nil {
		return nil, 0, err
	}

	mapper, err := NewPortMapperContext(ctx)
	if err != nil {
		return nil, 0, err
	}

	if err := ctx.Err(); err != nil {
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
	return createUDPMappingContext(context.Background(), port)
}

// createUDPMappingContext establishes a UDP port mapping with context support.
// The context is checked before and after the discovery and mapping operations.
func createUDPMappingContext(ctx context.Context, port int) (PortMapper, int, error) {
	if err := ctx.Err(); err != nil {
		return nil, 0, err
	}

	mapper, err := NewPortMapperContext(ctx)
	if err != nil {
		return nil, 0, err
	}

	if err := ctx.Err(); err != nil {
		return nil, 0, err
	}

	externalPort, err := mapper.MapPort("UDP", port, mappingDuration)
	if err != nil {
		return nil, 0, err
	}

	return mapper, externalPort, nil
}

// Gateway discovery functions have been moved to platform-specific files:
// - gateway.go: discoverGateway() and discoverGatewayFallback() (cross-platform)
// - gateway_linux.go: readDefaultGateway() using /proc/net/route
// - gateway_bsd.go: readDefaultGateway() using netstat (macOS, FreeBSD, OpenBSD, etc.)
// - gateway_windows.go: readDefaultGateway() using route print
// - gateway_other.go: stub for other platforms (uses fallback)
