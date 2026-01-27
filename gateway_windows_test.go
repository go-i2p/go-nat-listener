//go:build windows

package nattraversal

import (
	"net"
	"testing"
)

func TestParseWindowsRouteOutput(t *testing.T) {
	testCases := []struct {
		name     string
		output   string
		expected net.IP
	}{
		{
			name: "Standard Windows route output",
			output: `===========================================================================
Interface List
 12...00 1c 42 a7 b3 c5 ......Intel(R) Ethernet Connection
===========================================================================

IPv4 Route Table
===========================================================================
Active Routes:
Network Destination        Netmask          Gateway       Interface  Metric
          0.0.0.0          0.0.0.0      192.168.1.1    192.168.1.100     25
        127.0.0.0        255.0.0.0         On-link         127.0.0.1    331
      192.168.1.0    255.255.255.0         On-link     192.168.1.100    281
===========================================================================
Persistent Routes:
  None
`,
			expected: net.IPv4(192, 168, 1, 1),
		},
		{
			name: "Multiple interfaces",
			output: `===========================================================================
IPv4 Route Table
===========================================================================
Active Routes:
Network Destination        Netmask          Gateway       Interface  Metric
          0.0.0.0          0.0.0.0       10.0.0.1       10.0.0.50     35
          0.0.0.0          0.0.0.0      172.16.0.1     172.16.0.50     55
===========================================================================
`,
			expected: net.IPv4(10, 0, 0, 1), // First match wins
		},
		{
			name: "On-link only (no gateway)",
			output: `===========================================================================
IPv4 Route Table
===========================================================================
Active Routes:
Network Destination        Netmask          Gateway       Interface  Metric
          0.0.0.0          0.0.0.0         On-link     192.168.1.100    281
===========================================================================
`,
			expected: nil,
		},
		{
			name: "No default route",
			output: `===========================================================================
IPv4 Route Table
===========================================================================
Active Routes:
Network Destination        Netmask          Gateway       Interface  Metric
        127.0.0.0        255.0.0.0         On-link         127.0.0.1    331
===========================================================================
`,
			expected: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gateway, err := parseWindowsRouteOutput(tc.output)
			if err != nil {
				t.Fatalf("parseWindowsRouteOutput failed: %v", err)
			}

			if tc.expected == nil {
				if gateway != nil {
					t.Errorf("Expected nil gateway, got %v", gateway)
				}
			} else {
				if gateway == nil {
					t.Errorf("Expected gateway %v, got nil", tc.expected)
				} else if !gateway.Equal(tc.expected) {
					t.Errorf("Expected gateway %v, got %v", tc.expected, gateway)
				}
			}
		})
	}
}

func TestReadDefaultGatewayWindows(t *testing.T) {
	// This test runs the actual route command
	gateway, err := readDefaultGateway()
	if err != nil {
		t.Logf("readDefaultGateway returned error: %v", err)
	}

	if gateway != nil {
		// If we got a gateway, it should be valid IPv4
		if gateway.To4() == nil {
			t.Errorf("Expected IPv4 gateway, got: %v", gateway)
		}
		if gateway.Equal(net.IPv4zero) {
			t.Error("Gateway should not be 0.0.0.0")
		}
		t.Logf("Gateway from route print: %v", gateway)
	} else {
		t.Log("No gateway found (may have no default route)")
	}
}
