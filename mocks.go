package nattraversal

import (
	"fmt"
	"math/rand"
	"net"
	"sync"
	"time"
)

// MockPortMapper implements PortMapper interface for testing
type MockPortMapper struct {
	mu             sync.RWMutex
	mappings       map[string]*PortMapping
	externalIP     string
	supportsUPnP   bool
	supportsNATPMP bool
	latency        time.Duration
	failureRate    float64
	portExhaustion bool
	natType        NATType
	rng            *rand.Rand // Seeded RNG for reproducible tests
}

// PortMapping represents a mock port mapping
type PortMapping struct {
	Protocol     string
	InternalPort int
	ExternalPort int
	ExpiresAt    time.Time
	Active       bool
}

// NATType represents different NAT behaviors
type NATType int

const (
	FullConeNAT NATType = iota
	RestrictedNAT
	PortRestrictedNAT
	SymmetricNAT
)

// NewMockPortMapper creates a new mock port mapper
func NewMockPortMapper() *MockPortMapper {
	return &MockPortMapper{
		mappings:       make(map[string]*PortMapping),
		externalIP:     "203.0.113.100", // RFC5737 test IP
		supportsUPnP:   true,
		supportsNATPMP: true,
		natType:        FullConeNAT,
		rng:            rand.New(rand.NewSource(42)), // Fixed seed for reproducibility
	}
}

// SetRandomSeed sets a custom random seed for reproducible tests.
// Use different seeds to test different random scenarios deterministically.
func (m *MockPortMapper) SetRandomSeed(seed int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rng = rand.New(rand.NewSource(seed))
}

// SetExternalIP sets the mock external IP
func (m *MockPortMapper) SetExternalIP(ip string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.externalIP = ip
}

// SetLatency simulates network latency
func (m *MockPortMapper) SetLatency(d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.latency = d
}

// SetFailureRate sets the probability of operations failing (0.0 to 1.0)
func (m *MockPortMapper) SetFailureRate(rate float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failureRate = rate
}

// SetPortExhaustion simulates port exhaustion scenarios
func (m *MockPortMapper) SetPortExhaustion(exhausted bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.portExhaustion = exhausted
}

// SetNATType sets the NAT behavior type
func (m *MockPortMapper) SetNATType(natType NATType) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.natType = natType
}

// SetProtocolSupport configures which protocols are supported
func (m *MockPortMapper) SetProtocolSupport(upnp, natpmp bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.supportsUPnP = upnp
	m.supportsNATPMP = natpmp
}

// MapPort implements the PortMapper interface
func (m *MockPortMapper) MapPort(protocol string, internalPort int, duration time.Duration) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Simulate latency
	if m.latency > 0 {
		time.Sleep(m.latency)
	}

	// Simulate failure rate
	if m.failureRate > 0 && m.shouldFail() {
		return 0, fmt.Errorf("mock: random failure occurred")
	}

	// Validate protocol
	if protocol != "TCP" && protocol != "UDP" {
		return 0, fmt.Errorf("mock: unsupported protocol: %s", protocol)
	}

	// Check protocol support
	if !m.supportsUPnP && !m.supportsNATPMP {
		return 0, fmt.Errorf("mock: no protocols supported")
	}

	// Simulate port exhaustion
	if m.portExhaustion {
		return 0, fmt.Errorf("mock: no available ports")
	}

	// Generate external port based on NAT type
	externalPort := m.generateExternalPort(internalPort)

	key := fmt.Sprintf("%s:%d", protocol, externalPort)
	m.mappings[key] = &PortMapping{
		Protocol:     protocol,
		InternalPort: internalPort,
		ExternalPort: externalPort,
		ExpiresAt:    time.Now().Add(duration),
		Active:       true,
	}

	return externalPort, nil
}

// UnmapPort implements the PortMapper interface
func (m *MockPortMapper) UnmapPort(protocol string, externalPort int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Simulate latency
	if m.latency > 0 {
		time.Sleep(m.latency)
	}

	// Simulate failure rate
	if m.failureRate > 0 && m.shouldFail() {
		return fmt.Errorf("mock: random failure occurred")
	}

	key := fmt.Sprintf("%s:%d", protocol, externalPort)
	if mapping, exists := m.mappings[key]; exists {
		mapping.Active = false
		delete(m.mappings, key)
	}

	return nil
}

// GetExternalIP implements the PortMapper interface
func (m *MockPortMapper) GetExternalIP() (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Simulate latency
	if m.latency > 0 {
		time.Sleep(m.latency)
	}

	// Simulate failure rate
	if m.failureRate > 0 && m.shouldFail() {
		return "", fmt.Errorf("mock: random failure occurred")
	}

	return m.externalIP, nil
}

// GetActiveMappings returns all active port mappings
func (m *MockPortMapper) GetActiveMappings() map[string]*PortMapping {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]*PortMapping)
	for k, v := range m.mappings {
		if v.Active && time.Now().Before(v.ExpiresAt) {
			result[k] = v
		}
	}
	return result
}

