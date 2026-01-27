package nattraversal

import (
	"fmt"
	"net"
	"testing"
	"time"
)

// TestHelper provides utilities for NAT traversal testing
type TestHelper struct {
	t           *testing.T
	portMapper  *MockPortMapper
	firewall    *MockFirewall
	conditions  *MockNetworkConditions
	activePorts []int
	renewalMgrs []*RenewalManager
}

// NewTestHelper creates a new test helper instance
func NewTestHelper(t *testing.T) *TestHelper {
	return &TestHelper{
		t:           t,
		portMapper:  NewMockPortMapper(),
		firewall:    NewMockFirewall(),
		conditions:  NewMockNetworkConditions(),
		activePorts: make([]int, 0),
		renewalMgrs: make([]*RenewalManager, 0),
	}
}

// SetupFullConeNAT configures the environment for Full Cone NAT testing
func (h *TestHelper) SetupFullConeNAT() {
	h.portMapper.SetNATType(FullConeNAT)
	h.portMapper.SetExternalIP("203.0.113.100")
	h.conditions.PacketLoss = 0.0
	h.conditions.Latency = 10 * time.Millisecond
}

// SetupRestrictedNAT configures the environment for Restricted NAT testing
func (h *TestHelper) SetupRestrictedNAT() {
	h.portMapper.SetNATType(RestrictedNAT)
	h.portMapper.SetExternalIP("203.0.113.101")
	h.conditions.PacketLoss = 0.02 // 2% packet loss
	h.conditions.Latency = 25 * time.Millisecond
}

// SetupSymmetricNAT configures the environment for Symmetric NAT testing
func (h *TestHelper) SetupSymmetricNAT() {
	h.portMapper.SetNATType(SymmetricNAT)
	h.portMapper.SetExternalIP("203.0.113.102")
	h.conditions.PacketLoss = 0.05 // 5% packet loss
	h.conditions.Latency = 50 * time.Millisecond
}

// SetupPoorNetwork simulates poor network conditions
func (h *TestHelper) SetupPoorNetwork() {
	h.conditions.PacketLoss = 0.15 // 15% packet loss
	h.conditions.Latency = 200 * time.Millisecond
	h.conditions.Jitter = 50 * time.Millisecond
	h.portMapper.SetLatency(100 * time.Millisecond)
}

// SetupRestrictiveFirewall configures a restrictive firewall
func (h *TestHelper) SetupRestrictiveFirewall() {
	h.firewall.SetDefaultPolicy(false) // Default deny
	// Allow only specific ports
	h.firewall.AllowConnection("203.0.113.100", 8080)
	h.firewall.AllowConnection("203.0.113.100", 9090)
}

// CreatePortMapping creates a port mapping and tracks it for cleanup
func (h *TestHelper) CreatePortMapping(protocol string, internalPort int, duration time.Duration) (int, error) {
	externalPort, err := h.portMapper.MapPort(protocol, internalPort, duration)
	if err != nil {
		return 0, err
	}

	h.activePorts = append(h.activePorts, externalPort)
	return externalPort, nil
}

// CreateRenewalManager creates a renewal manager and tracks it for cleanup
func (h *TestHelper) CreateRenewalManager(protocol string, internalPort, externalPort int) *RenewalManager {
	renewal := NewRenewalManager(h.portMapper, protocol, internalPort, externalPort)
	h.renewalMgrs = append(h.renewalMgrs, renewal)
	return renewal
}

// CreateMockConnection creates a mock UDP connection with configured conditions
func (h *TestHelper) CreateMockConnection(localPort, remotePort int) *MockUDPConn {
	localAddr, _ := net.ResolveUDPAddr("udp", fmt.Sprintf("192.168.1.100:%d", localPort))
	remoteAddr, _ := net.ResolveUDPAddr("udp", fmt.Sprintf("203.0.113.100:%d", remotePort))

	conn := NewMockUDPConn(localAddr, remoteAddr)
	conn.SetNetworkConditions(h.conditions)
	conn.SetFirewall(h.firewall)

	return conn
}

// VerifyMapping checks if a port mapping exists and is active
func (h *TestHelper) VerifyMapping(protocol string, externalPort int) bool {
	mappings := h.portMapper.GetActiveMappings()
	key := fmt.Sprintf("%s:%d", protocol, externalPort)
	_, exists := mappings[key]
	return exists
}

// SimulateNetworkFailure temporarily increases failure rates
func (h *TestHelper) SimulateNetworkFailure() func() {
	originalFailureRate := h.portMapper.failureRate
	originalPacketLoss := h.conditions.PacketLoss

	h.portMapper.SetFailureRate(0.8) // 80% failure rate
	h.conditions.PacketLoss = 0.5    // 50% packet loss

	// Return cleanup function
	return func() {
		h.portMapper.SetFailureRate(originalFailureRate)
		h.conditions.PacketLoss = originalPacketLoss
	}
}

// SimulatePortExhaustion enables port exhaustion simulation
func (h *TestHelper) SimulatePortExhaustion() func() {
	h.portMapper.SetPortExhaustion(true)

	// Return cleanup function
	return func() {
		h.portMapper.SetPortExhaustion(false)
	}
}

// WaitForRenewal waits for at least one renewal cycle to complete
func (h *TestHelper) WaitForRenewal() {
	time.Sleep(150 * time.Millisecond) // Longer than mock renewal interval
}

// AssertNoError fails the test if error is not nil
func (h *TestHelper) AssertNoError(err error, message string) {
	if err != nil {
		h.t.Fatalf("%s: %v", message, err)
	}
}

