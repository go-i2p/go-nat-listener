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
}

// Accept returns a packet connection (satisfies a hypothetical net.PacketListener interface).
func (l *NATPacketListener) Accept() (net.PacketConn, error) {
	l.mu.Lock()
	closed := l.closed
	l.mu.Unlock()

	if closed {
		return nil, fmt.Errorf("packet listener closed")
	}

	return &NATPacketConn{
		PacketConn: l.conn,
		localAddr:  l.addr,
	}, nil
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
func (l *NATPacketListener) PacketConn() net.PacketConn {
	return &NATPacketConn{
		PacketConn: l.conn,
		localAddr:  l.addr,
	}
}