// ExpireMapping simulates a mapping expiring
func (m *MockPortMapper) ExpireMapping(protocol string, externalPort int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s:%d", protocol, externalPort)
	if mapping, exists := m.mappings[key]; exists {
		mapping.ExpiresAt = time.Now().Add(-time.Second)
	}
}

// SimulateMappingChange simulates NAT mapping changes mid-connection
func (m *MockPortMapper) SimulateMappingChange(protocol string, oldPort, newPort int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	oldKey := fmt.Sprintf("%s:%d", protocol, oldPort)
	newKey := fmt.Sprintf("%s:%d", protocol, newPort)

	if mapping, exists := m.mappings[oldKey]; exists {
		delete(m.mappings, oldKey)
		mapping.ExternalPort = newPort
		m.mappings[newKey] = mapping
	}
}

// generateExternalPort generates external port based on NAT type
func (m *MockPortMapper) generateExternalPort(internalPort int) int {
	switch m.natType {
	case FullConeNAT:
		// Full cone: same external port for all internal endpoints
		return internalPort
	case RestrictedNAT, PortRestrictedNAT:
		// Restricted: predictable mapping
		return internalPort + 1000
	case SymmetricNAT:
		// Symmetric: different port for each destination (use seeded RNG)
		return internalPort + m.rng.Intn(10000)
	default:
		return internalPort
	}
}

// shouldFail determines if operation should fail based on failure rate.
// Uses seeded RNG for reproducible test results.
func (m *MockPortMapper) shouldFail() bool {
	if m.failureRate <= 0 {
		return false
	}
	return m.rng.Float64() < m.failureRate
}

// MockRenewalManager provides a testable renewal manager
type MockRenewalManager struct {
	mu              sync.RWMutex
	renewalCount    int
	failureCount    int
	renewalInterval time.Duration
	shouldFail      bool
	stopped         bool
}

// NewMockRenewalManager creates a new mock renewal manager
func NewMockRenewalManager() *MockRenewalManager {
	return &MockRenewalManager{
		renewalInterval: 100 * time.Millisecond, // Fast for testing
	}
}

// SetShouldFail configures renewal failures
func (m *MockRenewalManager) SetShouldFail(fail bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldFail = fail
}

// GetRenewalCount returns the number of successful renewals
func (m *MockRenewalManager) GetRenewalCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.renewalCount
}

// GetFailureCount returns the number of failed renewals
func (m *MockRenewalManager) GetFailureCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.failureCount
}

// IsStopped returns whether the renewal manager has been stopped
func (m *MockRenewalManager) IsStopped() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.stopped
}

// MockNetworkConditions simulates various network conditions
type MockNetworkConditions struct {
	PacketLoss  float64 // 0.0 to 1.0
	Latency     time.Duration
	Jitter      time.Duration
	Bandwidth   int64 // bytes per second
	Blocked     bool
	Unreachable bool
	rng         *rand.Rand // Seeded RNG for reproducible tests
}

// NewMockNetworkConditions creates default network conditions
func NewMockNetworkConditions() *MockNetworkConditions {
	return &MockNetworkConditions{
		PacketLoss: 0.0,
		Latency:    10 * time.Millisecond,
		Jitter:     2 * time.Millisecond,
		Bandwidth:  1024 * 1024,                  // 1MB/s
		rng:        rand.New(rand.NewSource(42)), // Fixed seed for reproducibility
	}
}

// SetRandomSeed sets a custom random seed for reproducible tests.
func (m *MockNetworkConditions) SetRandomSeed(seed int64) {
	m.rng = rand.New(rand.NewSource(seed))
}

// SimulatePacketLoss determines if a packet should be dropped.
// Uses seeded RNG for reproducible test results.
func (m *MockNetworkConditions) SimulatePacketLoss() bool {
	if m.PacketLoss <= 0 {
		return false
	}
	return m.rng.Float64() < m.PacketLoss
}

// SimulateLatency adds simulated network latency.
// Uses seeded RNG for jitter calculation.
func (m *MockNetworkConditions) SimulateLatency() {
	if m.Latency > 0 {
		jitter := time.Duration(0)
		if m.Jitter > 0 {
			jitter = time.Duration(m.rng.Int63n(int64(m.Jitter)))
		}
		time.Sleep(m.Latency + jitter)
	}
}

// MockFirewall simulates firewall behaviors
type MockFirewall struct {
	mu            sync.RWMutex
	blockedPorts  map[int]bool
	blockedIPs    map[string]bool
	allowedPairs  map[string]bool // "ip:port" pairs
	defaultPolicy bool            // true = allow, false = block
}

// NewMockFirewall creates a new mock firewall
func NewMockFirewall() *MockFirewall {
	return &MockFirewall{
		blockedPorts:  make(map[int]bool),
		blockedIPs:    make(map[string]bool),
		allowedPairs:  make(map[string]bool),
		defaultPolicy: true,
	}
}

// BlockPort adds a port to the blocked list
func (f *MockFirewall) BlockPort(port int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.blockedPorts[port] = true
}

