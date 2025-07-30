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
func (r *RenewalManager) Start() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.started {
		return
	}

	r.started = true
	r.done = make(chan struct{}) // Create new channel each time
	r.ticker = time.NewTicker(renewalInterval)
	go r.renewLoop()
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
func (r *RenewalManager) renewLoop() {
	for {
		select {
		case <-r.ticker.C:
			r.renew()
		case <-r.done:
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
