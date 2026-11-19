package observability

import (
	"strings"
	"time"
)

// ConcurrencySampleQuality turns heartbeat state into a read-side status.
// A stopped sampler therefore becomes stale/unavailable even when its last
// persisted status still says healthy.
func ConcurrencySampleQuality(status string, lastSampleAt *time.Time, now time.Time, staleAfter time.Duration) string {
	status = strings.TrimSpace(status)
	if status == "" || lastSampleAt == nil {
		return "unavailable"
	}
	if now.Sub(*lastSampleAt) > staleAfter {
		return "stale"
	}
	return status
}
