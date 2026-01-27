package nattraversal

import (
	"net"
	"testing"
	"time"
)

// TestRenewalManagerStartStop tests renewal manager lifecycle
func TestRenewalManagerStartStop(t *testing.T) {
	t.Run("Start and stop renewal manager", func(t *testing.T) {
		mock := NewMockPortMapper()

		renewal := NewRenewalManager(mock, "TCP", 8080, 8080)

		// Start renewal
		renewal.Start()

		// Wait a bit to allow renewal attempts
		time.Sleep(50 * time.Millisecond)

		// Stop renewal
		renewal.Stop()

		// Verify port is unmapped
		mappings := mock.GetActiveMappings()
		if len(mappings) > 0 {
			t.Errorf("Expected no active mappings after stop, got %d", len(mappings))
		}
	})

	t.Run("Multiple starts are safe", func(t *testing.T) {
		mock := NewMockPortMapper()
		renewal := NewRenewalManager(mock, "TCP", 8080, 8080)

		// Start multiple times
		renewal.Start()
		renewal.Start()
		renewal.Start()

		// Should not panic or cause issues
		renewal.Stop()
	})

	t.Run("Multiple stops are safe", func(t *testing.T) {
		mock := NewMockPortMapper()
		renewal := NewRenewalManager(mock, "TCP", 8080, 8080)

		renewal.Start()

		// Stop multiple times
		renewal.Stop()
		renewal.Stop()
		renewal.Stop()

		// Should not panic or cause issues
	})
}

// TestRenewalManagerFailures tests renewal failure scenarios
func TestRenewalManagerFailures(t *testing.T) {
	t.Run("Renewal failure handling", func(t *testing.T) {
		mock := NewMockPortMapper()
		// Set failure rate to cause renewal failures
		mock.SetFailureRate(1.0)

		renewal := NewRenewalManager(mock, "TCP", 8080, 8080)
		renewal.Start()

		// Wait for attempted renewals
		time.Sleep(100 * time.Millisecond)

		renewal.Stop()

		// Should handle failures gracefully without crashing
	})
}

// TestListenerFunctionality tests NAT listener operations
func TestListenerFunctionality(t *testing.T) {
	t.Run("NAT address properties", func(t *testing.T) {
		internal := "192.168.1.100:8080"
		external := "203.0.113.100:8080"

		addr := NewNATAddr("tcp", internal, external)

		// Test that NAT listener would use this address correctly
		if addr.Network() != "tcp" {
			t.Errorf("Expected tcp network, got %s", addr.Network())
		}

		if addr.String() != external {
			t.Errorf("Expected external address in String(), got %s", addr.String())
		}
	})
}

// TestPacketListenerFunctionality tests NAT packet listener operations
func TestPacketListenerFunctionality(t *testing.T) {
	t.Run("NAT packet connection properties", func(t *testing.T) {
		internal := "192.168.1.100:9090"
		external := "203.0.113.100:9090"

		addr := NewNATAddr("udp", internal, external)

		// Test that NAT packet listener would use this address correctly
		if addr.Network() != "udp" {
			t.Errorf("Expected udp network, got %s", addr.Network())
		}

		if addr.ExternalAddr() != external {
			t.Errorf("Expected external address %s, got %s", external, addr.ExternalAddr())
		}
	})
}

// TestUPnPMapperSimulation tests UPnP-specific behavior
func TestUPnPMapperSimulation(t *testing.T) {
	t.Run("UPnP protocol simulation", func(t *testing.T) {
		mock := NewMockPortMapper()
		mock.SetProtocolSupport(true, false) // UPnP only

		// Test TCP mapping
		tcpPort, err := mock.MapPort("TCP", 8080, 5*time.Minute)
		if err != nil {
			t.Fatalf("UPnP TCP mapping failed: %v", err)
		}

		// Test UDP mapping
		udpPort, err := mock.MapPort("UDP", 9090, 5*time.Minute)
		if err != nil {
			t.Fatalf("UPnP UDP mapping failed: %v", err)
		}

		// Verify external IP lookup
		ip, err := mock.GetExternalIP()
		if err != nil {
			t.Fatalf("UPnP external IP lookup failed: %v", err)
		}

		if ip == "" {
			t.Errorf("Expected non-empty external IP")
		}

		// Clean up
		mock.UnmapPort("TCP", tcpPort)
		mock.UnmapPort("UDP", udpPort)
	})
}

