package nattraversal

import "github.com/go-i2p/logger"

// NATAddr represents a network address with NAT traversal information.
// Moved from: addr.go
type NATAddr struct {
	network      string
	internalAddr string
	externalAddr string
}

// NewNATAddr creates a new NATAddr with internal and external addresses.
func NewNATAddr(network, internalAddr, externalAddr string) *NATAddr {
	log.WithFields(logger.Fields{
		"network":      network,
		"internalAddr": internalAddr,
		"externalAddr": externalAddr,
	}).Debug("creating NATAddr")
	return &NATAddr{
		network:      network,
		internalAddr: internalAddr,
		externalAddr: externalAddr,
	}
}

// Network returns the network type (tcp/udp).
func (a *NATAddr) Network() string {
	return a.network
}

// String returns the external address for external connections.
func (a *NATAddr) String() string {
	return a.externalAddr
}

// InternalAddr returns the internal network address.
func (a *NATAddr) InternalAddr() string {
	return a.internalAddr
}

// ExternalAddr returns the external network address.
func (a *NATAddr) ExternalAddr() string {
	return a.externalAddr
}
