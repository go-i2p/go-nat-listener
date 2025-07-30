package nattraversal

import "net"

// NATConn wraps a net.Conn with NAT-aware addressing.
// Moved from: conn.go
type NATConn struct {
	net.Conn
	localAddr  *NATAddr
	remoteAddr net.Addr
}

// LocalAddr returns the local network address with NAT info.
func (c *NATConn) LocalAddr() net.Addr {
	return c.localAddr
}

// RemoteAddr returns the remote network address.
func (c *NATConn) RemoteAddr() net.Addr {
	return c.remoteAddr
}
