package nattraversal

import (
	"fmt"
	"net"
	"sync"
	"testing"
	"time"
)

// TestPortMappingCreationAndDeletion tests basic port mapping operations
func TestPortMappingCreationAndDeletion(t *testing.T) {
	tests := []struct {
		name     string
		protocol string
		port     int
		duration time.Duration
		wantErr  bool
	}{
		{
			name:     "TCP mapping success",
			protocol: "TCP",
			port:     8080,
			duration: 5 * time.Minute,
			wantErr:  false,
		},
		{
			name:     "UDP mapping success",
			protocol: "UDP",
			port:     9090,
			duration: 5 * time.Minute,
			wantErr:  false,
		},
		{
			name:     "Invalid protocol",
			protocol: "SCTP",
			port:     8080,
			duration: 5 * time.Minute,
			wantErr:  true,
		},
		{
			name:     "Port below valid range",
			protocol: "TCP",
			port:     0,
			duration: 5 * time.Minute,
			wantErr:  true,
		},
		{
			name:     "Negative port number",
			protocol: "TCP",
			port:     -1,
			duration: 5 * time.Minute,
			wantErr:  true,
		},
		{
			name:     "Port above valid range",
			protocol: "TCP",
			port:     65536,
			duration: 5 * time.Minute,
			wantErr:  true,
		},
		{
			name:     "Port at lower boundary",
			protocol: "TCP",
			port:     1,
			duration: 5 * time.Minute,
			wantErr:  false,
		},
		{
			name:     "Port at upper boundary",
			protocol: "TCP",
			port:     65535,
			duration: 5 * time.Minute,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock port mapper
			mock := NewMockPortMapper()

			// Test port mapping creation
			externalPort, err := mock.MapPort(tt.protocol, tt.port, tt.duration)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error for protocol %s, but got none", tt.protocol)
				}
				return
			}

			if err != nil {
				t.Fatalf("Failed to create port mapping: %v", err)
			}

			if externalPort <= 0 {
				t.Errorf("Expected positive external port, got %d", externalPort)
			}

			// Verify mapping exists
			mappings := mock.GetActiveMappings()
			key := fmt.Sprintf("%s:%d", tt.protocol, externalPort)
			mapping, exists := mappings[key]
			if !exists {
				t.Errorf("Expected mapping %s to exist", key)
			}

			if mapping.InternalPort != tt.port {
				t.Errorf("Expected internal port %d, got %d", tt.port, mapping.InternalPort)
			}

			// Test port mapping deletion
			err = mock.UnmapPort(tt.protocol, externalPort)
			if err != nil {
				t.Fatalf("Failed to delete port mapping: %v", err)
			}

			// Verify mapping no longer exists
			mappings = mock.GetActiveMappings()
			if _, exists := mappings[key]; exists {
				t.Errorf("Expected mapping %s to be deleted", key)
			}
		})
	}
}

// TestPublicIPDiscovery tests external IP address discovery
func TestPublicIPDiscovery(t *testing.T) {
	tests := []struct {
		name       string
		externalIP string
		wantErr    bool
	}{
		{
			name:       "Valid IPv4 address",
			externalIP: "203.0.113.100",
			wantErr:    false,
		},
		{
			name:       "Valid IPv6 address",
			externalIP: "2001:db8::1",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockPortMapper()
			mock.SetExternalIP(tt.externalIP)

			ip, err := mock.GetExternalIP()

			if tt.wantErr && err == nil {
				t.Errorf("Expected error, but got none")
				return
			}

			if !tt.wantErr && err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if ip != tt.externalIP {
				t.Errorf("Expected IP %s, got %s", tt.externalIP, ip)
			}
		})
	}
}

