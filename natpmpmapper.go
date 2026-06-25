package nattraversal

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/go-i2p/logger"
	natpmp "github.com/jackpal/go-nat-pmp"
)

// NATPMPMapper implements PortMapper using NAT-PMP protocol.
// Moved from: addr.go
type NATPMPMapper struct {
	client *natpmp.Client
}

// NewNATPMPMapper discovers and creates a NAT-PMP mapper.
func NewNATPMPMapper() (*NATPMPMapper, error) {
	log.Debug("starting NAT-PMP gateway discovery")

	gateway, err := discoverGateway()
	if err != nil {
		log.WithError(err).Error("NAT-PMP gateway discovery failed")
		return nil, fmt.Errorf("NAT-PMP gateway discovery failed: %w", err)
	}

	log.WithField("gateway", gateway.String()).Debug("NAT-PMP gateway discovered")
	client := natpmp.NewClient(gateway)

	// Test connectivity — treat failure as non-fatal so NAT traversal can still work via other methods
	_, err = client.GetExternalAddress()
	if err != nil {
		log.WithError(err).WithField("gateway", gateway.String()).Warning("NAT-PMP connectivity test failed — will attempt fallback")
		return nil, fmt.Errorf("NAT-PMP connectivity test failed: %w", err)
	}

	log.WithField("gateway", gateway.String()).Debug("NAT-PMP mapper created successfully")
	return &NATPMPMapper{client: client}, nil
}

// MapPort creates a port mapping via NAT-PMP.
func (n *NATPMPMapper) MapPort(protocol string, internalPort int, duration time.Duration) (int, error) {
	log.WithFields(logger.Fields{
		"protocol":     protocol,
		"internalPort": internalPort,
		"duration":     duration.String(),
	}).Debug("mapping port via NAT-PMP")

	// Validate port range to prevent invalid mappings
	if internalPort < 1 || internalPort > 65535 {
		return 0, fmt.Errorf("invalid port number: %d (must be 1-65535)", internalPort)
	}

	protocolStr := strings.ToUpper(protocol)
	if protocolStr != "TCP" && protocolStr != "UDP" {
		return 0, fmt.Errorf("unsupported protocol: %s", protocol)
	}

	result, err := n.client.AddPortMapping(
		protocolStr,
		internalPort,
		internalPort,
		int(duration.Seconds()),
	)
	if err != nil {
		log.WithError(err).WithFields(logger.Fields{
			"protocol":     protocol,
			"internalPort": internalPort,
		}).Error("NAT-PMP port mapping failed")
		return 0, fmt.Errorf("NAT-PMP port mapping failed: %w", err)
	}

	externalPort := int(result.MappedExternalPort)
	log.WithFields(logger.Fields{
		"protocol":     protocol,
		"internalPort": internalPort,
		"externalPort": externalPort,
	}).Debug("NAT-PMP port mapped successfully")
	return externalPort, nil
}

// UnmapPort removes a port mapping via NAT-PMP.
func (n *NATPMPMapper) UnmapPort(protocol string, externalPort int) error {
	log.WithFields(logger.Fields{
		"protocol":     protocol,
		"externalPort": externalPort,
	}).Debug("unmapping port via NAT-PMP")

	// Validate port range to prevent invalid unmappings
	if externalPort < 1 || externalPort > 65535 {
		return fmt.Errorf("invalid port number: %d (must be 1-65535)", externalPort)
	}

	protocolStr := strings.ToUpper(protocol)
	if protocolStr != "TCP" && protocolStr != "UDP" {
		return fmt.Errorf("unsupported protocol: %s", protocol)
	}

	_, err := n.client.AddPortMapping(protocolStr, externalPort, 0, 0)
	if err != nil {
		log.WithError(err).WithFields(logger.Fields{
			"protocol":     protocol,
			"externalPort": externalPort,
		}).Error("NAT-PMP port unmapping failed")
		return fmt.Errorf("NAT-PMP port unmapping failed: %w", err)
	}

	log.WithFields(logger.Fields{
		"protocol":     protocol,
		"externalPort": externalPort,
	}).Debug("NAT-PMP port unmapped successfully")
	return nil
}

// GetExternalIP returns the external IP address via NAT-PMP.
func (n *NATPMPMapper) GetExternalIP() (string, error) {
	log.Debug("getting external IP via NAT-PMP")
	result, err := n.client.GetExternalAddress()
	if err != nil {
		log.WithError(err).Error("NAT-PMP external IP lookup failed")
		return "", fmt.Errorf("NAT-PMP external IP lookup failed: %w", err)
	}
	ip := net.IPv4(result.ExternalIPAddress[0], result.ExternalIPAddress[1],
		result.ExternalIPAddress[2], result.ExternalIPAddress[3])
	log.WithField("externalIP", ip.String()).Debug("NAT-PMP external IP retrieved")
	return ip.String(), nil
}
