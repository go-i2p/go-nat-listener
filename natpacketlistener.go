package nattraversal

import (
	"fmt"
	"net"
	"sync"
)

// NATPacketListener implements a packet listener with NAT traversal.
// Moved from: packetlistener.go
type NATPacketListener struct {
	conn         net.PacketConn
	renewal      *RenewalManager
	externalPort int
	externalIP   string
	addr         *NATAddr
	closed       bool
	mu           sync.Mutex
	// cachedPacketConn is the cached NATPacketConn wrapper, created once and reused
	cachedPacketConn *NATPacketConn
}

// updateExternalPort handles external port changes during renewal.
// It updates the externalPort field and recreates the NATAddr with the new port.
func (l *NATPacketListener) updateExternalPort(newPort int) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.externalPort = newPort
	// Recreate NATAddr with the new external port
	newExternalAddr := fmt.Sprintf("%s:%d", l.externalIP, newPort)
	l.addr = NewNATAddr(l.addr.Network(), l.addr.InternalAddr(), newExternalAddr)

	// Update the cached packet conn's local address if it exists
	if l.cachedPacketConn != nil {
		l.cachedPacketConn.localAddr = l.addr
	}
}

// Accept returns a packet connection (satisfies a hypothetical net.PacketListener interface).
// Note: For UDP, this returns the same cached connection each time since UDP is connectionless.
// Unlike TCP's Accept which blocks waiting for new connections, this immediately returns
// the single packet connection associated with this listener.
func (l *NATPacketListener) Accept() (net.PacketConn, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.closed {
		return nil, fmt.Errorf("packet listener closed")
	}

	return l.getOrCreatePacketConn(), nil
}

// Close closes the packet listener and stops port renewal.
// This method is idempotent - calling it multiple times is safe.
// It coordinates with NATPacketConn.Close() to ensure the underlying
// connection is only closed once, even if both are called.
func (l *NATPacketListener) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.closed {
		return nil
	}
	l.closed = true

	l.renewal.Stop()

	// If a NATPacketConn was created, close through it to use sync.Once
	// This ensures the underlying connection is closed exactly once,
	// even if NATPacketConn.Close() was already called.
	if l.cachedPacketConn != nil {
		return l.cachedPacketConn.Close()
	}

	// No NATPacketConn was created, close the underlying conn directly
	return l.conn.Close()
}

// Addr returns the listener's network address.
func (l *NATPacketListener) Addr() net.Addr {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.addr
}

// ExternalPort returns the external port number assigned by the NAT device.
// This value may change if the NAT device assigns a different port during renewal.
func (l *NATPacketListener) ExternalPort() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.externalPort
}

// PacketConn returns the underlying packet connection.
// Returns the same cached instance on each call.
func (l *NATPacketListener) PacketConn() net.PacketConn {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.getOrCreatePacketConn()
}

// getOrCreatePacketConn returns the cached NATPacketConn or creates it if needed.
// Must be called with l.mu held.
func (l *NATPacketListener) getOrCreatePacketConn() *NATPacketConn {
	if l.cachedPacketConn == nil {
		l.cachedPacketConn = &NATPacketConn{
			PacketConn: l.conn,
			localAddr:  l.addr,
		}
	}
	return l.cachedPacketConn
}