// TestKeepAlivePacketTransmission tests keep-alive functionality
func TestKeepAlivePacketTransmission(t *testing.T) {
	t.Run("Successful keep-alive", func(t *testing.T) {
		// Create mock UDP addresses
		localAddr, _ := net.ResolveUDPAddr("udp", "192.168.1.100:8080")
		remoteAddr, _ := net.ResolveUDPAddr("udp", "203.0.113.100:8080")

		// Create mock connection
		conn := NewMockUDPConn(localAddr, remoteAddr)

		// Simulate keep-alive packet
		keepAliveData := []byte("KEEPALIVE")
		n, err := conn.Write(keepAliveData)

		if err != nil {
			t.Fatalf("Failed to send keep-alive: %v", err)
		}

		if n != len(keepAliveData) {
			t.Errorf("Expected to write %d bytes, wrote %d", len(keepAliveData), n)
		}

		// Verify data was written
		written := conn.GetWrittenData()
		if len(written) != 1 {
			t.Errorf("Expected 1 packet written, got %d", len(written))
		}

		if string(written[0]) != "KEEPALIVE" {
			t.Errorf("Expected KEEPALIVE packet, got %s", string(written[0]))
		}
	})

	t.Run("Keep-alive with packet loss", func(t *testing.T) {
		localAddr, _ := net.ResolveUDPAddr("udp", "192.168.1.100:8080")
		remoteAddr, _ := net.ResolveUDPAddr("udp", "203.0.113.100:8080")

		conn := NewMockUDPConn(localAddr, remoteAddr)

		// Set high packet loss rate
		conditions := NewMockNetworkConditions()
		conditions.PacketLoss = 1.0 // 100% packet loss
		conn.SetNetworkConditions(conditions)

		keepAliveData := []byte("KEEPALIVE")
		_, err := conn.Write(keepAliveData)

		if err == nil {
			t.Errorf("Expected packet loss error, but got none")
		}

		if err.Error() != "packet lost" {
			t.Errorf("Expected 'packet lost' error, got '%s'", err.Error())
		}
	})
}

// TestConnectionEstablishmentThroughNAT tests NAT traversal connection establishment
func TestConnectionEstablishmentThroughNAT(t *testing.T) {
	tests := []struct {
		name    string
		natType NATType
		wantErr bool
	}{
		{
			name:    "Full Cone NAT",
			natType: FullConeNAT,
			wantErr: false,
		},
		{
			name:    "Restricted NAT",
			natType: RestrictedNAT,
			wantErr: false,
		},
		{
			name:    "Port Restricted NAT",
			natType: PortRestrictedNAT,
			wantErr: false,
		},
		{
			name:    "Symmetric NAT",
			natType: SymmetricNAT,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockPortMapper()
			mock.SetNATType(tt.natType)

			// Create port mapping for connection
			internalPort := 8080
			externalPort, err := mock.MapPort("TCP", internalPort, 5*time.Minute)

			if err != nil {
				t.Fatalf("Failed to create port mapping: %v", err)
			}

			// Verify external port assignment based on NAT type
			switch tt.natType {
			case FullConeNAT:
				if externalPort != internalPort {
					t.Errorf("Full Cone NAT should use same port, got internal=%d, external=%d",
						internalPort, externalPort)
				}
			case RestrictedNAT, PortRestrictedNAT:
				expectedPort := internalPort + 1000
				if externalPort != expectedPort {
					t.Errorf("Restricted NAT should use port+1000, expected=%d, got=%d",
						expectedPort, externalPort)
				}
			case SymmetricNAT:
				if externalPort == internalPort {
					t.Errorf("Symmetric NAT should use different port, got same port %d", externalPort)
				}
			}

			// Verify external IP is accessible
			externalIP, err := mock.GetExternalIP()
			if err != nil {
				t.Fatalf("Failed to get external IP: %v", err)
			}

			if externalIP == "" {
				t.Errorf("Expected non-empty external IP")
			}
		})
	}
}

// TestConnectionTimeoutsAndFailures tests error handling scenarios
func TestConnectionTimeoutsAndFailures(t *testing.T) {
	t.Run("Network latency simulation", func(t *testing.T) {
		mock := NewMockPortMapper()
		mock.SetLatency(100 * time.Millisecond)

		start := time.Now()
		_, err := mock.MapPort("TCP", 8080, 5*time.Minute)
		elapsed := time.Since(start)

		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if elapsed < 100*time.Millisecond {
			t.Errorf("Expected latency >= 100ms, got %v", elapsed)
		}
	})

	t.Run("Random failure simulation", func(t *testing.T) {
		mock := NewMockPortMapper()
		mock.SetFailureRate(1.0) // 100% failure rate

		_, err := mock.MapPort("TCP", 8080, 5*time.Minute)

		if err == nil {
			t.Errorf("Expected failure with 100%% failure rate")
		}

		if err.Error() != "mock: random failure occurred" {
			t.Errorf("Expected specific failure message, got: %s", err.Error())
		}
	})

	t.Run("Connection timeout", func(t *testing.T) {
		localAddr, _ := net.ResolveUDPAddr("udp", "192.168.1.100:8080")
		remoteAddr, _ := net.ResolveUDPAddr("udp", "203.0.113.100:8080")

		conn := NewMockUDPConn(localAddr, remoteAddr)

		// Set very high latency to simulate timeout
		conditions := NewMockNetworkConditions()
		conditions.Latency = 1 * time.Second
		conn.SetNetworkConditions(conditions)

		start := time.Now()
		_, err := conn.Write([]byte("test"))
		elapsed := time.Since(start)

		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if elapsed < 1*time.Second {
			t.Errorf("Expected delay >= 1s due to latency, got %v", elapsed)
		}
	})
}

