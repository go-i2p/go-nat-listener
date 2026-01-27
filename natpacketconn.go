package nattraversal

import (
	"net"
	"sync"
	"time"
)

// NATPacketConn wraps a net.PacketConn with NAT-aware addressing.
// It coordinates with NATPacketListener to ensure Close() is idempotent
// and safe to call from either the connection or the listener.
type NATPacketConn struct {
	net.PacketConn
	localAddr *NATAddr

	// closeOnce ensures the underlying connection is closed exactly once,
	// preventing double-close issues when both NATPacketConn.Close() and
	// NATPacketListener.Close() are called.
	closeOnce sync.Once
	closeErr  error
}

// LocalAddr returns the local network address with NAT info.
func (c *NATPacketConn) LocalAddr() net.Addr {
	return c.localAddr
}

// ReadFrom reads a packet from the connection.
func (c *NATPacketConn) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	return c.PacketConn.ReadFrom(p)
}

// WriteTo writes a packet to the connection.
func (c *NATPacketConn) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	return c.PacketConn.WriteTo(p, addr)
}

// Close closes the connection.
// This method is idempotent - calling it multiple times is safe and will
// only close the underlying connection once, returning the same error.
func (c *NATPacketConn) Close() error {
	c.closeOnce.Do(func() {
		c.closeErr = c.PacketConn.Close()
	})
	return c.closeErr
}

// SetDeadline sets the read and write deadlines.
func (c *NATPacketConn) SetDeadline(t time.Time) error {
	return c.PacketConn.SetDeadline(t)
}

// SetReadDeadline sets the deadline for future ReadFrom calls.
func (c *NATPacketConn) SetReadDeadline(t time.Time) error {
	return c.PacketConn.SetReadDeadline(t)
}

// SetWriteDeadline sets the deadline for future WriteTo calls.
func (c *NATPacketConn) SetWriteDeadline(t time.Time) error {
	return c.PacketConn.SetWriteDeadline(t)
}
