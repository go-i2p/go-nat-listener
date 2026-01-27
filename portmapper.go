// Package nattraversal provides NAT traversal using UPnP and NAT-PMP protocols
// with standard Go network interfaces and automatic port renewal.
package nattraversal

import (
	"context"
	"fmt"
)

// NewPortMapper creates a port mapper, trying UPnP first, then NAT-PMP.
// This is a convenience wrapper around NewPortMapperContext using context.Background().
func NewPortMapper() (PortMapper, error) {
	return NewPortMapperContext(context.Background())
}

// NewPortMapperContext creates a port mapper with context support, trying UPnP first, then NAT-PMP.
// The context is passed through to the discovery process, allowing cancellation during slow network operations.
func NewPortMapperContext(ctx context.Context) (PortMapper, error) {
	// Check context before starting
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("context cancelled: %w", err)
	}

	// Try UPnP first with context support
	upnp, err := NewUPnPMapperContext(ctx)
	if err == nil {
		return upnp, nil
	}

	// Check context before fallback
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("context cancelled after UPnP attempt: %w", err)
	}

	// Fall back to NAT-PMP
	natpmp, err := NewNATPMPMapper()
	if err != nil {
		return nil, fmt.Errorf("no NAT traversal available: UPnP failed, NAT-PMP failed: %w", err)
	}

	return natpmp, nil
}