// BlockIP adds an IP to the blocked list
func (f *MockFirewall) BlockIP(ip string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.blockedIPs[ip] = true
}

// AllowConnection explicitly allows a specific IP:port combination
func (f *MockFirewall) AllowConnection(ip string, port int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.allowedPairs[fmt.Sprintf("%s:%d", ip, port)] = true
}

// SetDefaultPolicy sets the default firewall policy
func (f *MockFirewall) SetDefaultPolicy(allow bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.defaultPolicy = allow
}

// IsBlocked checks if a connection should be blocked
func (f *MockFirewall) IsBlocked(ip string, port int) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()

	// Check explicit allow list first
	if f.allowedPairs[fmt.Sprintf("%s:%d", ip, port)] {
		return false
	}

	// Check blocked IP
	if f.blockedIPs[ip] {
		return true
	}

	// Check blocked port
	if f.blockedPorts[port] {
		return true
	}

	// Return opposite of default policy
	return !f.defaultPolicy
}

// Reset clears all firewall rules and resets to default allow policy.
func (f *MockFirewall) Reset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.blockedPorts = make(map[int]bool)
	f.blockedIPs = make(map[string]bool)
	f.allowedPairs = make(map[string]bool)
	f.defaultPolicy = true
}

// MockUDPConn provides a mock UDP connection for testing
type MockUDPConn struct {
	localAddr   *net.UDPAddr
	remoteAddr  *net.UDPAddr
	readBuffer  [][]byte
	writeBuffer [][]byte
	mu          sync.RWMutex
	closed      bool
	conditions  *MockNetworkConditions
	firewall    *MockFirewall
}

// NewMockUDPConn creates a new mock UDP connection
func NewMockUDPConn(localAddr, remoteAddr *net.UDPAddr) *MockUDPConn {
	return &MockUDPConn{
		localAddr:   localAddr,
		remoteAddr:  remoteAddr,
		readBuffer:  make([][]byte, 0),
		writeBuffer: make([][]byte, 0),
		conditions:  NewMockNetworkConditions(),
		firewall:    NewMockFirewall(),
	}
}

// Read implements net.Conn interface
func (c *MockUDPConn) Read(b []byte) (n int, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return 0, fmt.Errorf("connection closed")
	}

	if len(c.readBuffer) == 0 {
		return 0, fmt.Errorf("no data available")
	}

	// Simulate network conditions (only if configured)
	if c.conditions != nil {
		c.conditions.SimulateLatency()
		if c.conditions.SimulatePacketLoss() {
			return 0, fmt.Errorf("packet lost")
		}
	}

	data := c.readBuffer[0]
	c.readBuffer = c.readBuffer[1:]

	copy(b, data)
	return len(data), nil
}

// Write implements net.Conn interface
func (c *MockUDPConn) Write(b []byte) (n int, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return 0, fmt.Errorf("connection closed")
	}

	// Check firewall (only if configured)
	if c.firewall != nil && c.remoteAddr != nil {
		if c.firewall.IsBlocked(c.remoteAddr.IP.String(), c.remoteAddr.Port) {
			return 0, fmt.Errorf("connection blocked by firewall")
		}
	}

	// Simulate network conditions (only if configured)
	if c.conditions != nil {
		c.conditions.SimulateLatency()
		if c.conditions.SimulatePacketLoss() {
			return 0, fmt.Errorf("packet lost")
		}
	}

	// Store written data
	data := make([]byte, len(b))
	copy(data, b)
	c.writeBuffer = append(c.writeBuffer, data)

	return len(b), nil
}

// Close implements net.Conn interface
func (c *MockUDPConn) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
	return nil
}

// LocalAddr implements net.Conn interface
func (c *MockUDPConn) LocalAddr() net.Addr {
	return c.localAddr
}

// RemoteAddr implements net.Conn interface
func (c *MockUDPConn) RemoteAddr() net.Addr {
	return c.remoteAddr
}

// SetDeadline implements net.Conn interface
func (c *MockUDPConn) SetDeadline(t time.Time) error {
	return nil
}

// SetReadDeadline implements net.Conn interface
func (c *MockUDPConn) SetReadDeadline(t time.Time) error {
	return nil
}

// SetWriteDeadline implements net.Conn interface
func (c *MockUDPConn) SetWriteDeadline(t time.Time) error {
	return nil
}

// AddReadData adds data to the read buffer
func (c *MockUDPConn) AddReadData(data []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.readBuffer = append(c.readBuffer, data)
}

// GetWrittenData returns all written data
func (c *MockUDPConn) GetWrittenData() [][]byte {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result := make([][]byte, len(c.writeBuffer))
	copy(result, c.writeBuffer)
	return result
}

// SetNetworkConditions sets the network conditions for this connection
func (c *MockUDPConn) SetNetworkConditions(conditions *MockNetworkConditions) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.conditions = conditions
}

// SetFirewall sets the firewall for this connection
func (c *MockUDPConn) SetFirewall(firewall *MockFirewall) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.firewall = firewall
}
