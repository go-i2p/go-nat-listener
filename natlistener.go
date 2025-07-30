package nattraversal

import (
	"fmt"
	"net"
	"sync"
)

// NATListener implements net.Listener with automatic NAT traversal.
// Moved from: listener.go
type NATListener struct {
	listener     net.Listener
	renewal      *RenewalManager
	externalPort int
	addr         *NATAddr
	closed       bool
	mu           sync.Mutex
}

// Accept waits for and returns the next connection to the listener.
func (l *NATListener) Accept() (net.Conn, error) {
	l.mu.Lock()
	closed := l.closed
	l.mu.Unlock()

	if closed {
		return nil, fmt.Errorf("listener closed")
	}

	conn, err := l.listener.Accept()
	if err != nil {
		return nil, err
	}

	return &NATConn{
		Conn:       conn,
		localAddr:  l.addr,
		remoteAddr: conn.RemoteAddr(),
	}, nil
}

// Close closes the listener and stops port renewal.
func (l *NATListener) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.closed {
		return nil
	}
	l.closed = true

	l.renewal.Stop()
	return l.listener.Close()
}

// Addr returns the listener's network address.
func (l *NATListener) Addr() net.Addr {
	return l.addr
}