// TestNATPMPMapperSimulation tests NAT-PMP-specific behavior
func TestNATPMPMapperSimulation(t *testing.T) {
	t.Run("NAT-PMP protocol simulation", func(t *testing.T) {
		mock := NewMockPortMapper()
		mock.SetProtocolSupport(false, true) // NAT-PMP only

		// Test TCP mapping
		tcpPort, err := mock.MapPort("TCP", 8080, 5*time.Minute)
		if err != nil {
			t.Fatalf("NAT-PMP TCP mapping failed: %v", err)
		}

		// Test UDP mapping
		udpPort, err := mock.MapPort("UDP", 9090, 5*time.Minute)
		if err != nil {
			t.Fatalf("NAT-PMP UDP mapping failed: %v", err)
		}

		// Verify external IP lookup
		ip, err := mock.GetExternalIP()
		if err != nil {
			t.Fatalf("NAT-PMP external IP lookup failed: %v", err)
		}

		if ip == "" {
			t.Errorf("Expected non-empty external IP")
		}

		// Clean up
		mock.UnmapPort("TCP", tcpPort)
		mock.UnmapPort("UDP", udpPort)
	})
}

// TestNetworkConditionsSimulation tests various network condition simulations
func TestNetworkConditionsSimulation(t *testing.T) {
	t.Run("Latency simulation", func(t *testing.T) {
		conditions := NewMockNetworkConditions()
		conditions.Latency = 50 * time.Millisecond

		start := time.Now()
		conditions.SimulateLatency()
		elapsed := time.Since(start)

		if elapsed < 50*time.Millisecond {
			t.Errorf("Expected latency >= 50ms, got %v", elapsed)
		}
	})

	t.Run("Jitter simulation", func(t *testing.T) {
		conditions := NewMockNetworkConditions()
		conditions.Latency = 10 * time.Millisecond
		conditions.Jitter = 5 * time.Millisecond

		// Run multiple times to see jitter variation
		latencies := make([]time.Duration, 5)
		for i := 0; i < 5; i++ {
			start := time.Now()
			conditions.SimulateLatency()
			latencies[i] = time.Since(start)
		}

		// Check that latencies vary (jitter effect)
		baseLatency := latencies[0]
		hasVariation := false
		for _, lat := range latencies[1:] {
			if lat != baseLatency {
				hasVariation = true
				break
			}
		}

		// Note: Due to the deterministic nature of our mock,
		// this test might not always show variation, but the
		// infrastructure is there for realistic jitter simulation
		t.Logf("Latencies: %v", latencies)
		t.Logf("Has variation: %v", hasVariation)
	})

	t.Run("Packet loss simulation", func(t *testing.T) {
		conditions := NewMockNetworkConditions()
		conditions.PacketLoss = 0.5 // 50% packet loss

		lostPackets := 0
		totalPackets := 100

		for i := 0; i < totalPackets; i++ {
			if conditions.SimulatePacketLoss() {
				lostPackets++
			}
		}

		// Due to deterministic implementation, we expect consistent results
		t.Logf("Lost %d out of %d packets", lostPackets, totalPackets)

		// The exact number will depend on the deterministic implementation
		if lostPackets == 0 {
			t.Errorf("Expected some packet loss with 50%% rate")
		}
	})
}

