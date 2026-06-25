package nattraversal

import (
	"fmt"
	"net"
	"time"
)

var cgnatIPv4Net = &net.IPNet{
	IP:   net.IPv4(100, 64, 0, 0),
	Mask: net.CIDRMask(10, 32),
}

// DirectPortMapper represents direct public connectivity where NAT traversal
// protocols are not required.
type DirectPortMapper struct {
	publicIP string
}

// Ensure DirectPortMapper satisfies the PortMapper interface.
var _ PortMapper = (*DirectPortMapper)(nil)

func newDirectPortMapper() (*DirectPortMapper, error) {
	ip, err := detectDirectPublicIP()
	if err != nil {
		return nil, err
	}

	return &DirectPortMapper{publicIP: ip}, nil
}

// MapPort is a no-op for direct connectivity and returns the internal port unchanged.
func (d *DirectPortMapper) MapPort(_ string, internalPort int, _ time.Duration) (int, error) {
	return internalPort, nil
}

// UnmapPort is a no-op for direct connectivity.
func (d *DirectPortMapper) UnmapPort(_ string, _ int) error {
	return nil
}

// GetExternalIP returns the detected directly-routable public IP.
func (d *DirectPortMapper) GetExternalIP() (string, error) {
	if d.publicIP == "" {
		return "", fmt.Errorf("no public IP available")
	}

	return d.publicIP, nil
}

func detectDirectPublicIP() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", fmt.Errorf("failed to list interfaces: %w", err)
	}

	var ipv4Candidate net.IP
	var ipv6Candidate net.IP

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			ip := ipFromAddr(addr)
			if ip == nil || !isGloballyRoutableIP(ip) {
				continue
			}

			if ip4 := ip.To4(); ip4 != nil {
				if ipv4Candidate == nil {
					ipv4Candidate = ip4
				}
				continue
			}

			if ipv6Candidate == nil {
				ipv6Candidate = ip
			}
		}
	}

	if ipv4Candidate != nil {
		return ipv4Candidate.String(), nil
	}

	if ipv6Candidate != nil {
		return ipv6Candidate.String(), nil
	}

	return "", fmt.Errorf("no globally routable interface IP found")
}

func ipFromAddr(addr net.Addr) net.IP {
	switch a := addr.(type) {
	case *net.IPNet:
		return a.IP
	case *net.IPAddr:
		return a.IP
	default:
		return nil
	}
}

func isGloballyRoutableIP(ip net.IP) bool {
	if ip == nil {
		return false
	}

	if !ip.IsGlobalUnicast() {
		return false
	}

	if ip.IsPrivate() ||
		ip.IsLoopback() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsMulticast() ||
		ip.IsUnspecified() {
		return false
	}

	if ip4 := ip.To4(); ip4 != nil && cgnatIPv4Net.Contains(ip4) {
		return false
	}

	return true
}
