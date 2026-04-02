package nattraversal

import (
	"fmt"
	"net"
	"sync"

	"github.com/go-i2p/logger"
)

// NATListener implements net.Listener with automatic NAT traversal.
// Moved from: listener.go
type NATListener struct {
	listener     net.Listener
	renewal      *RenewalManager
	externalPort int
	externalIP   string
	addr         *NATAddr
	closed       bool
	fallback     bool // true if NAT traversal failed and we're using a standard listener
	mu           sync.Mutex
}

// updateExternalPort handles external port changes during renewal.
// It updates the externalPort field and recreates the NATAddr with the new port.
func (l *NATListener) updateExternalPort(newPort int) {
	l.mu.Lock()
	defer l.mu.Unlock()

	oldPort := l.externalPort
	l.externalPort = newPort
	// Recreate NATAddr with the new external port
	newExternalAddr := fmt.Sprintf("%s:%d", l.externalIP, newPort)
	l.addr = NewNATAddr(l.addr.Network(), l.addr.InternalAddr(), newExternalAddr)
	log.WithFields(logger.Fields{
		"oldPort": oldPort,
		"newPort": newPort,
	}).Debug("TCP listener external port updated")
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
		log.WithError(err).Debug("TCP listener accept error")
		return nil, err
	}

	log.WithFields(logger.Fields{
		"remoteAddr": conn.RemoteAddr().String(),
		"localAddr":  l.addr.String(),
	}).Debug("accepted new TCP connection")

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

	log.WithFields(logger.Fields{
		"addr":     l.addr.String(),
		"fallback": l.fallback,
	}).Debug("closing TCP listener")

	if l.renewal != nil {
		l.renewal.Stop()
	}
	err := l.listener.Close()
	if err != nil {
		log.WithError(err).Error("error closing TCP listener")
	}
	return err
}

// Addr returns the listener's network address.
func (l *NATListener) Addr() net.Addr {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.addr
}

// ExternalPort returns the external port number assigned by the NAT device.
// This value may change if the NAT device assigns a different port during renewal.
// In fallback mode, this returns the same as the internal port.
func (l *NATListener) ExternalPort() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.externalPort
}

// IsFallback returns true if NAT traversal failed and the listener is using
// a standard net.Listener without NAT hole-punching.
func (l *NATListener) IsFallback() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.fallback
}
