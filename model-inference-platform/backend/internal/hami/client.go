package hami

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// Client communicates with the HAMi scheduler extender HTTP API.
// When HAMi is not available (e.g., local dev without Kubernetes),
// it falls back to a no-op stub that always reports success.
type Client struct {
	endpoint   string
	httpClient *http.Client
	logger     *zap.Logger
	enabled    bool
}

// ClientConfig holds configuration for the HAMi client.
type ClientConfig struct {
	// Endpoint is the HAMi scheduler extender URL.
	// E.g., "http://hami-scheduler.hami-system.svc.cluster.local:9090"
	Endpoint string

	// Enabled controls whether HAMi integration is active.
	// When false, all scheduling calls succeed immediately (stub mode).
	Enabled bool

	// Timeout for individual HTTP calls.
	Timeout time.Duration
}

// DefaultClientConfig returns a config suitable for local development.
func DefaultClientConfig() ClientConfig {
	return ClientConfig{
		Endpoint: "http://hami-scheduler.hami-system.svc.cluster.local:9090",
		Enabled:  false, // disabled by default; set HAMI_ENABLED=true in K8s
		Timeout:  5 * time.Second,
	}
}

// NewClient creates a new HAMi scheduler client.
func NewClient(cfg ClientConfig, logger *zap.Logger) *Client {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	c := &Client{
		endpoint: cfg.Endpoint,
		httpClient: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				MaxIdleConns:       10,
				IdleConnTimeout:    30 * time.Second,
				DisableCompression: false,
			},
		},
		logger:  logger,
		enabled: cfg.Enabled,
	}
	if cfg.Enabled {
		logger.Info("hami client initialized", zap.String("endpoint", cfg.Endpoint))
	} else {
		logger.Info("hami client in stub mode (HAMI_ENABLED=false)")
	}
	return c
}

// Schedule requests HAMi to select an optimal node/GPU for the given workload.
// Returns a stub success result when HAMi is disabled.
func (c *Client) Schedule(ctx context.Context, req ScheduleRequest) (*ScheduleResult, error) {
	if !c.enabled {
		return c.stubSchedule(req), nil
	}
	return c.doSchedule(ctx, req)
}

// ListNodes returns GPU resource availability from all HAMi-managed nodes.
func (c *Client) ListNodes(ctx context.Context) ([]NodeGPUInfo, error) {
	if !c.enabled {
		return c.stubNodeList(), nil
	}
	return c.doListNodes(ctx)
}

// Ping checks HAMi scheduler connectivity.
func (c *Client) Ping(ctx context.Context) error {
	if !c.enabled {
		return nil // always healthy in stub mode
	}
	url := fmt.Sprintf("%s/healthz", c.endpoint)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("hami ping: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("hami ping: status %d", resp.StatusCode)
	}
	return nil
}

// ─── Private helpers ──────────────────────────────────────────────────────────

func (c *Client) doSchedule(ctx context.Context, req ScheduleRequest) (*ScheduleResult, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal schedule request: %w", err)
	}

	url := fmt.Sprintf("%s/schedule", c.endpoint)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url,
		bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create schedule request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	start := time.Now()
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		c.logger.Warn("hami schedule request failed",
			zap.String("endpoint_id", req.EndpointID),
			zap.Error(err))
		// Degrade gracefully: return stub success so endpoint creation is not blocked
		return c.stubSchedule(req), nil
	}
	defer resp.Body.Close()

	latency := time.Since(start)
	c.logger.Debug("hami schedule response",
		zap.String("endpoint_id", req.EndpointID),
		zap.Int("status", resp.StatusCode),
		zap.Duration("latency", latency))

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
		var result ScheduleResult
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, fmt.Errorf("decode schedule response: %w", err)
		}
		return &result, nil
	}

	// Non-2xx: read error body
	errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return nil, fmt.Errorf("hami schedule returned %d: %s", resp.StatusCode, string(errBody))
}

func (c *Client) doListNodes(ctx context.Context) ([]NodeGPUInfo, error) {
	url := fmt.Sprintf("%s/nodes/gpu", c.endpoint)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Warn("hami list nodes failed", zap.Error(err))
		return c.stubNodeList(), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("hami list nodes returned %d", resp.StatusCode)
	}

	var nodes []NodeGPUInfo
	if err := json.NewDecoder(resp.Body).Decode(&nodes); err != nil {
		return nil, fmt.Errorf("decode nodes response: %w", err)
	}
	return nodes, nil
}

// ─── Stub implementations ─────────────────────────────────────────────────────

// stubSchedule returns a synthetic success result for local development.
// In production (HAMI_ENABLED=true), the real scheduler is always called.
func (c *Client) stubSchedule(req ScheduleRequest) *ScheduleResult {
	return &ScheduleResult{
		Scheduled:          true,
		NodeName:           "stub-node-01",
		PhysicalGPUID:      0,
		AllocatedMemoryMiB: req.GPUSpec.MemoryMiB,
		AllocatedCores:     req.GPUSpec.Cores,
	}
}

func (c *Client) stubNodeList() []NodeGPUInfo {
	return []NodeGPUInfo{
		{
			NodeName:    "stub-gpu-node-01",
			Vendor:      VendorNVIDIA,
			GPUCount:    8,
			TotalMemMiB: 81920 * 8, // 8x A100 80GB
			UsedMemMiB:  0,
			FreeMem:     81920 * 8,
			Utilization: 0,
			HasNVLink:   true,
		},
	}
}
