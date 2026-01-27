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
	addr         *NATAddr
	closed       bool
	mu           sync.Mutex
	// cachedPacketConn is the cached NATPacketConn wrapper, created once and reused
	cachedPacketConn *NATPacketConn
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
func (l *NATPacketListener) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.closed {
		return nil
	}
	l.closed = true

	l.renewal.Stop()
	return l.conn.Close()
}

// Addr returns the listener's network address.
func (l *NATPacketListener) Addr() net.Addr {
	return l.addr
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
