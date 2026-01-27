//go:build windows

package nattraversal

import (
	"bufio"
	"net"
	"os/exec"
	"strings"
)

// readDefaultGateway reads the default gateway using `route print` on Windows.
// Returns nil, nil if the gateway cannot be determined (will use fallback).
func readDefaultGateway() (net.IP, error) {
	// Use route print to get the routing table
	// Filter for 0.0.0.0 to find default route
	cmd := exec.Command("route", "print", "0.0.0.0")
	output, err := cmd.Output()
	if err != nil {
		// Command failed, use fallback
		return nil, nil
	}

	return parseWindowsRouteOutput(string(output))
}

// parseWindowsRouteOutput parses the output of `route print 0.0.0.0` on Windows.
// The output format looks like:
//
// ===========================================================================
// IPv4 Route Table
// ===========================================================================
// Active Routes:
// Network Destination        Netmask          Gateway       Interface  Metric
//
//	0.0.0.0          0.0.0.0      192.168.1.1    192.168.1.100     25
//
// ===========================================================================
func parseWindowsRouteOutput(output string) (net.IP, error) {
	scanner := bufio.NewScanner(strings.NewReader(output))

	inActiveRoutes := false

	for scanner.Scan() {
		line := scanner.Text()

		// Detect when we're in the Active Routes section
		if strings.Contains(line, "Active Routes:") {
			inActiveRoutes = true
			continue
		}

		// Stop at the next separator or end
		if inActiveRoutes && strings.HasPrefix(line, "====") {
			break
		}

		if !inActiveRoutes {
			continue
		}

		fields := strings.Fields(line)

		// Skip header line and empty lines
		if len(fields) < 4 {
			continue
		}

		// Skip the header row
		if fields[0] == "Network" {
			continue
		}

		destination := fields[0]
		netmask := fields[1]

		// Look for default route: destination 0.0.0.0 with netmask 0.0.0.0
		if destination == "0.0.0.0" && netmask == "0.0.0.0" {
			gatewayStr := fields[2]

			// Skip "On-link" entries
			if gatewayStr == "On-link" {
				continue
			}

			gateway := net.ParseIP(gatewayStr)
			if gateway != nil && gateway.To4() != nil {
				return gateway.To4(), nil
			}
		}
	}

	return nil, nil // No default gateway found, use fallback
}
