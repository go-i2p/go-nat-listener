// Package nattraversal provides NAT traversal using UPnP and NAT-PMP protocols
// with standard Go network interfaces and automatic port renewal.
package nattraversal

import (
	"context"
	"fmt"
)

// NewPortMapper creates a port mapper, trying direct connectivity first,
// then UPnP, then NAT-PMP.
// This is a convenience wrapper around NewPortMapperContext using context.Background().
func NewPortMapper() (PortMapper, error) {
	return NewPortMapperContext(context.Background())
}

// NewPortMapperContext creates a port mapper with context support, trying direct
// connectivity first, then UPnP, then NAT-PMP.
// The context is passed through to the discovery process, allowing cancellation during slow network operations.
func NewPortMapperContext(ctx context.Context) (PortMapper, error) {
	log.Debug("discovering port mapper")

	// Check context before starting
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("context cancelled: %w", err)
	}

	// Try direct connectivity first.
	direct, err := newDirectPortMapper()
	if err == nil {
		log.Debug("direct port mapper selected")
		return direct, nil
	}

	log.WithError(err).Debug("direct connectivity detection failed, trying UPnP")

	// Try UPnP with context support
	upnp, err := NewUPnPMapperContext(ctx)
	if err == nil {
		log.Debug("UPnP port mapper selected")
		return upnp, nil
	}

	log.WithError(err).Debug("UPnP discovery failed, trying NAT-PMP")

	// Check context before fallback
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("context cancelled after UPnP attempt: %w", err)
	}

	// Fall back to NAT-PMP
	natpmp, err := NewNATPMPMapper()
	if err != nil {
		log.WithError(err).Error("all NAT traversal protocols failed")
		return nil, fmt.Errorf("no NAT traversal available: UPnP failed, NAT-PMP failed: %w", err)
	}

	log.Debug("NAT-PMP port mapper selected")
	return natpmp, nil
}