// TestRouterProtocolNotSupported tests unsupported protocol scenarios
func TestRouterProtocolNotSupported(t *testing.T) {
	t.Run("No protocols supported", func(t *testing.T) {
		mock := NewMockPortMapper()
		mock.SetProtocolSupport(false, false) // Disable both UPnP and NAT-PMP

		_, err := mock.MapPort("TCP", 8080, 5*time.Minute)

		if err == nil {
			t.Errorf("Expected error when no protocols are supported")
		}

		if err.Error() != "mock: no protocols supported" {
			t.Errorf("Expected 'no protocols supported' error, got: %s", err.Error())
		}
	})

	t.Run("UPnP only", func(t *testing.T) {
		mock := NewMockPortMapper()
		mock.SetProtocolSupport(true, false) // Only UPnP

		_, err := mock.MapPort("TCP", 8080, 5*time.Minute)

		if err != nil {
			t.Errorf("Expected success with UPnP support, got: %v", err)
		}
	})

	t.Run("NAT-PMP only", func(t *testing.T) {
		mock := NewMockPortMapper()
		mock.SetProtocolSupport(false, true) // Only NAT-PMP

		_, err := mock.MapPort("TCP", 8080, 5*time.Minute)

		if err != nil {
			t.Errorf("Expected success with NAT-PMP support, got: %v", err)
		}
	})
}

// TestNATMappingChanges tests mid-connection mapping changes
func TestNATMappingChanges(t *testing.T) {
	t.Run("Mapping change during connection", func(t *testing.T) {
		mock := NewMockPortMapper()

		// Create initial mapping
		internalPort := 8080
		externalPort, err := mock.MapPort("TCP", internalPort, 5*time.Minute)
		if err != nil {
			t.Fatalf("Failed to create initial mapping: %v", err)
		}

		// Verify initial mapping exists
		mappings := mock.GetActiveMappings()
		oldKey := fmt.Sprintf("TCP:%d", externalPort)
		if _, exists := mappings[oldKey]; !exists {
			t.Errorf("Initial mapping should exist")
		}

		// Simulate mapping change
		newExternalPort := externalPort + 100
		mock.SimulateMappingChange("TCP", externalPort, newExternalPort)

		// Verify old mapping is gone and new one exists
		mappings = mock.GetActiveMappings()
		if _, exists := mappings[oldKey]; exists {
			t.Errorf("Old mapping should be removed")
		}

		newKey := fmt.Sprintf("TCP:%d", newExternalPort)
		if mapping, exists := mappings[newKey]; !exists {
			t.Errorf("New mapping should exist")
		} else if mapping.ExternalPort != newExternalPort {
			t.Errorf("Expected new external port %d, got %d", newExternalPort, mapping.ExternalPort)
		}
	})

	t.Run("Mapping expiration", func(t *testing.T) {
		mock := NewMockPortMapper()

		// Create mapping with short duration
		externalPort, err := mock.MapPort("TCP", 8080, 1*time.Millisecond)
		if err != nil {
			t.Fatalf("Failed to create mapping: %v", err)
		}

		// Wait for expiration
		time.Sleep(2 * time.Millisecond)

		// Force expiration check
		mock.ExpireMapping("TCP", externalPort)

		// Verify mapping is considered expired
		mappings := mock.GetActiveMappings()
		key := fmt.Sprintf("TCP:%d", externalPort)
		if _, exists := mappings[key]; exists {
			t.Errorf("Expired mapping should not be active")
		}
	})
}

