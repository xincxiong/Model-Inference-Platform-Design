package hami

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"go.uber.org/zap"
)

// ClientConfig holds configuration for the HAMi HTTP client.
type ClientConfig struct {
	// Endpoint is the HAMi scheduler extender URL.
	Endpoint string
	// Enabled indicates whether real HAMi scheduling is active.
	Enabled bool
	// Timeout is the HTTP request timeout.
	Timeout time.Duration
}

// Client provides HTTP communication with the HAMi scheduler extender.
// In stub mode (Enabled=false), all scheduling calls succeed immediately
// without connecting to a real HAMi instance.
type Client struct {
	config  ClientConfig
	http    *http.Client
	logger  *zap.Logger
	enabled bool
}

// NewClient creates a new HAMi client from configuration.
func NewClient(cfg ClientConfig, logger *zap.Logger) *Client {
	return &Client{
		config:  cfg,
		http:    &http.Client{Timeout: cfg.Timeout},
		logger:  logger,
		enabled: cfg.Enabled,
	}
}

// NewSchedulerClient creates a Client from environment variables.
//
// Environment variables:
//
//	HAMI_ENABLED               - "true" to enable real HAMi scheduling (default: false)
//	HAMI_SCHEDULER_ENDPOINT    - HAMi extender URL (default: cluster DNS)
//	HAMI_SCHEDULER_TIMEOUT_SEC - HTTP timeout in seconds (default: 5)
func NewSchedulerClient(logger *zap.Logger) *Client {
	enabled := os.Getenv("HAMI_ENABLED") == "true"
	endpoint := os.Getenv("HAMI_SCHEDULER_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://hami-scheduler.hami-system.svc.cluster.local:9090"
	}

	timeoutSec := 5
	if s := os.Getenv("HAMI_SCHEDULER_TIMEOUT_SEC"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			timeoutSec = n
		}
	}

	cfg := ClientConfig{
		Endpoint: endpoint,
		Enabled:  enabled,
		Timeout:  time.Duration(timeoutSec) * time.Second,
	}

	return NewClient(cfg, logger)
}

// Schedule sends a scheduling request to the HAMi scheduler extender.
// In stub mode, it returns a simulated successful result immediately.
func (c *Client) Schedule(ctx context.Context, req ScheduleRequest) (*ScheduleResult, error) {
	if !c.enabled {
		c.logger.Debug("HAMi stub mode: simulating schedule",
			zap.String("endpoint_id", req.EndpointID),
			zap.String("model", req.ModelName),
		)
		return &ScheduleResult{
			Scheduled:         true,
			NodeName:          "stub-node",
			PhysicalGPUID:     0,
			AllocatedMemoryMiB: req.GPUSpec.MemoryMiB,
			AllocatedCores:    req.GPUSpec.Cores,
		}, nil
	}

	url := c.config.Endpoint + "/v1/schedule"
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal schedule request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create schedule request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("schedule request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("schedule request failed with status %d", resp.StatusCode)
	}

	var result ScheduleResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode schedule response: %w", err)
	}

	return &result, nil
}

// ListNodes returns GPU resource information for all nodes in the cluster.
// In stub mode, it returns an empty list.
func (c *Client) ListNodes(ctx context.Context) ([]NodeGPUInfo, error) {
	if !c.enabled {
		c.logger.Debug("HAMi stub mode: returning empty node list")
		return []NodeGPUInfo{}, nil
	}

	url := c.config.Endpoint + "/v1/nodes"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create list nodes request: %w", err)
	}

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("list nodes request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list nodes request failed with status %d", resp.StatusCode)
	}

	var nodes []NodeGPUInfo
	if err := json.NewDecoder(resp.Body).Decode(&nodes); err != nil {
		return nil, fmt.Errorf("decode list nodes response: %w", err)
	}

	return nodes, nil
}

// Ping checks connectivity to the HAMi scheduler.
// In stub mode, it always succeeds.
func (c *Client) Ping(ctx context.Context) error {
	if !c.enabled {
		return nil
	}

	url := c.config.Endpoint + "/healthz"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("create ping request: %w", err)
	}

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ping failed with status %d", resp.StatusCode)
	}

	return nil
}
