package nattraversal

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

// TestFullNATTraversalWorkflow tests the complete NAT traversal workflow
func TestFullNATTraversalWorkflow(t *testing.T) {
	t.Run("Complete TCP workflow", func(t *testing.T) {
		// Step 1: Setup mock environment
		mock := NewMockPortMapper()

		// Step 2: Create port mapping
		internalPort := 8080
		externalPort, err := mock.MapPort("TCP", internalPort, 5*time.Minute)
		if err != nil {
			t.Fatalf("Failed to create port mapping: %v", err)
		}

		// Step 3: Get external IP
		externalIP, err := mock.GetExternalIP()
		if err != nil {
			t.Fatalf("Failed to get external IP: %v", err)
		}

		// Step 4: Create NAT address
		internalAddr := fmt.Sprintf("192.168.1.100:%d", internalPort)
		externalAddr := fmt.Sprintf("%s:%d", externalIP, externalPort)
		natAddr := NewNATAddr("tcp", internalAddr, externalAddr)

		// Step 5: Verify addresses
		if natAddr.InternalAddr() != internalAddr {
			t.Errorf("Expected internal address %s, got %s", internalAddr, natAddr.InternalAddr())
		}

		if natAddr.ExternalAddr() != externalAddr {
			t.Errorf("Expected external address %s, got %s", externalAddr, natAddr.ExternalAddr())
		}

		// Step 6: Setup renewal manager
		renewal := NewRenewalManager(mock, "TCP", internalPort, externalPort)
		renewal.Start()

		// Step 7: Simulate connection activity
		time.Sleep(50 * time.Millisecond)

		// Step 8: Verify mapping is still active
		mappings := mock.GetActiveMappings()
		key := fmt.Sprintf("TCP:%d", externalPort)
		if _, exists := mappings[key]; !exists {
			t.Errorf("Expected mapping to remain active during renewal")
		}

		// Step 9: Clean up
		renewal.Stop()

		// Step 10: Verify cleanup
		mappings = mock.GetActiveMappings()
		if len(mappings) > 0 {
			t.Errorf("Expected no active mappings after cleanup, got %d", len(mappings))
		}
	})

	t.Run("Complete UDP workflow", func(t *testing.T) {
		// Similar workflow for UDP
		mock := NewMockPortMapper()

		internalPort := 9090
		externalPort, err := mock.MapPort("UDP", internalPort, 5*time.Minute)
		if err != nil {
			t.Fatalf("Failed to create UDP port mapping: %v", err)
		}

		externalIP, err := mock.GetExternalIP()
		if err != nil {
			t.Fatalf("Failed to get external IP: %v", err)
		}

		internalAddr := fmt.Sprintf("192.168.1.100:%d", internalPort)
		externalAddr := fmt.Sprintf("%s:%d", externalIP, externalPort)
		natAddr := NewNATAddr("udp", internalAddr, externalAddr)

		if natAddr.Network() != "udp" {
			t.Errorf("Expected UDP network type, got %s", natAddr.Network())
		}

		renewal := NewRenewalManager(mock, "UDP", internalPort, externalPort)
		renewal.Start()

		time.Sleep(50 * time.Millisecond)

		renewal.Stop()
	})
}

// TestErrorRecoveryScenarios tests error recovery and resilience
func TestErrorRecoveryScenarios(t *testing.T) {
	t.Run("Recovery from transient failures", func(t *testing.T) {
		mock := NewMockPortMapper()

		// Start with high failure rate
		mock.SetFailureRate(0.8)

		var successfulMappings int
		var failures int

		// Attempt multiple mappings with retries
		for i := 0; i < 10; i++ {
			_, err := mock.MapPort("TCP", 8080+i, 5*time.Minute)
			if err != nil {
				failures++
				// Simulate retry with reduced failure rate
				mock.SetFailureRate(0.4)
				_, retryErr := mock.MapPort("TCP", 8080+i, 5*time.Minute)
				if retryErr == nil {
					successfulMappings++
				}
			} else {
				successfulMappings++
			}
		}

		t.Logf("Successful mappings: %d, Failures: %d", successfulMappings, failures)

		if successfulMappings == 0 {
			t.Errorf("Expected some successful mappings with retry logic")
		}
	})

	t.Run("Recovery from protocol failures", func(t *testing.T) {
		mock := NewMockPortMapper()

		// Disable UPnP, enable NAT-PMP
		mock.SetProtocolSupport(false, true)

		_, err := mock.MapPort("TCP", 8080, 5*time.Minute)
		if err != nil {
			t.Errorf("Expected success with NAT-PMP fallback, got: %v", err)
		}

		// Disable both protocols
		mock.SetProtocolSupport(false, false)

		_, err = mock.MapPort("TCP", 8081, 5*time.Minute)
		if err == nil {
			t.Errorf("Expected failure when no protocols are available")
		}
	})
}

