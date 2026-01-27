package nattraversal

import (
	"log/slog"
	"sync"
	"time"
)

// PortChangeCallback is called when the external port changes during renewal.
// The callback receives the new external port number.
type PortChangeCallback func(newExternalPort int)

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
	onPortChange PortChangeCallback
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

// SetPortChangeCallback sets a callback function that will be invoked when
// the external port changes during renewal. This can happen if the NAT device
// assigns a different port during renewal (rare but possible).
func (r *RenewalManager) SetPortChangeCallback(callback PortChangeCallback) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.onPortChange = callback
}

// ExternalPort returns the current external port number.
// This may change if the NAT device assigns a different port during renewal.
func (r *RenewalManager) ExternalPort() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.externalPort
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
// If the NAT device assigns a different external port during renewal,
// the callback (if set) will be invoked with the new port number.
func (r *RenewalManager) renew() {
	newPort, err := r.mapper.MapPort(r.protocol, r.internalPort, mappingDuration)
	if err != nil {
		slog.Warn("port mapping renewal failed",
			"protocol", r.protocol,
			"port", r.externalPort,
			"error", err)
		return
	}

	r.mu.Lock()
	oldPort := r.externalPort
	callback := r.onPortChange
	if newPort != oldPort {
		r.externalPort = newPort
		slog.Info("external port changed during renewal",
			"protocol", r.protocol,
			"oldPort", oldPort,
			"newPort", newPort)
	}
	r.mu.Unlock()

	// Invoke callback outside the lock to prevent deadlocks
	if newPort != oldPort && callback != nil {
		callback(newPort)
	}

	slog.Debug("port mapping renewed",
		"protocol", r.protocol,
		"port", newPort)
}