// AssertError fails the test if error is nil
func (h *TestHelper) AssertError(err error, message string) {
	if err == nil {
		h.t.Fatalf("%s: expected error but got none", message)
	}
}

// AssertEqual fails the test if values are not equal
func (h *TestHelper) AssertEqual(expected, actual interface{}, message string) {
	if expected != actual {
		h.t.Fatalf("%s: expected %v, got %v", message, expected, actual)
	}
}

// AssertNotEqual fails the test if values are equal
func (h *TestHelper) AssertNotEqual(expected, actual interface{}, message string) {
	if expected == actual {
		h.t.Fatalf("%s: expected %v to not equal %v", message, expected, actual)
	}
}

// AssertPortMappingExists verifies a port mapping exists
func (h *TestHelper) AssertPortMappingExists(protocol string, externalPort int, message string) {
	if !h.VerifyMapping(protocol, externalPort) {
		h.t.Fatalf("%s: port mapping %s:%d does not exist", message, protocol, externalPort)
	}
}

// AssertPortMappingNotExists verifies a port mapping does not exist
func (h *TestHelper) AssertPortMappingNotExists(protocol string, externalPort int, message string) {
	if h.VerifyMapping(protocol, externalPort) {
		h.t.Fatalf("%s: port mapping %s:%d should not exist", message, protocol, externalPort)
	}
}

// GetPortMapper returns the mock port mapper for advanced operations
func (h *TestHelper) GetPortMapper() *MockPortMapper {
	return h.portMapper
}

// GetFirewall returns the mock firewall for configuration
func (h *TestHelper) GetFirewall() *MockFirewall {
	return h.firewall
}

// GetNetworkConditions returns the mock network conditions for configuration
func (h *TestHelper) GetNetworkConditions() *MockNetworkConditions {
	return h.conditions
}

// Cleanup cleans up all resources created during testing
func (h *TestHelper) Cleanup() {
	// Stop all renewal managers
	for _, renewal := range h.renewalMgrs {
		renewal.Stop()
	}

	// Clean up any remaining mappings
	for _, port := range h.activePorts {
		h.portMapper.UnmapPort("TCP", port)
		h.portMapper.UnmapPort("UDP", port)
	}

	// Reset state
	h.activePorts = h.activePorts[:0]
	h.renewalMgrs = h.renewalMgrs[:0]
}

// Reset resets the TestHelper state between subtests to prevent state leakage.
// This should be called at the start of each subtest when using a shared TestHelper.
func (h *TestHelper) Reset() {
	// Reset network conditions to defaults
	h.conditions.PacketLoss = 0.0
	h.conditions.Latency = 10 * time.Millisecond
	h.conditions.Jitter = 2 * time.Millisecond
	h.conditions.Bandwidth = 1024 * 1024
	h.conditions.Blocked = false
	h.conditions.Unreachable = false
	h.conditions.SetRandomSeed(42) // Reset RNG for reproducibility

	// Reset port mapper to defaults
	h.portMapper.SetNATType(FullConeNAT)
	h.portMapper.SetExternalIP("203.0.113.100")
	h.portMapper.SetLatency(0)
	h.portMapper.SetFailureRate(0)
	h.portMapper.SetPortExhaustion(false)
	h.portMapper.SetRandomSeed(42) // Reset RNG for reproducibility

	// Reset firewall to defaults
	h.firewall.Reset()
}

// RunWithCleanup runs a test function and ensures cleanup happens
func (h *TestHelper) RunWithCleanup(testFunc func()) {
	defer h.Cleanup()
	testFunc()
}

// TestScenario represents a test scenario configuration
type TestScenario struct {
	Name        string
	NATType     NATType
	HasFirewall bool
	PoorNetwork bool
	FailureRate float64
	Expected    bool
}

// RunScenarios runs multiple test scenarios
func (h *TestHelper) RunScenarios(scenarios []TestScenario, testFunc func(*TestHelper, TestScenario)) {
	for _, scenario := range scenarios {
		h.t.Run(scenario.Name, func(t *testing.T) {
			// Create new helper for this scenario
			scenarioHelper := NewTestHelper(t)
			defer scenarioHelper.Cleanup()

			// Configure scenario
			scenarioHelper.portMapper.SetNATType(scenario.NATType)
			scenarioHelper.portMapper.SetFailureRate(scenario.FailureRate)

			if scenario.HasFirewall {
				scenarioHelper.SetupRestrictiveFirewall()
			}

			if scenario.PoorNetwork {
				scenarioHelper.SetupPoorNetwork()
			}

			// Run test
			testFunc(scenarioHelper, scenario)
		})
	}
}

// BenchmarkHelper provides utilities for benchmarking
type BenchmarkHelper struct {
	portMapper *MockPortMapper
}

// NewBenchmarkHelper creates a new benchmark helper
func NewBenchmarkHelper() *BenchmarkHelper {
	return &BenchmarkHelper{
		portMapper: NewMockPortMapper(),
	}
}

// SetupForBenchmark configures optimal settings for benchmarking
func (b *BenchmarkHelper) SetupForBenchmark() {
	b.portMapper.SetLatency(0)            // No artificial latency
	b.portMapper.SetFailureRate(0)        // No failures
	b.portMapper.SetPortExhaustion(false) // No port exhaustion
}

// GetPortMapper returns the port mapper for benchmarking
func (b *BenchmarkHelper) GetPortMapper() *MockPortMapper {
	return b.portMapper
}