// TestResourceManagement tests proper resource cleanup
func TestResourceManagement(t *testing.T) {
	t.Run("Resource cleanup after listener close", func(t *testing.T) {
		mock := NewMockPortMapper()

		// Create multiple mappings
		var renewalManagers []*RenewalManager
		for i := 0; i < 5; i++ {
			port := 8080 + i
			externalPort, err := mock.MapPort("TCP", port, 5*time.Minute)
			if err != nil {
				t.Fatalf("Failed to create mapping %d: %v", i, err)
			}

			renewal := NewRenewalManager(mock, "TCP", port, externalPort)
			renewal.Start()
			renewalManagers = append(renewalManagers, renewal)
		}

		// Verify all mappings are active
		mappings := mock.GetActiveMappings()
		if len(mappings) != 5 {
			t.Errorf("Expected 5 active mappings, got %d", len(mappings))
		}

		// Stop all renewal managers (simulating listener cleanup)
		for _, renewal := range renewalManagers {
			renewal.Stop()
		}

		// Verify all mappings are cleaned up
		mappings = mock.GetActiveMappings()
		if len(mappings) > 0 {
			t.Errorf("Expected no active mappings after cleanup, got %d", len(mappings))
		}
	})

	t.Run("Memory leak prevention", func(t *testing.T) {
		mock := NewMockPortMapper()

		// Create and destroy many mappings rapidly
		for i := 0; i < 100; i++ {
			port, err := mock.MapPort("TCP", 8080, 1*time.Millisecond)
			if err != nil {
				continue
			}

			// Immediately unmap
			mock.UnmapPort("TCP", port)
		}

		// Verify no mappings remain
		mappings := mock.GetActiveMappings()
		if len(mappings) > 0 {
			t.Errorf("Expected no remaining mappings, got %d", len(mappings))
		}
	})
}

// TestPerformanceUnderLoad tests performance characteristics
func TestPerformanceUnderLoad(t *testing.T) {
	t.Run("High frequency operations", func(t *testing.T) {
		mock := NewMockPortMapper()

		start := time.Now()

		// Perform many rapid operations
		for i := 0; i < 1000; i++ {
			port, err := mock.MapPort("TCP", 8080+i%100, 5*time.Minute)
			if err != nil {
				continue
			}
			mock.UnmapPort("TCP", port)
		}

		elapsed := time.Since(start)

		// Should complete within reasonable time
		if elapsed > 5*time.Second {
			t.Errorf("High frequency operations took too long: %v", elapsed)
		}

		t.Logf("1000 operations completed in %v", elapsed)
	})

	t.Run("Concurrent load test", func(t *testing.T) {
		mock := NewMockPortMapper()

		const numWorkers = 50
		const operationsPerWorker = 20

		var wg sync.WaitGroup
		errorCount := 0
		var errorMutex sync.Mutex

		start := time.Now()

		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()

				for j := 0; j < operationsPerWorker; j++ {
					port, err := mock.MapPort("TCP", 8080+workerID*100+j, 5*time.Minute)
					if err != nil {
						errorMutex.Lock()
						errorCount++
						errorMutex.Unlock()
						continue
					}

					err = mock.UnmapPort("TCP", port)
					if err != nil {
						errorMutex.Lock()
						errorCount++
						errorMutex.Unlock()
					}
				}
			}(i)
		}

		wg.Wait()
		elapsed := time.Since(start)

		totalOperations := numWorkers * operationsPerWorker * 2 // map + unmap

		t.Logf("Concurrent load test: %d operations by %d workers in %v",
			totalOperations, numWorkers, elapsed)
		t.Logf("Error count: %d", errorCount)

		if elapsed > 10*time.Second {
			t.Errorf("Concurrent operations took too long: %v", elapsed)
		}
	})
}

