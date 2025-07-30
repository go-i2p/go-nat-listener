package nattraversal

import (
	"net"
	"time"
)

// NATPacketConn wraps a net.PacketConn with NAT-aware addressing.
// Moved from: packetconn.go
type NATPacketConn struct {
	net.PacketConn
	localAddr *NATAddr
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
func (c *NATPacketConn) Close() error {
	return c.PacketConn.Close()
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
