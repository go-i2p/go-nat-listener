package nattraversal

import (
	"testing"
	"time"
)

// TestComprehensiveNATScenarios tests various NAT scenarios using test helpers
func TestComprehensiveNATScenarios(t *testing.T) {
	scenarios := []TestScenario{
		{
			Name:        "Ideal conditions with Full Cone NAT",
			NATType:     FullConeNAT,
			HasFirewall: false,
			PoorNetwork: false,
			FailureRate: 0.0,
			Expected:    true,
		},
		{
			Name:        "Full Cone NAT with firewall",
			NATType:     FullConeNAT,
			HasFirewall: true,
			PoorNetwork: false,
			FailureRate: 0.0,
			Expected:    true,
		},
		{
			Name:        "Restricted NAT with poor network",
			NATType:     RestrictedNAT,
			HasFirewall: false,
			PoorNetwork: true,
			FailureRate: 0.0, // No artificial failure rate for deterministic tests
			Expected:    true,
		},
		{
			Name:        "Symmetric NAT with challenges",
			NATType:     SymmetricNAT,
			HasFirewall: true,
			PoorNetwork: true,
			FailureRate: 0.0,  // No artificial failure rate for deterministic tests
			Expected:    true, // Should succeed with proper error handling
		},
		{
			Name:        "Port Restricted NAT normal conditions",
			NATType:     PortRestrictedNAT,
			HasFirewall: false,
			PoorNetwork: false,
			FailureRate: 0.0, // No artificial failure rate for deterministic tests
			Expected:    true,
		},
	}

	helper := NewTestHelper(t)
	helper.RunScenarios(scenarios, func(h *TestHelper, scenario TestScenario) {
		// Test port mapping creation
		externalPort, err := h.CreatePortMapping("TCP", 8080, 5*time.Minute)

		if scenario.Expected {
			h.AssertNoError(err, "Port mapping should succeed in scenario: "+scenario.Name)
			h.AssertPortMappingExists("TCP", externalPort, "Mapping should exist")

			// Test external IP discovery
			externalIP, err := h.GetPortMapper().GetExternalIP()
			h.AssertNoError(err, "External IP discovery should succeed")
			h.AssertNotEqual("", externalIP, "External IP should not be empty")

			// Test renewal manager
			renewal := h.CreateRenewalManager("TCP", 8080, externalPort)
			renewal.Start()

			// Wait for renewal cycle
			h.WaitForRenewal()

			// Mapping should still exist after renewal
			h.AssertPortMappingExists("TCP", externalPort, "Mapping should persist after renewal")

		} else {
			// In challenging scenarios, we might expect failures
			if err != nil {
				t.Logf("Expected failure in challenging scenario: %s, error: %v", scenario.Name, err)
			}
		}
	})
}

// TestConnectionEstablishmentScenarios tests connection establishment under various conditions
func TestConnectionEstablishmentScenarios(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	t.Run("Simple connection establishment", func(t *testing.T) {
		helper.Reset() // Reset state for isolation
		helper.SetupFullConeNAT()

		// Create port mapping
		externalPort, err := helper.CreatePortMapping("TCP", 8080, 5*time.Minute)
		helper.AssertNoError(err, "Port mapping creation")

		// Get external IP
		externalIP, err := helper.GetPortMapper().GetExternalIP()
		helper.AssertNoError(err, "External IP discovery")

		// Create NAT address
		internalAddr := "192.168.1.100:8080"
		externalAddr := externalIP + ":" + string(rune(externalPort))
		natAddr := NewNATAddr("tcp", internalAddr, externalAddr)

		// Verify address properties
		helper.AssertEqual("tcp", natAddr.Network(), "Network type")
		helper.AssertEqual(internalAddr, natAddr.InternalAddr(), "Internal address")
		helper.AssertEqual(externalAddr, natAddr.ExternalAddr(), "External address")
	})

	t.Run("Connection with network challenges", func(t *testing.T) {
		helper.Reset() // Reset state for isolation
		helper.SetupPoorNetwork()
		helper.SetupSymmetricNAT()

		// Should still be able to create mapping, just slower
		start := time.Now()
		_, err := helper.CreatePortMapping("UDP", 9090, 5*time.Minute)
		elapsed := time.Since(start)

		helper.AssertNoError(err, "Port mapping with poor network")

		// Should take longer due to latency
		if elapsed < 100*time.Millisecond {
			t.Errorf("Expected operation to take longer due to network latency")
		}

		// Test packet transmission with loss
		conn := helper.CreateMockConnection(9090, 9090)

		// Try multiple sends to account for packet loss
		successCount := 0
		for i := 0; i < 10; i++ {
			_, err := conn.Write([]byte("test"))
			if err == nil {
				successCount++
			}
		}

		// Should have some successes despite packet loss
		if successCount == 0 {
			t.Errorf("Expected some successful transmissions despite packet loss")
		}

		t.Logf("Successful transmissions: %d/10", successCount)
	})

	t.Run("Firewall traversal", func(t *testing.T) {
		helper.Reset() // Reset state for isolation - clears poor network conditions
		helper.SetupRestrictiveFirewall()

		// Create connection to allowed port
		conn := helper.CreateMockConnection(8080, 8080)
		_, err := conn.Write([]byte("allowed"))
		helper.AssertNoError(err, "Connection to allowed port should succeed")

		// Create connection to blocked port
		conn2 := helper.CreateMockConnection(8080, 9999)
		_, err = conn2.Write([]byte("blocked"))
		helper.AssertError(err, "Connection to blocked port should fail")
	})
}

