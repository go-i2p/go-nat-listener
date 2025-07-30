package nattraversal

import "time"

// Constants for port mapping renewal management
const (
	renewalInterval = 45 * time.Minute
	mappingDuration = 90 * time.Minute // double the interval for safety
)
