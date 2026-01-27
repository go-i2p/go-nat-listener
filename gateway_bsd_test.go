//go:build darwin || freebsd || openbsd || netbsd || dragonfly

package nattraversal

import (
	"net"
	"testing"
)

func TestParseNetstatOutput(t *testing.T) {
	testCases := []struct {
		name     string
		output   string
		expected net.IP
	}{
		{
			name: "macOS format with default",
			output: `Routing tables

Internet:
Destination        Gateway            Flags        Netif Expire
default            192.168.1.1        UGSc           en0
127.0.0.1          127.0.0.1          UH             lo0
192.168.1/24       link#4             UCS            en0
`,
			expected: net.IPv4(192, 168, 1, 1),
		},
		{
			name: "FreeBSD format with 0.0.0.0",
			output: `Routing tables

Internet:
Destination        Gateway            Flags    Refs      Use  Netif Expire
0.0.0.0            10.0.0.1           UGS         0        0    em0
10.0.0.0/24        link#1             U           0        0    em0
`,
			expected: net.IPv4(10, 0, 0, 1),
		},
		{
			name: "OpenBSD format",
			output: `Routing tables

Internet:
Destination        Gateway            Flags   Refs      Use   Mtu  Prio Iface
default            172.16.0.1         UGS        2    12345     -    12 em0
`,
			expected: net.IPv4(172, 16, 0, 1),
		},
		{
			name: "No default route",
			output: `Routing tables

Internet:
Destination        Gateway            Flags        Netif Expire
127.0.0.1          127.0.0.1          UH             lo0
`,
			expected: nil,
		},
		{
			name: "Link-local gateway (should skip)",
			output: `Routing tables

Internet:
Destination        Gateway            Flags        Netif Expire
default            link#5             UGSc           en0
192.168.1.1        192.168.1.1        UGSc           en0
`,
			expected: net.IPv4(192, 168, 1, 1),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gateway, err := parseNetstatOutput(tc.output)
			if err != nil {
				t.Fatalf("parseNetstatOutput failed: %v", err)
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

func TestReadDefaultGatewayBSD(t *testing.T) {
	// This test runs the actual netstat command
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
		t.Logf("Gateway from netstat: %v", gateway)
	} else {
		t.Log("No gateway found (may have no default route or netstat unavailable)")
	}
}
