//go:build !linux && !darwin && !freebsd && !openbsd && !netbsd && !dragonfly && !windows

package nattraversal

import "net"

// readDefaultGateway is a stub for platforms without specific gateway detection.
// Returns nil, nil to trigger the fallback heuristic.
//
// This includes platforms like:
// - Android (uses Linux kernel but /proc may not be accessible)
// - iOS (no shell access)
// - Plan 9
// - js/wasm
// - Other less common platforms
func readDefaultGateway() (net.IP, error) {
	// No platform-specific implementation available
	// The discoverGateway function will use discoverGatewayFallback()
	return nil, nil
}