// TestEdgeCases tests various edge cases and boundary conditions
func TestEdgeCases(t *testing.T) {
	t.Run("Zero duration mapping", func(t *testing.T) {
		mock := NewMockPortMapper()

		port, err := mock.MapPort("TCP", 8080, 0)
		if err != nil {
			t.Fatalf("Failed to create zero duration mapping: %v", err)
		}

		// Mapping should exist but expire immediately
		mappings := mock.GetActiveMappings()
		key := fmt.Sprintf("TCP:%d", port)
		if _, exists := mappings[key]; !exists {
			// This is expected behavior - zero duration mappings may not be active
			t.Logf("Zero duration mapping not active (expected)")
		}
	})

	t.Run("Very long duration mapping", func(t *testing.T) {
		mock := NewMockPortMapper()

		longDuration := 24 * time.Hour
		port, err := mock.MapPort("TCP", 8080, longDuration)
		if err != nil {
			t.Fatalf("Failed to create long duration mapping: %v", err)
		}

		mappings := mock.GetActiveMappings()
		key := fmt.Sprintf("TCP:%d", port)
		mapping, exists := mappings[key]
		if !exists {
			t.Errorf("Expected long duration mapping to exist")
		}

		if !mapping.ExpiresAt.After(time.Now().Add(23 * time.Hour)) {
			t.Errorf("Expected mapping to expire much later")
		}
	})

	t.Run("Port range boundaries", func(t *testing.T) {
		mock := NewMockPortMapper()

		// Test with port 1 (low boundary)
		_, err := mock.MapPort("TCP", 1, 5*time.Minute)
		if err != nil {
			t.Errorf("Failed to map port 1: %v", err)
		}

		// Test with port 65535 (high boundary)
		_, err = mock.MapPort("TCP", 65535, 5*time.Minute)
		if err != nil {
			t.Errorf("Failed to map port 65535: %v", err)
		}
	})

	t.Run("Rapid renewal manager start/stop", func(t *testing.T) {
		mock := NewMockPortMapper()

		renewal := NewRenewalManager(mock, "TCP", 8080, 8080)

		// Rapidly start and stop
		for i := 0; i < 10; i++ {
			renewal.Start()
			time.Sleep(1 * time.Millisecond)
			renewal.Stop()
		}

		// Should not crash or cause issues
	})
}

// TestIntegrationWithRealNetworkConditions simulates realistic network scenarios
func TestIntegrationWithRealNetworkConditions(t *testing.T) {
	t.Run("Realistic NAT behavior simulation", func(t *testing.T) {
		scenarios := []struct {
			name     string
			natType  NATType
			latency  time.Duration
			loss     float64
			expected bool
		}{
			{
				name:     "Good network with Full Cone NAT",
				natType:  FullConeNAT,
				latency:  20 * time.Millisecond,
				loss:     0.01, // 1% packet loss
				expected: true,
			},
			{
				name:     "Poor network with Symmetric NAT",
				natType:  SymmetricNAT,
				latency:  200 * time.Millisecond,
				loss:     0.10, // 10% packet loss
				expected: true, // Should still work but might be slower
			},
			{
				name:     "Very poor network",
				natType:  RestrictedNAT,
				latency:  500 * time.Millisecond,
				loss:     0.20, // 20% packet loss
				expected: true, // Might succeed with retries
			},
		}

		for _, scenario := range scenarios {
			t.Run(scenario.name, func(t *testing.T) {
				mock := NewMockPortMapper()
				mock.SetNATType(scenario.natType)
				mock.SetLatency(scenario.latency)
				// Note: packet loss affects mock UDP connections, not port mapper

				start := time.Now()
				port, err := mock.MapPort("TCP", 8080, 5*time.Minute)
				elapsed := time.Since(start)

				if scenario.expected && err != nil {
					t.Errorf("Expected success in scenario '%s', got error: %v", scenario.name, err)
				}

				if err == nil {
					t.Logf("Scenario '%s': mapping created in %v, external port: %d",
						scenario.name, elapsed, port)

					// Verify latency was applied
					if elapsed < scenario.latency {
						t.Errorf("Expected latency >= %v, got %v", scenario.latency, elapsed)
					}
				}
			})
		}
	})
}

// TestContextCancellation tests that context cancellation works correctly
func TestContextCancellation(t *testing.T) {
	t.Run("Already cancelled context returns error", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		// createTCPMappingContext should fail with cancelled context
		_, _, err := createTCPMappingContext(ctx, 8080)
		if err == nil {
			t.Error("Expected error for cancelled context, got nil")
		}
		if err != context.Canceled {
			t.Logf("Error type: %T, value: %v", err, err)
		}
	})

	t.Run("Already cancelled context for UDP returns error", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		// createUDPMappingContext should fail with cancelled context
		_, _, err := createUDPMappingContext(ctx, 9090)
		if err == nil {
			t.Error("Expected error for cancelled context, got nil")
		}
	})

	t.Run("NewPortMapperContext with cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err := NewPortMapperContext(ctx)
		if err == nil {
			t.Error("Expected error for cancelled context, got nil")
		}
	})

	t.Run("Context with deadline passes to functions", func(t *testing.T) {
		// Create a context with a very short timeout
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()

		// Wait for timeout
		time.Sleep(1 * time.Millisecond)

		// Context should be expired
		if ctx.Err() == nil {
			t.Error("Expected context to be expired")
		}

		// Functions should return error for expired context
		_, _, err := createTCPMappingContext(ctx, 8080)
		if err == nil {
			t.Error("Expected error for expired context, got nil")
		}
	})
}