// TestFirewallBlockingScenarios tests firewall interference
func TestFirewallBlockingScenarios(t *testing.T) {
	t.Run("Port blocked by firewall", func(t *testing.T) {
		localAddr, _ := net.ResolveUDPAddr("udp", "192.168.1.100:8080")
		remoteAddr, _ := net.ResolveUDPAddr("udp", "203.0.113.100:9090")

		conn := NewMockUDPConn(localAddr, remoteAddr)

		// Setup firewall to block the remote port
		firewall := NewMockFirewall()
		firewall.BlockPort(9090)
		conn.SetFirewall(firewall)

		_, err := conn.Write([]byte("test"))

		if err == nil {
			t.Errorf("Expected firewall to block connection")
		}

		if err.Error() != "connection blocked by firewall" {
			t.Errorf("Expected firewall block error, got: %s", err.Error())
		}
	})

	t.Run("IP blocked by firewall", func(t *testing.T) {
		localAddr, _ := net.ResolveUDPAddr("udp", "192.168.1.100:8080")
		remoteAddr, _ := net.ResolveUDPAddr("udp", "203.0.113.100:9090")

		conn := NewMockUDPConn(localAddr, remoteAddr)

		// Setup firewall to block the remote IP
		firewall := NewMockFirewall()
		firewall.BlockIP("203.0.113.100")
		conn.SetFirewall(firewall)

		_, err := conn.Write([]byte("test"))

		if err == nil {
			t.Errorf("Expected firewall to block connection")
		}
	})

	t.Run("Firewall allows specific connection", func(t *testing.T) {
		localAddr, _ := net.ResolveUDPAddr("udp", "192.168.1.100:8080")
		remoteAddr, _ := net.ResolveUDPAddr("udp", "203.0.113.100:9090")

		conn := NewMockUDPConn(localAddr, remoteAddr)

		// Setup firewall with default deny but allow specific connection
		firewall := NewMockFirewall()
		firewall.SetDefaultPolicy(false) // Default deny
		firewall.AllowConnection("203.0.113.100", 9090)
		conn.SetFirewall(firewall)

		_, err := conn.Write([]byte("test"))

		if err != nil {
			t.Errorf("Expected firewall to allow specific connection, got: %v", err)
		}
	})
}

// TestPortExhaustion tests port exhaustion scenarios
func TestPortExhaustion(t *testing.T) {
	t.Run("Port exhaustion error", func(t *testing.T) {
		mock := NewMockPortMapper()
		mock.SetPortExhaustion(true)

		_, err := mock.MapPort("TCP", 8080, 5*time.Minute)

		if err == nil {
			t.Errorf("Expected port exhaustion error")
		}

		if err.Error() != "mock: no available ports" {
			t.Errorf("Expected port exhaustion error, got: %s", err.Error())
		}
	})

	t.Run("Multiple mappings until exhaustion", func(t *testing.T) {
		mock := NewMockPortMapper()

		// Create several successful mappings
		var ports []int
		for i := 0; i < 5; i++ {
			port, err := mock.MapPort("TCP", 8080+i, 5*time.Minute)
			if err != nil {
				t.Fatalf("Failed to create mapping %d: %v", i, err)
			}
			ports = append(ports, port)
		}

		// Verify all mappings exist
		mappings := mock.GetActiveMappings()
		if len(mappings) != 5 {
			t.Errorf("Expected 5 active mappings, got %d", len(mappings))
		}

		// Enable port exhaustion
		mock.SetPortExhaustion(true)

		// Next mapping should fail
		_, err := mock.MapPort("TCP", 8085, 5*time.Minute)
		if err == nil {
			t.Errorf("Expected port exhaustion after multiple mappings")
		}
	})
}

// TestConcurrentOperations tests thread safety
func TestConcurrentOperations(t *testing.T) {
	t.Run("Concurrent port mappings", func(t *testing.T) {
		mock := NewMockPortMapper()

		const numGoroutines = 10
		var wg sync.WaitGroup
		errors := make(chan error, numGoroutines)
		ports := make(chan int, numGoroutines)

		// Launch concurrent mapping operations
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(portOffset int) {
				defer wg.Done()

				port, err := mock.MapPort("TCP", 8080+portOffset, 5*time.Minute)
				if err != nil {
					errors <- err
					return
				}
				ports <- port
			}(i)
		}

		wg.Wait()
		close(errors)
		close(ports)

		// Check for errors
		for err := range errors {
			t.Errorf("Concurrent operation failed: %v", err)
		}

		// Count successful mappings
		var portCount int
		for range ports {
			portCount++
		}

		if portCount != numGoroutines {
			t.Errorf("Expected %d successful mappings, got %d", numGoroutines, portCount)
		}

		// Verify mappings exist
		mappings := mock.GetActiveMappings()
		if len(mappings) != numGoroutines {
			t.Errorf("Expected %d active mappings, got %d", numGoroutines, len(mappings))
		}
	})

	t.Run("Concurrent mapping and unmapping", func(t *testing.T) {
		mock := NewMockPortMapper()

		const numOperations = 20
		var wg sync.WaitGroup

		// Alternate between mapping and unmapping operations
		for i := 0; i < numOperations; i++ {
			wg.Add(1)
			if i%2 == 0 {
				// Mapping operation
				go func(portOffset int) {
					defer wg.Done()
					_, err := mock.MapPort("TCP", 8080+portOffset, 5*time.Minute)
					if err != nil {
						t.Errorf("Mapping failed: %v", err)
					}
				}(i)
			} else {
				// Unmapping operation (may fail if port doesn't exist)
				go func(portOffset int) {
					defer wg.Done()
					_ = mock.UnmapPort("TCP", 8080+portOffset)
				}(i)
			}
		}

		wg.Wait()

		// Final state should have some mappings but not necessarily all
		mappings := mock.GetActiveMappings()
		t.Logf("Final active mappings: %d", len(mappings))
	})
}

