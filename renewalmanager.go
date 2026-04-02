package nattraversal

import (
	"sync"
	"time"

	"github.com/go-i2p/logger"
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
	log.WithFields(logger.Fields{
		"protocol":     protocol,
		"internalPort": internalPort,
		"externalPort": externalPort,
	}).Debug("creating renewal manager")
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
		log.WithFields(logger.Fields{
			"protocol": r.protocol,
			"port":     r.externalPort,
		}).Debug("renewal manager already started, ignoring")
		return
	}

	r.started = true
	r.done = make(chan struct{})
	r.ticker = time.NewTicker(renewalInterval)

	log.WithFields(logger.Fields{
		"protocol":        r.protocol,
		"internalPort":    r.internalPort,
		"externalPort":    r.externalPort,
		"renewalInterval": renewalInterval.String(),
	}).Debug("starting port renewal")

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
		log.WithFields(logger.Fields{
			"protocol": r.protocol,
			"port":     r.externalPort,
		}).Debug("renewal manager already stopped, ignoring")
		return
	}

	log.WithFields(logger.Fields{
		"protocol": r.protocol,
		"port":     r.externalPort,
	}).Debug("stopping port renewal manager")

	r.started = false
	close(r.done)
	r.ticker.Stop()

	// Unmap the port
	err := r.mapper.UnmapPort(r.protocol, r.externalPort)
	if err != nil {
		log.WithError(err).WithFields(logger.Fields{
			"protocol": r.protocol,
			"port":     r.externalPort,
		}).Warn("failed to unmap port during shutdown")
	} else {
		log.WithFields(logger.Fields{
			"protocol": r.protocol,
			"port":     r.externalPort,
		}).Debug("port unmapped successfully during shutdown")
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
	log.WithFields(logger.Fields{
		"protocol": r.protocol,
		"port":     r.externalPort,
	}).Debug("attempting port mapping renewal")

	newPort, err := r.mapper.MapPort(r.protocol, r.internalPort, mappingDuration)
	if err != nil {
		log.WithError(err).WithFields(logger.Fields{
			"protocol": r.protocol,
			"port":     r.externalPort,
		}).Warn("port mapping renewal failed")
		return
	}

	r.mu.Lock()
	oldPort := r.externalPort
	callback := r.onPortChange
	if newPort != oldPort {
		r.externalPort = newPort
		log.WithFields(logger.Fields{
			"protocol": r.protocol,
			"oldPort":  oldPort,
			"newPort":  newPort,
		}).Info("external port changed during renewal")
	}
	r.mu.Unlock()

	// Invoke callback outside the lock to prevent deadlocks
	if newPort != oldPort && callback != nil {
		callback(newPort)
	}

	log.WithFields(logger.Fields{
		"protocol": r.protocol,
		"port":     newPort,
	}).Debug("port mapping renewed successfully")
}
