// Package nattraversal provides NAT traversal using UPnP and NAT-PMP protocols
// with standard Go network interfaces and automatic port renewal.
package nattraversal

import (
    "fmt"
    "time"
)

// PortMapper defines the interface for NAT traversal protocols.
type PortMapper interface {
    MapPort(protocol string, internalPort int, duration time.Duration) (externalPort int, error)
    UnmapPort(protocol string, externalPort int) error
    GetExternalIP() (string, error)
}

// NewPortMapper creates a port mapper, trying UPnP first, then NAT-PMP.
func NewPortMapper() (PortMapper, error) {
    // Try UPnP first
    upnp, err := NewUPnPMapper()
    if err == nil {
        return upnp, nil
    }
    
    // Fall back to NAT-PMP
    natpmp, err := NewNATPMPMapper()
    if err != nil {
        return nil, fmt.Errorf("no NAT traversal available: UPnP failed, NAT-PMP failed: %w", err)
    }
    
    return natpmp, nil
}