// TestKeepAliveScenarios tests keep-alive functionality under various conditions
func TestKeepAliveScenarios(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	t.Run("Keep-alive with stable connection", func(t *testing.T) {
		helper.Reset() // Ensure clean state
		helper.SetupFullConeNAT()

		conn := helper.CreateMockConnection(8080, 8080)

		// Send multiple keep-alive packets
		for i := 0; i < 5; i++ {
			data := []byte("KEEPALIVE" + string(rune(i)))
			n, err := conn.Write(data)
			helper.AssertNoError(err, "Keep-alive transmission")
			helper.AssertEqual(len(data), n, "Bytes written")

			time.Sleep(10 * time.Millisecond)
		}

		// Verify all packets were sent
		written := conn.GetWrittenData()
		helper.AssertEqual(5, len(written), "Number of keep-alive packets")
	})

	t.Run("Keep-alive with packet loss", func(t *testing.T) {
		helper.Reset() // Ensure clean state
		helper.SetupPoorNetwork()

		conn := helper.CreateMockConnection(8080, 8080)

		// Send keep-alive packets with retries
		successCount := 0
		for i := 0; i < 20; i++ {
			data := []byte("KEEPALIVE")
			_, err := conn.Write(data)
			if err == nil {
				successCount++
			}
			time.Sleep(5 * time.Millisecond)
		}

		// Should have some successes despite packet loss
		if successCount == 0 {
			t.Errorf("Expected some successful keep-alive transmissions")
		}

		t.Logf("Keep-alive success rate: %d/20", successCount)
	})
}

// TestPortMappingRenewalScenarios tests renewal under various conditions
func TestPortMappingRenewalScenarios(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	t.Run("Renewal with stable conditions", func(t *testing.T) {
		helper.Reset() // Ensure clean state
		helper.SetupFullConeNAT()

		externalPort, err := helper.CreatePortMapping("TCP", 8080, 5*time.Minute)
		helper.AssertNoError(err, "Initial port mapping")

		renewal := helper.CreateRenewalManager("TCP", 8080, externalPort)
		renewal.Start()

		// Wait for multiple renewal cycles
		time.Sleep(250 * time.Millisecond)

		// Mapping should still exist
		helper.AssertPortMappingExists("TCP", externalPort, "Mapping after renewal")
	})

	t.Run("Renewal with intermittent failures", func(t *testing.T) {
		helper.Reset() // Ensure clean state
		helper.SetupFullConeNAT()

		externalPort, err := helper.CreatePortMapping("TCP", 8080, 5*time.Minute)
		helper.AssertNoError(err, "Initial port mapping")

		renewal := helper.CreateRenewalManager("TCP", 8080, externalPort)
		renewal.Start()

		// Simulate network issues after renewal starts
		time.Sleep(50 * time.Millisecond)
		cleanupFailure := helper.SimulateNetworkFailure()

		// Wait during failure period
		time.Sleep(100 * time.Millisecond)

		// Restore network
		cleanupFailure()

		// Wait for recovery
		time.Sleep(100 * time.Millisecond)

		// System should recover and mapping should still exist
		helper.AssertPortMappingExists("TCP", externalPort, "Mapping after failure recovery")
	})
}

