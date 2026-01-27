package nattraversal

import (
	"log/slog"
	"sync"
	"time"
)

// RenewalManager handles automatic port mapping renewal.
// Moved from: renew.go
type RenewalManager struct {
	mapper       PortMapper
	protocol     string
	internalPort int
	externalPort int
	ticker       *time.Ticker
	done         chan struct{}
	mu           sync.Mutex
	started      bool
}

// NewRenewalManager creates a renewal manager for a port mapping.
func NewRenewalManager(mapper PortMapper, protocol string, internalPort, externalPort int) *RenewalManager {
	return &RenewalManager{
		mapper:       mapper,
		protocol:     protocol,
		internalPort: internalPort,
		externalPort: externalPort,
		// done channel will be created when Start() is called
	}
}

// Start begins the renewal process in a background goroutine.
// Multiple Start/Stop cycles are safe - each cycle creates fresh channels
// and the goroutine captures local references to avoid data races.
func (r *RenewalManager) Start() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.started {
		return
	}

	r.started = true
	r.done = make(chan struct{})
	r.ticker = time.NewTicker(renewalInterval)

	// Capture local references to avoid data race between goroutine reads
	// and subsequent Start() writes after Stop() is called.
	done := r.done
	ticker := r.ticker
	go r.renewLoop(ticker.C, done)
}

// Stop terminates the renewal process and unmaps the port.
func (r *RenewalManager) Stop() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.started {
		return
	}

	r.started = false
	close(r.done)
	r.ticker.Stop()

	// Unmap the port
	err := r.mapper.UnmapPort(r.protocol, r.externalPort)
	if err != nil {
		slog.Warn("failed to unmap port during shutdown",
			"protocol", r.protocol,
			"port", r.externalPort,
			"error", err)
	}
}

// renewLoop runs the renewal ticker in a goroutine.
// It receives the ticker channel and done channel as parameters to avoid
// data races when Start() is called after Stop() on the same instance.
func (r *RenewalManager) renewLoop(tickerC <-chan time.Time, done <-chan struct{}) {
	for {
		select {
		case <-tickerC:
			r.renew()
		case <-done:
			return
		}
	}
}

// renew attempts to refresh the port mapping.
func (r *RenewalManager) renew() {
	_, err := r.mapper.MapPort(r.protocol, r.internalPort, mappingDuration)
	if err != nil {
		slog.Warn("port mapping renewal failed",
			"protocol", r.protocol,
			"port", r.externalPort,
			"error", err)
	} else {
		slog.Debug("port mapping renewed",
			"protocol", r.protocol,
			"port", r.externalPort)
	}
}
