package services

import (
	"sort"
	"sync"
)

// Stats holds the last N synth durations in ms.
type Stats struct {
	mu   sync.Mutex
	vals []float64
}

// NewStats builds a tracker.
func NewStats() *Stats { return &Stats{} }

// Observe records one synth call in ms.
func (s *Stats) Observe(ms float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.vals = append(s.vals, ms)
	if len(s.vals) > 1024 {
		s.vals = append([]float64(nil), s.vals[len(s.vals)-1024:]...)
	}
}

// Snapshot returns p50/p95 over the ring (0,0 when empty).
func (s *Stats) Snapshot() (p50, p95 float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.vals) == 0 {
		return 0, 0
	}
	cp := append([]float64(nil), s.vals...)
	sort.Float64s(cp)
	p50 = cp[int(float64(len(cp)-1)*0.50)]
	p95 = cp[int(float64(len(cp)-1)*0.95)]
	return p50, p95
}
