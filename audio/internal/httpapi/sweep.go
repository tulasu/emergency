package httpapi

import (
	"strconv"
	"time"
)

// parseLimit parses ?limit= for the sweep endpoint.
func parseLimit(q string) (int, error) {
	return strconv.Atoi(q)
}

// sweepTTL is the orphan grace: 7 days survive the PUT→synth race
// (a reused hash clears unreferenced_since via the trigger).
func sweepTTL() time.Duration { return 7 * 24 * time.Hour }
