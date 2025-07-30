package nattraversal

import "time"

// PortMapper defines the interface for NAT traversal protocols.
type PortMapper interface {
	MapPort(protocol string, internalPort int, duration time.Duration) (externalPort int, err error)
	UnmapPort(protocol string, externalPort int) error
	GetExternalIP() (string, error)
}
