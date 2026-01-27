//go:build darwin || freebsd || openbsd || netbsd || dragonfly

package nattraversal

import (
	"bufio"
	"net"
	"os/exec"
	"strings"
)

// readDefaultGateway reads the default gateway using netstat on BSD-like systems.
// This includes macOS (darwin), FreeBSD, OpenBSD, NetBSD, and DragonFly BSD.
// Returns nil, nil if the gateway cannot be determined (will use fallback).
func readDefaultGateway() (net.IP, error) {
	// Use netstat -rn to get the routing table
	// -r: show routing table
	// -n: show numerical addresses (don't resolve hostnames)
	cmd := exec.Command("netstat", "-rn")
	output, err := cmd.Output()
	if err != nil {
		// Command failed, use fallback
		return nil, nil
	}

	return parseNetstatOutput(string(output))
}

// parseNetstatOutput parses the output of `netstat -rn` to find the default gateway.
// The format varies slightly between BSD variants, but the general structure is:
//
//	Destination        Gateway            Flags    ...
//	default            192.168.1.1        UGS      ...
//	0.0.0.0            192.168.1.1        UGS      ...
func parseNetstatOutput(output string) (net.IP, error) {
	scanner := bufio.NewScanner(strings.NewReader(output))

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)

		if len(fields) < 2 {
			continue
		}

		destination := fields[0]

		// Look for default route (marked as "default" or "0.0.0.0")
		if destination == "default" || destination == "0.0.0.0" || destination == "0.0.0.0/0" {
			gatewayStr := fields[1]

			// Skip link-local or interface names (e.g., "link#5", "en0")
			if strings.Contains(gatewayStr, "#") || !strings.Contains(gatewayStr, ".") {
				continue
			}

			// Handle gateway with %interface suffix (e.g., "192.168.1.1%en0")
			if idx := strings.Index(gatewayStr, "%"); idx != -1 {
				gatewayStr = gatewayStr[:idx]
			}

			gateway := net.ParseIP(gatewayStr)
			if gateway != nil && gateway.To4() != nil {
				return gateway.To4(), nil
			}
		}
	}

	return nil, nil // No default gateway found, use fallback
}
