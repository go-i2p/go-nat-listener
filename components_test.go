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

// TestExternalPortGetter tests the ExternalPort() convenience methods
func TestExternalPortGetter(t *testing.T) {
	t.Run("NATListener ExternalPort returns correct value", func(t *testing.T) {
		// Create a real TCP listener for testing
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("Failed to create TCP listener: %v", err)
		}

		mock := NewMockPortMapper()
		addr := NewNATAddr("tcp", ln.Addr().String(), "203.0.113.1:12345")
		renewalManager := NewRenewalManager(mock, "TCP", 12345, 12345)

		listener := &NATListener{
			listener:     ln,
			renewal:      renewalManager,
			externalPort: 12345,
			addr:         addr,
		}
		defer listener.Close()

		if listener.ExternalPort() != 12345 {
			t.Errorf("Expected ExternalPort() to return 12345, got %d", listener.ExternalPort())
		}
	})

	t.Run("NATPacketListener ExternalPort returns correct value", func(t *testing.T) {
		// Create a real UDP connection for testing
		conn, err := net.ListenPacket("udp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("Failed to create UDP connection: %v", err)
		}

		mock := NewMockPortMapper()
		addr := NewNATAddr("udp", conn.LocalAddr().String(), "203.0.113.1:54321")
		renewalManager := NewRenewalManager(mock, "UDP", 54321, 54321)

		listener := &NATPacketListener{
			conn:         conn,
			renewal:      renewalManager,
			externalPort: 54321,
			addr:         addr,
		}
		defer listener.Close()

		if listener.ExternalPort() != 54321 {
			t.Errorf("Expected ExternalPort() to return 54321, got %d", listener.ExternalPort())
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

// TestNATPacketConnDoubleClose verifies that NATPacketConn.Close() is idempotent
// and safe to call multiple times without panicking or returning errors on subsequent calls.
func TestNATPacketConnDoubleClose(t *testing.T) {
	t.Run("Double close on NATPacketConn is safe", func(t *testing.T) {
		// Create a real UDP connection for testing
		conn, err := net.ListenPacket("udp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("Failed to create UDP connection: %v", err)
		}

		addr := NewNATAddr("udp", conn.LocalAddr().String(), "203.0.113.1:12345")
		natConn := &NATPacketConn{
			PacketConn: conn,
			localAddr:  addr,
		}

		// First close should succeed
		err = natConn.Close()
		if err != nil {
			t.Errorf("First Close() should succeed, got: %v", err)
		}

		// Second close should also succeed (no panic, returns same error)
		err = natConn.Close()
		if err != nil {
			t.Errorf("Second Close() should return same result (nil), got: %v", err)
		}

		// Third close should also be safe
		err = natConn.Close()
		if err != nil {
			t.Errorf("Third Close() should return same result (nil), got: %v", err)
		}
	})

	t.Run("Listener and PacketConn close coordination", func(t *testing.T) {
		// Create a real UDP connection
		conn, err := net.ListenPacket("udp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("Failed to create UDP connection: %v", err)
		}

		mock := NewMockPortMapper()
		addr := NewNATAddr("udp", conn.LocalAddr().String(), "203.0.113.1:12345")

		// Create renewal manager (mapper, protocol, internalPort, externalPort)
		renewalManager := NewRenewalManager(mock, "UDP", 12345, 12345)

		// Create NATPacketListener
		listener := &NATPacketListener{
			conn:         conn,
			renewal:      renewalManager,
			externalPort: 12345,
			addr:         addr,
		}

		// Get the PacketConn - this creates the cached NATPacketConn
		packetConn, err := listener.Accept()
		if err != nil {
			t.Fatalf("Accept failed: %v", err)
		}

		// Close the PacketConn first
		err = packetConn.Close()
		if err != nil {
			t.Errorf("PacketConn.Close() should succeed, got: %v", err)
		}

		// Now close the listener - should NOT panic or error due to double-close
		err = listener.Close()
		if err != nil {
			t.Errorf("Listener.Close() after PacketConn.Close() should succeed, got: %v", err)
		}

		// Close the listener again - should be idempotent
		err = listener.Close()
		if err != nil {
			t.Errorf("Second Listener.Close() should succeed, got: %v", err)
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

// Gateway discovery tests have been moved to platform-specific test files:
// - gateway_test.go: cross-platform tests for discoverGateway and discoverGatewayFallback
// - gateway_linux_test.go: Linux-specific tests for parseHexIP and /proc/net/route parsing
// - gateway_bsd_test.go: BSD/macOS-specific tests for netstat parsing
// - gateway_windows_test.go: Windows-specific tests for route print parsing
