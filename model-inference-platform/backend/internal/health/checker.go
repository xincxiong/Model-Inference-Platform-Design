// Package health provides infrastructure health checks for DB, Redis, and worker pool.
package health

import (
	"context"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// Status represents the health status of a component.
type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusDegraded  Status = "degraded"
	StatusUnhealthy Status = "unhealthy"
)

// ComponentHealth holds the health state of a single component.
type ComponentHealth struct {
	Status    Status `json:"status"`
	Message   string `json:"message,omitempty"`
	Latency   string `json:"latency_ms,omitempty"`
	CheckedAt string `json:"checked_at"`
}

// Report is the full system health report.
type Report struct {
	Status     Status                     `json:"status"`
	Timestamp  string                     `json:"timestamp"`
	Components map[string]ComponentHealth `json:"components"`
}

// Checker runs periodic health checks on critical infrastructure components.
type Checker struct {
	db    *pgxpool.Pool
	rdb   *redis.Client
	mu    sync.RWMutex
	cache *Report
}

// New creates a Checker and starts a background goroutine that refreshes
// health status every 30 seconds.
func New(db *pgxpool.Pool, rdb *redis.Client) *Checker {
	c := &Checker{db: db, rdb: rdb}
	c.cache = c.run()
	go c.background()
	return c
}

// Latest returns the most recent cached health report (non-blocking).
func (c *Checker) Latest() *Report {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.cache
}

// Check performs an on-demand health check (may block up to 5 s).
func (c *Checker) Check() *Report {
	r := c.run()
	c.mu.Lock()
	c.cache = r
	c.mu.Unlock()
	return r
}

func (c *Checker) background() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		r := c.run()
		c.mu.Lock()
		c.cache = r
		c.mu.Unlock()
	}
}

func (c *Checker) run() *Report {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	components := make(map[string]ComponentHealth)

	// ── Postgres ──────────────────────────────────────────────────────────
	start := time.Now()
	pgErr := c.db.Ping(ctx)
	pgLatency := time.Since(start)
	if pgErr != nil {
		components["postgres"] = ComponentHealth{
			Status:    StatusUnhealthy,
			Message:   pgErr.Error(),
			Latency:   latencyStr(pgLatency),
			CheckedAt: nowStr(),
		}
	} else {
		stat := c.db.Stat()
		msg := ""
		s := StatusHealthy
		if stat.AcquireCount() > 0 && stat.IdleConns() == 0 {
			s = StatusDegraded
			msg = "connection pool exhausted"
		}
		components["postgres"] = ComponentHealth{
			Status:    s,
			Message:   msg,
			Latency:   latencyStr(pgLatency),
			CheckedAt: nowStr(),
		}
	}

	// ── Redis ─────────────────────────────────────────────────────────────
	start = time.Now()
	rErr := c.rdb.Ping(ctx).Err()
	rLatency := time.Since(start)
	if rErr != nil {
		components["redis"] = ComponentHealth{
			Status:    StatusUnhealthy,
			Message:   rErr.Error(),
			Latency:   latencyStr(rLatency),
			CheckedAt: nowStr(),
		}
	} else {
		components["redis"] = ComponentHealth{
			Status:    StatusHealthy,
			Latency:   latencyStr(rLatency),
			CheckedAt: nowStr(),
		}
	}

	// ── Aggregate ─────────────────────────────────────────────────────────
	overall := StatusHealthy
	for _, ch := range components {
		if ch.Status == StatusUnhealthy {
			overall = StatusUnhealthy
			break
		}
		if ch.Status == StatusDegraded {
			overall = StatusDegraded
		}
	}

	return &Report{
		Status:     overall,
		Timestamp:  nowStr(),
		Components: components,
	}
}

func latencyStr(d time.Duration) string {
	return d.Truncate(time.Millisecond).String()
}

func nowStr() string {
	return time.Now().UTC().Format(time.RFC3339)
}
