package nattraversal

import (
	"fmt"
	"net"
)

// createTCPMapping establishes a TCP port mapping.
// Moved from: listener.go
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

// createUDPMapping establishes a UDP port mapping.
// Moved from: packetlistener.go
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

// discoverGateway finds the default gateway for NAT-PMP.
// Moved from: natpmpmapper.go
func discoverGateway() (net.IP, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	ip := localAddr.IP.To4()
	if ip == nil {
		return nil, fmt.Errorf("not IPv4 address")
	}

	// Assume gateway is .1 in the same subnet
	gateway := net.IPv4(ip[0], ip[1], ip[2], 1)
	return gateway, nil
}
