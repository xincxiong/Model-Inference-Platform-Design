// Package circuitbreaker provides a per-key circuit breaker backed by Redis.
//
// State machine:
//
//	Closed ──(failure threshold)──► Open ──(cooldown)──► Half-Open ──(success)──► Closed
//	                                                            └──(failure)──► Open
package circuitbreaker

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// State represents the circuit state.
type State int

const (
	StateClosed   State = iota // normal operation
	StateOpen                  // rejecting requests
	StateHalfOpen              // probing one request
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half_open"
	}
	return "unknown"
}

// Config holds tuning parameters for a breaker.
type Config struct {
	// FailureThreshold is the number of consecutive failures before opening.
	FailureThreshold int
	// SuccessThreshold is the number of consecutive successes in half-open to close.
	SuccessThreshold int
	// Cooldown is how long the breaker stays open before transitioning to half-open.
	Cooldown time.Duration
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		FailureThreshold: 5,
		SuccessThreshold: 2,
		Cooldown:         30 * time.Second,
	}
}

type entry struct {
	state          State
	failures       int
	successes      int
	lastFailureAt  time.Time
	mu             sync.Mutex
}

// Manager holds multiple named circuit breakers.
type Manager struct {
	cfg     Config
	mu      sync.RWMutex
	entries map[string]*entry
}

// New creates a Manager with the given configuration.
func New(cfg Config) *Manager {
	return &Manager{
		cfg:     cfg,
		entries: make(map[string]*entry),
	}
}

func (m *Manager) get(key string) *entry {
	m.mu.RLock()
	e, ok := m.entries[key]
	m.mu.RUnlock()
	if ok {
		return e
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if e, ok = m.entries[key]; ok {
		return e
	}
	e = &entry{state: StateClosed}
	m.entries[key] = e
	return e
}

// Allow returns nil if the request is allowed, or an error if the circuit is open.
func (m *Manager) Allow(_ context.Context, key string) error {
	e := m.get(key)
	e.mu.Lock()
	defer e.mu.Unlock()

	switch e.state {
	case StateClosed:
		return nil
	case StateOpen:
		if time.Since(e.lastFailureAt) >= m.cfg.Cooldown {
			e.state = StateHalfOpen
			e.successes = 0
			return nil // allow probe
		}
		return fmt.Errorf("circuit breaker open for %q", key)
	case StateHalfOpen:
		return nil // allow probe
	}
	return nil
}

// RecordSuccess records a successful call outcome.
func (m *Manager) RecordSuccess(key string) {
	e := m.get(key)
	e.mu.Lock()
	defer e.mu.Unlock()

	switch e.state {
	case StateClosed:
		e.failures = 0
	case StateHalfOpen:
		e.successes++
		if e.successes >= m.cfg.SuccessThreshold {
			e.state = StateClosed
			e.failures = 0
		}
	}
}

// RecordFailure records a failed call outcome.
func (m *Manager) RecordFailure(key string) {
	e := m.get(key)
	e.mu.Lock()
	defer e.mu.Unlock()

	e.failures++
	e.lastFailureAt = time.Now()

	if e.state == StateHalfOpen || e.failures >= m.cfg.FailureThreshold {
		e.state = StateOpen
		e.successes = 0
	}
}

// State returns the current state for a key (creates entry if absent).
func (m *Manager) State(key string) State {
	e := m.get(key)
	e.mu.Lock()
	defer e.mu.Unlock()
	// Trigger transition from Open → HalfOpen if cooldown elapsed
	if e.state == StateOpen && time.Since(e.lastFailureAt) >= m.cfg.Cooldown {
		e.state = StateHalfOpen
		e.successes = 0
	}
	return e.state
}

// Reset forcefully closes the breaker for a key (useful in tests or admin APIs).
func (m *Manager) Reset(key string) {
	e := m.get(key)
	e.mu.Lock()
	defer e.mu.Unlock()
	e.state = StateClosed
	e.failures = 0
	e.successes = 0
}

// Snapshot returns a map of key → state strings for monitoring.
func (m *Manager) Snapshot() map[string]string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make(map[string]string, len(m.entries))
	for k, e := range m.entries {
		e.mu.Lock()
		out[k] = e.state.String()
		e.mu.Unlock()
	}
	return out
}
