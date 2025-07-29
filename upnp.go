package nattraversal

import (
    "fmt"
    "net"
    "time"
    
    "github.com/huin/goupnp/dcps/internetgateway2"
)

// UPnPMapper implements PortMapper using UPnP IGD protocol.
type UPnPMapper struct {
    client internetgateway2.WANPPPConnection1
}

// NewUPnPMapper discovers and creates a UPnP mapper.
func NewUPnPMapper() (*UPnPMapper, error) {
    clients, _, err := internetgateway2.NewWANPPPConnection1Clients()
    if err != nil {
        return nil, fmt.Errorf("UPnP discovery failed: %w", err)
    }
    
    if len(clients) == 0 {
        return nil, fmt.Errorf("no UPnP IGD devices found")
    }
    
    return &UPnPMapper{client: *clients[0]}, nil
}

// MapPort creates a port mapping via UPnP.
func (u *UPnPMapper) MapPort(protocol string, internalPort int, duration time.Duration) (int, error) {
    localIP, err := u.getLocalIP()
    if err != nil {
        return 0, fmt.Errorf("failed to get local IP: %w", err)
    }
    
    leaseDuration := uint32(duration.Seconds())
    
    err = u.client.AddPortMapping(
        "",                    // remote host (any)
        uint16(internalPort),  // external port (same as internal)
        protocol,              // TCP or UDP
        uint16(internalPort),  // internal port
        localIP,               // internal client
        true,                  // enabled
        "nattraversal",        // description
        leaseDuration,         // lease duration
    )
    
    if err != nil {
        return 0, fmt.Errorf("UPnP port mapping failed: %w", err)
    }
    
    return internalPort, nil
}

// UnmapPort removes a port mapping via UPnP.
func (u *UPnPMapper) UnmapPort(protocol string, externalPort int) error {
    err := u.client.DeletePortMapping("", uint16(externalPort), protocol)
    if err != nil {
        return fmt.Errorf("UPnP port unmapping failed: %w", err)
    }
    return nil
}

// GetExternalIP returns the external IP address via UPnP.
func (u *UPnPMapper) GetExternalIP() (string, error) {
    ip, err := u.client.GetExternalIPAddress()
    if err != nil {
        return "", fmt.Errorf("UPnP external IP lookup failed: %w", err)
    }
    return ip, nil
}

// getLocalIP discovers the local IP address for port mapping.
func (u *UPnPMapper) getLocalIP() (string, error) {
    conn, err := net.Dial("udp", "8.8.8.8:80")
    if err != nil {
        return "", err
    }
    defer conn.Close()
    
    localAddr := conn.LocalAddr().(*net.UDPAddr)
    return localAddr.IP.String(), nil
}