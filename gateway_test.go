package nattraversal

import (
	"net"
	"testing"
)

// TestDiscoverGateway tests the cross-platform gateway discovery function.
// This test runs on all platforms.
func TestDiscoverGateway(t *testing.T) {
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
		ipv4 := gateway.To4()
		if ipv4[3] != 1 {
			t.Errorf("Expected fallback gateway to end in .1, got last octet: %d", ipv4[3])
		}

		t.Logf("Fallback gateway: %v", gateway)
	})
}

// TestReadDefaultGateway tests that the platform-specific implementation exists
// and doesn't panic. The actual behavior is tested in platform-specific test files.
func TestReadDefaultGateway(t *testing.T) {
	t.Run("readDefaultGateway does not panic", func(t *testing.T) {
		// This should not panic on any platform
		gateway, err := readDefaultGateway()

		// Log the result for debugging
		if err != nil {
			t.Logf("readDefaultGateway returned error: %v", err)
		} else if gateway != nil {
			t.Logf("readDefaultGateway returned gateway: %v", gateway)
		} else {
			t.Log("readDefaultGateway returned nil (fallback will be used)")
		}

		// If we got a gateway, validate it
		if gateway != nil {
			if gateway.To4() == nil {
				t.Errorf("Expected IPv4 gateway, got: %v", gateway)
			}
			if gateway.Equal(net.IPv4zero) {
				t.Error("Gateway should not be 0.0.0.0")
			}
		}
	})
}