// TestNATAddrFunctionality tests NAT address functionality
func TestNATAddrFunctionality(t *testing.T) {
	t.Run("NAT address creation and methods", func(t *testing.T) {
		internal := "192.168.1.100:8080"
		external := "203.0.113.100:8080"
		network := "tcp"

		addr := NewNATAddr(network, internal, external)

		if addr.Network() != network {
			t.Errorf("Expected network %s, got %s", network, addr.Network())
		}

		if addr.InternalAddr() != internal {
			t.Errorf("Expected internal addr %s, got %s", internal, addr.InternalAddr())
		}

		if addr.ExternalAddr() != external {
			t.Errorf("Expected external addr %s, got %s", external, addr.ExternalAddr())
		}

		if addr.String() != external {
			t.Errorf("Expected String() to return external addr %s, got %s", external, addr.String())
		}
	})
}

// BenchmarkPortMapping benchmarks port mapping operations
func BenchmarkPortMapping(b *testing.B) {
	mock := NewMockPortMapper()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		port, err := mock.MapPort("TCP", 8080+i%1000, 5*time.Minute)
		if err != nil {
			b.Fatalf("Mapping failed: %v", err)
		}

		err = mock.UnmapPort("TCP", port)
		if err != nil {
			b.Fatalf("Unmapping failed: %v", err)
		}
	}
}

// BenchmarkConcurrentMappings benchmarks concurrent port mapping operations
func BenchmarkConcurrentMappings(b *testing.B) {
	mock := NewMockPortMapper()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			port, err := mock.MapPort("TCP", 8080+i%1000, 5*time.Minute)
			if err != nil {
				b.Fatalf("Mapping failed: %v", err)
			}

			err = mock.UnmapPort("TCP", port)
			if err != nil {
				b.Fatalf("Unmapping failed: %v", err)
			}
			i++
		}
	})
}

// TestRenewalManagerPortChangeCallback tests that port change callbacks are invoked correctly
func TestRenewalManagerPortChangeCallback(t *testing.T) {
	t.Run("Callback invoked when port changes", func(t *testing.T) {
		// Create a mock mapper that returns a different port on second call
		callCount := 0
		mock := NewMockPortMapper()
		mock.SetExternalIP("203.0.113.100")

		// Create a custom mock that changes port on subsequent calls
		portChangingMapper := &portChangingMockMapper{
			MockPortMapper: mock,
			ports:          []int{8080, 9090}, // First call returns 8080, second returns 9090
			callCount:      &callCount,
		}

		renewal := NewRenewalManager(portChangingMapper, "TCP", 8080, 8080)

		var mu sync.Mutex
		var newPortReceived int
		callbackInvoked := false

		renewal.SetPortChangeCallback(func(newPort int) {
			mu.Lock()
			defer mu.Unlock()
			callbackInvoked = true
			newPortReceived = newPort
		})

		// Manually trigger a renewal that will return a different port
		callCount = 1 // Skip to second call which returns 9090
		renewal.renew()

		mu.Lock()
		invoked := callbackInvoked
		port := newPortReceived
		mu.Unlock()

		if !invoked {
			t.Error("Expected callback to be invoked when port changed")
		}

		if port != 9090 {
			t.Errorf("Expected new port to be 9090, got %d", port)
		}

		// Verify the renewal manager's port was updated
		if renewal.ExternalPort() != 9090 {
			t.Errorf("Expected RenewalManager external port to be 9090, got %d", renewal.ExternalPort())
		}
	})

	t.Run("Callback not invoked when port stays the same", func(t *testing.T) {
		mock := NewMockPortMapper()
		mock.SetExternalIP("203.0.113.100")

		renewal := NewRenewalManager(mock, "TCP", 8080, 8080)

		callbackInvoked := false
		renewal.SetPortChangeCallback(func(newPort int) {
			callbackInvoked = true
		})

		// Initial mapping
		mock.MapPort("TCP", 8080, 5*time.Minute)

		// Trigger renewal - should return same port
		renewal.renew()

		if callbackInvoked {
			t.Error("Callback should not be invoked when port stays the same")
		}
	})
}