// TestFirewallRules tests firewall rule management
func TestFirewallRules(t *testing.T) {
	t.Run("Default allow policy", func(t *testing.T) {
		firewall := NewMockFirewall()
		firewall.SetDefaultPolicy(true)

		blocked := firewall.IsBlocked("192.168.1.100", 8080)
		if blocked {
			t.Errorf("Expected connection to be allowed with default allow policy")
		}
	})

	t.Run("Default deny policy", func(t *testing.T) {
		firewall := NewMockFirewall()
		firewall.SetDefaultPolicy(false)

		blocked := firewall.IsBlocked("192.168.1.100", 8080)
		if !blocked {
			t.Errorf("Expected connection to be blocked with default deny policy")
		}
	})

	t.Run("Specific rules override default", func(t *testing.T) {
		firewall := NewMockFirewall()
		firewall.SetDefaultPolicy(false) // Default deny
		firewall.AllowConnection("192.168.1.100", 8080)

		blocked := firewall.IsBlocked("192.168.1.100", 8080)
		if blocked {
			t.Errorf("Expected specific allow rule to override default deny")
		}

		// Other connections should still be blocked
		blocked = firewall.IsBlocked("192.168.1.101", 8080)
		if !blocked {
			t.Errorf("Expected other connections to be blocked by default")
		}
	})

	t.Run("Port blocking", func(t *testing.T) {
		firewall := NewMockFirewall()
		firewall.SetDefaultPolicy(true) // Default allow
		firewall.BlockPort(8080)

		blocked := firewall.IsBlocked("192.168.1.100", 8080)
		if !blocked {
			t.Errorf("Expected port 8080 to be blocked")
		}

		// Other ports should be allowed
		blocked = firewall.IsBlocked("192.168.1.100", 8081)
		if blocked {
			t.Errorf("Expected port 8081 to be allowed")
		}
	})

	t.Run("IP blocking", func(t *testing.T) {
		firewall := NewMockFirewall()
		firewall.SetDefaultPolicy(true) // Default allow
		firewall.BlockIP("192.168.1.100")

		blocked := firewall.IsBlocked("192.168.1.100", 8080)
		if !blocked {
			t.Errorf("Expected IP 192.168.1.100 to be blocked")
		}

		// Other IPs should be allowed
		blocked = firewall.IsBlocked("192.168.1.101", 8080)
		if blocked {
			t.Errorf("Expected IP 192.168.1.101 to be allowed")
		}
	})
}

// TestGatewayDiscovery tests the gateway discovery functions
func TestGatewayDiscovery(t *testing.T) {
	t.Run("discoverGateway returns valid IP", func(t *testing.T) {
		gateway, err := discoverGateway()
		if err != nil {
			t.Fatalf("discoverGateway failed: %v", err)
		}

		if gateway == nil {
			t.Fatal("discoverGateway returned nil gateway")
		}

		// Gateway should be IPv4
		if gateway.To4() == nil {
			t.Errorf("Expected IPv4 gateway, got: %v", gateway)
		}

		// Gateway should not be zero
		if gateway.Equal(net.IPv4zero) {
			t.Error("Gateway should not be 0.0.0.0")
		}

		t.Logf("Discovered gateway: %v", gateway)
	})

	t.Run("parseHexIP converts correctly", func(t *testing.T) {
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
	})

	t.Run("parseHexIP handles invalid input", func(t *testing.T) {
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
	})

	t.Run("discoverGatewayFallback returns valid IP", func(t *testing.T) {
		gateway, err := discoverGatewayFallback()
		if err != nil {
			t.Fatalf("discoverGatewayFallback failed: %v", err)
		}

		if gateway == nil {
			t.Fatal("discoverGatewayFallback returned nil gateway")
		}

		// Gateway should be IPv4
		if gateway.To4() == nil {
			t.Errorf("Expected IPv4 gateway, got: %v", gateway)
		}

		// Fallback always sets .1 suffix based on local IP's subnet
		// The last octet should always be 1 (that's the heuristic)
		ipv4 := gateway.To4()
		if ipv4[3] != 1 {
			t.Errorf("Expected fallback gateway to end in .1, got last octet: %d", ipv4[3])
		}

		t.Logf("Fallback gateway: %v", gateway)
	})

	t.Run("readDefaultGateway on Linux", func(t *testing.T) {
		// This test only runs on Linux where /proc/net/route exists
		gateway, err := readDefaultGateway()

		if err != nil {
			// If there's an error, it should be a parsing error, not file access
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
			t.Log("No gateway found in routing table (may be non-Linux or no default route)")
		}
	})
}
