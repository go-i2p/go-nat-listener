package nattraversal

import (
    "fmt"
    "net"
    "sync"
)

// NATListener implements net.Listener with automatic NAT traversal.
type NATListener struct {
    listener     net.Listener
    renewal      *RenewalManager
    externalPort int
    addr         *NATAddr
    closed       bool
    mu           sync.Mutex
}

// Listen creates a TCP listener with NAT traversal on the specified port.
func Listen(port int) (*NATListener, error) {
    mapper, externalPort, err := createTCPMapping(port)
    if err != nil {
        return nil, fmt.Errorf("failed to create port mapping: %w", err)
    }
    
    listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
    if err != nil {
        mapper.UnmapPort("TCP", externalPort)
        return nil, fmt.Errorf("failed to create listener: %w", err)
    }
    
    // Get addresses for NATAddr
    internalAddr := listener.Addr().String()
    externalIP, err := mapper.GetExternalIP()
    if err != nil {
        listener.Close()
        mapper.UnmapPort("TCP", externalPort)
        return nil, fmt.Errorf("failed to get external IP: %w", err)
    }
    
    externalAddr := fmt.Sprintf("%s:%d", externalIP, externalPort)
    addr := NewNATAddr("tcp", internalAddr, externalAddr)
    
    renewal := NewRenewalManager(mapper, "TCP", port, externalPort)
    renewal.Start()
    
    return &NATListener{
        listener:     listener,
        renewal:      renewal,
        externalPort: externalPort,
        addr:         addr,
    }, nil
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

// createTCPMapping establishes a TCP port mapping.
func createTCPMapping(port int) (PortMapper, int, error) {
    mapper, err := NewPortMapper()
    if err != nil {
        return nil, 0, err
    }
    
    externalPort, err := mapper.MapPort("TCP", port, mappingDuration)
    if err != nil {
        return nil, 0, err
    }
    
    return mapper, externalPort, nil
}