// portChangingMockMapper is a mock that returns different ports on subsequent calls
type portChangingMockMapper struct {
	*MockPortMapper
	ports     []int
	callCount *int
}

func (m *portChangingMockMapper) MapPort(protocol string, internalPort int, duration time.Duration) (int, error) {
	idx := *m.callCount
	if idx >= len(m.ports) {
		idx = len(m.ports) - 1
	}
	*m.callCount++
	return m.ports[idx], nil
}

// TestNATListenerExternalPortUpdate tests that NATListener updates correctly when port changes
func TestNATListenerExternalPortUpdate(t *testing.T) {
	t.Run("NATListener updates external port and address", func(t *testing.T) {
		// Create a listener directly for testing (bypassing actual network)
		mock := NewMockPortMapper()
		mock.SetExternalIP("203.0.113.100")

		addr := NewNATAddr("tcp", "0.0.0.0:8080", "203.0.113.100:8080")
		renewal := NewRenewalManager(mock, "TCP", 8080, 8080)

		listener := &NATListener{
			renewal:      renewal,
			externalPort: 8080,
			externalIP:   "203.0.113.100",
			addr:         addr,
		}

		// Simulate port change callback
		listener.updateExternalPort(9090)

		if listener.ExternalPort() != 9090 {
			t.Errorf("Expected external port to be 9090, got %d", listener.ExternalPort())
		}

		expectedAddr := "203.0.113.100:9090"
		if listener.Addr().String() != expectedAddr {
			t.Errorf("Expected address to be %s, got %s", expectedAddr, listener.Addr().String())
		}
	})
}

// TestNATPacketListenerExternalPortUpdate tests that NATPacketListener updates correctly when port changes
func TestNATPacketListenerExternalPortUpdate(t *testing.T) {
	t.Run("NATPacketListener updates external port and address", func(t *testing.T) {
		mock := NewMockPortMapper()
		mock.SetExternalIP("203.0.113.100")

		addr := NewNATAddr("udp", "0.0.0.0:8080", "203.0.113.100:8080")
		renewal := NewRenewalManager(mock, "UDP", 8080, 8080)

		// Create a mock packet conn
		localAddr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:8080")
		remoteAddr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:9999")
		mockConn := NewMockUDPConn(localAddr, remoteAddr)

		listener := &NATPacketListener{
			conn:         mockConn,
			renewal:      renewal,
			externalPort: 8080,
			externalIP:   "203.0.113.100",
			addr:         addr,
		}

		// Simulate port change callback
		listener.updateExternalPort(9090)

		if listener.ExternalPort() != 9090 {
			t.Errorf("Expected external port to be 9090, got %d", listener.ExternalPort())
		}

		expectedAddr := "203.0.113.100:9090"
		if listener.Addr().String() != expectedAddr {
			t.Errorf("Expected address to be %s, got %s", expectedAddr, listener.Addr().String())
		}
	})

	t.Run("Cached packet conn address is also updated", func(t *testing.T) {
		mock := NewMockPortMapper()
		mock.SetExternalIP("203.0.113.100")

		addr := NewNATAddr("udp", "0.0.0.0:8080", "203.0.113.100:8080")
		renewal := NewRenewalManager(mock, "UDP", 8080, 8080)

		localAddr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:8080")
		remoteAddr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:9999")
		mockConn := NewMockUDPConn(localAddr, remoteAddr)

		listener := &NATPacketListener{
			conn:         mockConn,
			renewal:      renewal,
			externalPort: 8080,
			externalIP:   "203.0.113.100",
			addr:         addr,
		}

		// Create cached packet conn by calling PacketConn()
		_ = listener.PacketConn()

		// Simulate port change
		listener.updateExternalPort(9090)

		// Verify cached packet conn also has updated address
		pconn := listener.PacketConn().(*NATPacketConn)
		expectedAddr := "203.0.113.100:9090"
		if pconn.LocalAddr().String() != expectedAddr {
			t.Errorf("Expected cached conn address to be %s, got %s", expectedAddr, pconn.LocalAddr().String())
		}
	})
}
