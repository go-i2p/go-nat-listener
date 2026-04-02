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
	n, addr, err = c.PacketConn.ReadFrom(p)
	if err != nil {
		log.WithError(err).Debug("NAT packet conn read error")
	}
	return n, addr, err
}

// WriteTo writes a packet to the connection.
func (c *NATPacketConn) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	n, err = c.PacketConn.WriteTo(p, addr)
	if err != nil {
		log.WithError(err).WithField("addr", addr.String()).Error("NAT packet conn write error")
	}
	return n, err
}

// Close closes the connection.
// This method is idempotent - calling it multiple times is safe and will
// only close the underlying connection once, returning the same error.
func (c *NATPacketConn) Close() error {
	c.closeOnce.Do(func() {
		log.WithField("addr", c.localAddr.String()).Debug("closing NAT packet connection")
		c.closeErr = c.PacketConn.Close()
		if c.closeErr != nil {
			log.WithError(c.closeErr).Error("error closing NAT packet connection")
		}
	})
	return c.closeErr
}

// SetDeadline sets the read and write deadlines.
func (c *NATPacketConn) SetDeadline(t time.Time) error {
	err := c.PacketConn.SetDeadline(t)
	if err != nil {
		log.WithError(err).Debug("failed to set deadline on NAT packet conn")
	}
	return err
}

// SetReadDeadline sets the deadline for future ReadFrom calls.
func (c *NATPacketConn) SetReadDeadline(t time.Time) error {
	err := c.PacketConn.SetReadDeadline(t)
	if err != nil {
		log.WithError(err).Debug("failed to set read deadline on NAT packet conn")
	}
	return err
}

// SetWriteDeadline sets the deadline for future WriteTo calls.
func (c *NATPacketConn) SetWriteDeadline(t time.Time) error {
	err := c.PacketConn.SetWriteDeadline(t)
	if err != nil {
		log.WithError(err).Debug("failed to set write deadline on NAT packet conn")
	}
	return err
}
