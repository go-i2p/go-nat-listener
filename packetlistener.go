package nattraversal

import (
    "fmt"
    "net"
    "sync"
)

// NATPacketListener implements a packet listener with NAT traversal.
type NATPacketListener struct {
    conn         net.PacketConn
    renewal      *RenewalManager
    externalPort int
    addr         *NATAddr
    closed       bool
    mu           sync.Mutex
}

// ListenPacket creates a UDP packet listener with NAT traversal on the specified port.
func ListenPacket(port int) (*NATPacketListener, error) {
    mapper, externalPort, err := createUDPMapping(port)
    if err != nil {
        return nil, fmt.Errorf("failed to create port mapping: %w", err)
    }
    
    conn, err := net.ListenPacket("udp", fmt.Sprintf(":%d", port))
    if err != nil {
        mapper.UnmapPort("UDP", externalPort)
        return nil, fmt.Errorf("failed to create packet conn: %w", err)
    }
    
    // Get addresses for NATAddr
    internalAddr := conn.LocalAddr().String()
    externalIP, err := mapper.GetExternalIP()
    if err != nil {
        conn.Close()
        mapper.UnmapPort("UDP", externalPort)
        return nil, fmt.Errorf("failed to get external IP: %w", err)
    }
    
    externalAddr := fmt.Sprintf("%s:%d", externalIP, externalPort)
    addr := NewNATAddr("udp", internalAddr, externalAddr)
    
    renewal := NewRenewalManager(mapper, "UDP", port, externalPort)
    renewal.Start()
    
    return &NATPacketListener{
        conn:         conn,
        renewal:      renewal,
        externalPort: externalPort,
        addr:         addr,
    }, nil
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

// createUDPMapping establishes a UDP port mapping.
func createUDPMapping(port int) (PortMapper, int, error) {
    mapper, err := NewPortMapper()
    if err != nil {
        return nil, 0, err
    }
    
    externalPort, err := mapper.MapPort("UDP", port, mappingDuration)
    if err != nil {
        return nil, 0, err
    }
    
    return mapper, externalPort, nil
}