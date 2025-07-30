# go-nat-listener

A NAT traversal library for Go that provides standard network interfaces with automatic port mapping and renewal. This library enables applications running behind SOHO routers to accept incoming connections by automatically configuring port forwarding through UPnP and NAT-PMP protocols.

## Features

- **Standard Go Network Interfaces**: Drop-in replacements for `net.Listen` and `net.ListenPacket`
- **Automatic NAT Traversal**: Supports UPnP and NAT-PMP protocols with automatic fallback
- **Port Renewal**: Automatically renews port mappings to maintain connectivity
- **TCP and UDP Support**: Works with both TCP listeners and UDP packet connections
- **External Address Discovery**: Provides access to both internal and external network addresses

## Installation

```bash
go get github.com/go-i2p/go-nat-listener
```

## Requirements

- Go 1.24.5 or later
- Router with UPnP or NAT-PMP support
- Network environment allowing NAT traversal protocols

## Quick Start

### TCP Listener

```go
package main

import (
    "fmt"
    "log"
    "net"
    
    "github.com/go-i2p/go-nat-listener"
)

func main() {
    // Create a NAT-traversing TCP listener on port 8080
    listener, err := nattraversal.Listen(8080)
    if err != nil {
        log.Fatal("Failed to create listener:", err)
    }
    defer listener.Close()
    
    fmt.Printf("Listening on %s (external: %s)\n", 
        listener.Addr().(*nattraversal.NATAddr).InternalAddr(),
        listener.Addr().String())
    
    for {
        conn, err := listener.Accept()
        if err != nil {
            log.Printf("Accept error: %v", err)
            continue
        }
        
        go handleConnection(conn)
    }
}

func handleConnection(conn net.Conn) {
    defer conn.Close()
    // Handle the connection...
}
```

### UDP Packet Listener

```go
package main

import (
    "fmt"
    "log"
    "net"
    
    "github.com/go-i2p/go-nat-listener"
)

func main() {
    // Create a NAT-traversing UDP listener on port 9090
    listener, err := nattraversal.ListenPacket(9090)
    if err != nil {
        log.Fatal("Failed to create packet listener:", err)
    }
    defer listener.Close()
    
    fmt.Printf("UDP listening on %s (external: %s)\n",
        listener.Addr().(*nattraversal.NATAddr).InternalAddr(),
        listener.Addr().String())
    
    // Get the underlying PacketConn for reading/writing
    conn := listener.PacketConn()
    
    buffer := make([]byte, 1024)
    for {
        n, addr, err := conn.ReadFrom(buffer)
        if err != nil {
            log.Printf("Read error: %v", err)
            continue
        }
        
        fmt.Printf("Received %d bytes from %s: %s\n", n, addr, string(buffer[:n]))
    }
}
```

## API Reference

### Core Functions

#### `Listen(port int) (*NATListener, error)`
Creates a TCP listener with NAT traversal on the specified port. Returns a `NATListener` that implements the standard `net.Listener` interface.

#### `ListenPacket(port int) (*NATPacketListener, error)`
Creates a UDP packet listener with NAT traversal on the specified port. Returns a `NATPacketListener` for UDP communication.

### Types

#### `NATListener`
Implements `net.Listener` with automatic NAT traversal support:
- `Accept() (net.Conn, error)` - Accepts incoming connections
- `Close() error` - Closes the listener and stops port renewal
- `Addr() net.Addr` - Returns the NAT-aware address

#### `NATPacketListener`
Provides UDP packet listening with NAT traversal:
- `Accept() (net.PacketConn, error)` - Returns the underlying packet connection
- `Close() error` - Closes the listener and stops port renewal
- `Addr() net.Addr` - Returns the NAT-aware address
- `PacketConn() net.PacketConn` - Direct access to the packet connection

#### `NATAddr`
Network address with NAT traversal information:
- `Network() string` - Returns the network type (tcp/udp)
- `String() string` - Returns the external address
- `InternalAddr() string` - Returns the internal network address
- `ExternalAddr() string` - Returns the external network address

#### `NATConn`
Wraps `net.Conn` with NAT-aware addressing:
- `LocalAddr() net.Addr` - Returns NAT-aware local address
- `RemoteAddr() net.Addr` - Returns remote address
- Embeds `net.Conn` for all standard connection operations

## How It Works

1. **Port Mapping**: When creating a listener, the library attempts to create a port mapping on your router using UPnP first, then falls back to NAT-PMP
2. **External IP Discovery**: Retrieves your router's external IP address
3. **Address Management**: Provides both internal (LAN) and external (WAN) addresses
4. **Automatic Renewal**: Continuously renews port mappings to prevent expiration
5. **Standard Interfaces**: Exposes familiar Go network interfaces for easy integration

## Supported Protocols

- **UPnP (Universal Plug and Play)**: Primary protocol for automatic port forwarding
- **NAT-PMP (NAT Port Mapping Protocol)**: Fallback protocol for routers that don't support UPnP

## Error Handling

The library provides descriptive error messages for common failure scenarios:
- No NAT traversal protocols available
- Port mapping failures
- External IP discovery issues
- Network connectivity problems

Example error handling:

```go
listener, err := nattraversal.Listen(8080)
if err != nil {
    log.Printf("NAT traversal failed: %v", err)
    // Fall back to local-only listener
    fallbackListener, err := net.Listen("tcp", ":8080")
    if err != nil {
        log.Fatal("All listener creation failed:", err)
    }
    listener = fallbackListener
}
```

## Limitations

- Requires router support for UPnP or NAT-PMP protocols
- May not work with symmetric NAT configurations
- Firewall settings may block automatic port mapping
- Some corporate/restricted networks disable these protocols

## Dependencies

- `github.com/huin/goupnp` - UPnP protocol implementation
- `github.com/jackpal/go-nat-pmp` - NAT-PMP protocol implementation

## License

See [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please ensure that:
- Code follows Go conventions and idioms
- Tests are included for new functionality
- Documentation is updated accordingly
- Changes maintain compatibility with existing APIs

