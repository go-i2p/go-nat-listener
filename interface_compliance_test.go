package nattraversal

import (
	"net"
	"testing"
)

// TestNATListenerImplementsNetListener verifies NATListener implements net.Listener
func TestNATListenerImplementsNetListener(t *testing.T) {
	var _ net.Listener = (*NATListener)(nil)
	t.Log("NATListener implements net.Listener")
}

// TestNATAddrImplementsNetAddr verifies NATAddr implements net.Addr
func TestNATAddrImplementsNetAddr(t *testing.T) {
	var _ net.Addr = (*NATAddr)(nil)
	t.Log("NATAddr implements net.Addr")
}

// TestNATConnImplementsNetConn verifies NATConn implements net.Conn
func TestNATConnImplementsNetConn(t *testing.T) {
	var _ net.Conn = (*NATConn)(nil)
	t.Log("NATConn implements net.Conn")
}

// TestNATPacketConnImplementsNetPacketConn verifies NATPacketConn implements net.PacketConn
func TestNATPacketConnImplementsNetPacketConn(t *testing.T) {
	var _ net.PacketConn = (*NATPacketConn)(nil)
	t.Log("NATPacketConn implements net.PacketConn")
}
