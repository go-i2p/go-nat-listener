//go:build linux

package nattraversal

import (
	"net"
	"testing"
)

func TestParseHexIP(t *testing.T) {
	testCases := []struct {
		hexIP    string
		expected net.IP
	}{
		{"0101A8C0", net.IPv4(192, 168, 1, 1)},     // 192.168.1.1
		{"FE01A8C0", net.IPv4(192, 168, 1, 254)},   // 192.168.1.254
		{"01000A0A", net.IPv4(10, 10, 0, 1)},       // 10.10.0.1
		{"00000000", net.IPv4(0, 0, 0, 0)},         // 0.0.0.0
		{"FFFFFFFF", net.IPv4(255, 255, 255, 255)}, // 255.255.255.255
	}

	for _, tc := range testCases {
		ip, err := parseHexIP(tc.hexIP)
		if err != nil {
			t.Errorf("parseHexIP(%s) failed: %v", tc.hexIP, err)
			continue
		}

		if !ip.Equal(tc.expected) {
			t.Errorf("parseHexIP(%s) = %v, expected %v", tc.hexIP, ip, tc.expected)
		}
	}
}

func TestParseHexIPInvalid(t *testing.T) {
	invalidInputs := []string{
		"",           // empty
		"0101A8",     // too short
		"0101A8C0FF", // too long
		"ZZZZZZZZ",   // invalid hex
	}

	for _, input := range invalidInputs {
		_, err := parseHexIP(input)
		if err == nil {
			t.Errorf("parseHexIP(%q) should have failed", input)
		}
	}
}

func TestReadDefaultGatewayLinux(t *testing.T) {
	// This test reads the actual /proc/net/route on Linux
	gateway, err := readDefaultGateway()
	if err != nil {
		// If there's an error, it should be a parsing error
		t.Logf("readDefaultGateway returned error: %v", err)
	}

	if gateway != nil {
		// If we got a gateway, it should be valid IPv4
		if gateway.To4() == nil {
			t.Errorf("Expected IPv4 gateway from routing table, got: %v", gateway)
		}
		if gateway.Equal(net.IPv4zero) {
			t.Error("Gateway from routing table should not be 0.0.0.0")
		}
		t.Logf("Gateway from routing table: %v", gateway)
	} else {
		t.Log("No gateway found in routing table (may have no default route)")
	}
}