// TestResourceCleanupScenarios tests resource cleanup under various conditions
func TestResourceCleanupScenarios(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	t.Run("Clean shutdown", func(t *testing.T) {
		helper.Reset() // Ensure clean state
		helper.RunWithCleanup(func() {
			// Create multiple mappings
			port1, err := helper.CreatePortMapping("TCP", 8080, 5*time.Minute)
			helper.AssertNoError(err, "First mapping")

			port2, err := helper.CreatePortMapping("UDP", 9090, 5*time.Minute)
			helper.AssertNoError(err, "Second mapping")

			// Start renewal managers
			renewal1 := helper.CreateRenewalManager("TCP", 8080, port1)
			renewal2 := helper.CreateRenewalManager("UDP", 9090, port2)
			renewal1.Start()
			renewal2.Start()

			// Verify mappings exist
			helper.AssertPortMappingExists("TCP", port1, "TCP mapping before cleanup")
			helper.AssertPortMappingExists("UDP", port2, "UDP mapping before cleanup")

			// Cleanup will happen automatically due to RunWithCleanup
		})

		// After cleanup, no mappings should remain
		mappings := helper.GetPortMapper().GetActiveMappings()
		helper.AssertEqual(0, len(mappings), "No mappings after cleanup")
	})

	t.Run("Cleanup under stress", func(t *testing.T) {
		helper.Reset() // Ensure clean state
		helper.RunWithCleanup(func() {
			// Create many mappings rapidly
			for i := 0; i < 10; i++ {
				_, err := helper.CreatePortMapping("TCP", 8080+i, 5*time.Minute)
				helper.AssertNoError(err, "Mass mapping creation")
			}

			// Verify all mappings exist
			mappings := helper.GetPortMapper().GetActiveMappings()
			helper.AssertEqual(10, len(mappings), "All mappings created")

			// Cleanup will handle all mappings
		})

		// Verify cleanup was successful
		mappings := helper.GetPortMapper().GetActiveMappings()
		helper.AssertEqual(0, len(mappings), "Cleanup removed all mappings")
	})
}

// TestErrorHandlingScenarios tests various error conditions and recovery
func TestErrorHandlingScenarios(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	t.Run("Protocol not supported", func(t *testing.T) {
		helper.Reset() // Ensure clean state
		helper.GetPortMapper().SetProtocolSupport(false, false)

		_, err := helper.GetPortMapper().MapPort("TCP", 8080, 5*time.Minute)
		helper.AssertError(err, "Should fail when no protocols supported")
	})

	t.Run("Port exhaustion recovery", func(t *testing.T) {
		helper.Reset() // Ensure clean state
		// Ensure protocols are supported first
		helper.GetPortMapper().SetProtocolSupport(true, true)

		// Enable port exhaustion
		cleanupExhaustion := helper.SimulatePortExhaustion()

		_, err := helper.GetPortMapper().MapPort("TCP", 8080, 5*time.Minute)
		helper.AssertError(err, "Should fail during port exhaustion")

		// Restore port availability
		cleanupExhaustion()

		// Should succeed after recovery
		_, err = helper.CreatePortMapping("TCP", 8080, 5*time.Minute)
		helper.AssertNoError(err, "Should succeed after port exhaustion recovery")
	})

	t.Run("Mapping expiration handling", func(t *testing.T) {
		helper.Reset() // Ensure clean state
		// Create mapping with very short duration
		externalPort, err := helper.CreatePortMapping("TCP", 8080, 1*time.Millisecond)
		helper.AssertNoError(err, "Short duration mapping")

		// Wait for expiration
		time.Sleep(5 * time.Millisecond)

		// Force expiration check
		helper.GetPortMapper().ExpireMapping("TCP", externalPort)

		// Mapping should no longer be active
		helper.AssertPortMappingNotExists("TCP", externalPort, "Expired mapping should not exist")
	})
}
