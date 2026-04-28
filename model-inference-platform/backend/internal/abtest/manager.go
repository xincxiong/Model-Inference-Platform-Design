// Package abtest provides A/B testing framework for model traffic splitting.
package abtest

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Variant represents a model variant in an A/B test.
type Variant struct {
	ModelID       string  `json:"model_id"`
	Weight        float64 `json:"weight"`        // traffic percentage (0-1)
	TotalRequests int64   `json:"total_requests"`
	TotalTokens   int64   `json:"total_tokens"`
	AvgLatency    float64 `json:"avg_latency"`   // milliseconds
	AvgCost       float64 `json:"avg_cost"`      // dollars per request
	SuccessCount  int64   `json:"success_count"`
	ErrorCount    int64   `json:"error_count"`
}

// ABTest represents an A/B test configuration.
type ABTest struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	EndpointID  string     `json:"endpoint_id"`
	Variants    []Variant  `json:"variants"`
	StartTime   time.Time  `json:"start_time"`
	EndTime     *time.Time `json:"end_time,omitempty"`
	IsActive    bool       `json:"is_active"`
	MinSamples  int64      `json:"min_samples"` // minimum samples per variant for significance
	Confidence  float64    `json:"confidence"`  // confidence level (e.g., 0.95)
}

// TestResult holds the statistical analysis result.
type TestResult struct {
	IsSignificant   bool            `json:"is_significant"`
	WinnerModelID   string          `json:"winner_model_id,omitempty"`
	ConfidenceLevel float64         `json:"confidence_level"`
	PValue          float64         `json:"p_value"`
	Recommendation  string          `json:"recommendation"`
	VariantResults  []VariantResult `json:"variant_results"`
}

type VariantResult struct {
	ModelID       string  `json:"model_id"`
	TotalRequests int64   `json:"total_requests"`
	AvgLatency    float64 `json:"avg_latency"`
	AvgCost       float64 `json:"avg_cost"`
	SuccessRate   float64 `json:"success_rate"`
}

// Manager manages A/B tests.
type Manager struct {
	rdb    *redis.Client
	logger *zap.Logger
	mu     sync.RWMutex
	tests  map[string]*ABTest // endpointID -> ABTest
}

// New creates a new A/B test manager.
func New(rdb *redis.Client, logger *zap.Logger) *Manager {
	return &Manager{
		rdb:    rdb,
		logger: logger,
		tests:  make(map[string]*ABTest),
	}
}

// Create creates a new A/B test.
func (m *Manager) Create(ctx context.Context, test *ABTest) error {
	// Validate weights sum to 1.0
	totalWeight := 0.0
	for _, v := range test.Variants {
		totalWeight += v.Weight
	}
	if math.Abs(totalWeight-1.0) > 0.01 {
		return fmt.Errorf("variant weights must sum to 1.0, got %.2f", totalWeight)
	}

	test.IsActive = true
	test.StartTime = time.Now()
	if test.MinSamples == 0 {
		test.MinSamples = 100
	}
	if test.Confidence == 0 {
		test.Confidence = 0.95
	}

	// Store in Redis
	key := fmt.Sprintf("abtest:%s", test.EndpointID)
	testData := map[string]interface{}{
		"id":          test.ID,
		"name":        test.Name,
		"endpoint_id": test.EndpointID,
		"variants":    variantsToJSON(test.Variants),
		"start_time":  test.StartTime.Unix(),
		"is_active":   test.IsActive,
		"min_samples": test.MinSamples,
		"confidence":  test.Confidence,
	}

	if err := m.rdb.HSet(ctx, key, testData).Err(); err != nil {
		return err
	}

	m.mu.Lock()
	m.tests[test.EndpointID] = test
	m.mu.Unlock()

	m.logger.Info("created A/B test",
		zap.String("test_id", test.ID),
		zap.String("endpoint_id", test.EndpointID),
		zap.Int("variants", len(test.Variants)))

	return nil
}

// SelectVariant selects a variant for a request based on traffic weights.
func (m *Manager) SelectVariant(ctx context.Context, endpointID string) (string, error) {
	m.mu.RLock()
	test, exists := m.tests[endpointID]
	m.mu.RUnlock()

	if !exists || !test.IsActive {
		return "", nil // no active test, use default model
	}

	// Select variant based on weights
	r := rand.Float64()
	cumulative := 0.0

	for _, v := range test.Variants {
		cumulative += v.Weight
		if r <= cumulative {
			return v.ModelID, nil
		}
	}

	// Fallback to last variant
	return test.Variants[len(test.Variants)-1].ModelID, nil
}

// RecordMetrics records request metrics for a variant.
func (m *Manager) RecordMetrics(ctx context.Context, endpointID, modelID string, latency float64, cost float64, tokens int64, success bool) {
	m.mu.RLock()
	test, exists := m.tests[endpointID]
	m.mu.RUnlock()

	if !exists {
		return
	}

	// Update variant metrics in Redis
	key := fmt.Sprintf("abtest:%s:variant:%s", endpointID, modelID)
	pipe := m.rdb.Pipeline()
	pipe.HIncrBy(ctx, key, "total_requests", 1)
	pipe.HIncrBy(ctx, key, "total_tokens", tokens)
	
	// Update running average for latency
	currentAvg, _ := m.rdb.HGet(ctx, key, "avg_latency").Float64()
	currentCount, _ := m.rdb.HGet(ctx, key, "total_requests").Int64()
	if currentCount > 0 {
		newAvg := (currentAvg*float64(currentCount-1) + latency) / float64(currentCount)
		pipe.HSet(ctx, key, "avg_latency", newAvg)
	}

	// Update running average for cost
	currentCostAvg, _ := m.rdb.HGet(ctx, key, "avg_cost").Float64()
	if currentCount > 0 {
		newCostAvg := (currentCostAvg*float64(currentCount-1) + cost) / float64(currentCount)
		pipe.HSet(ctx, key, "avg_cost", newCostAvg)
	}

	if success {
		pipe.HIncrBy(ctx, key, "success_count", 1)
	} else {
		pipe.HIncrBy(ctx, key, "error_count", 1)
	}

	pipe.Exec(ctx)

	// Update in-memory copy
	for i, v := range test.Variants {
		if v.ModelID == modelID {
			test.Variants[i].TotalRequests++
			test.Variants[i].TotalTokens += tokens
			break
		}
	}
}

// Analyze performs statistical analysis on the A/B test.
func (m *Manager) Analyze(ctx context.Context, endpointID string) (*TestResult, error) {
	m.mu.RLock()
	test, exists := m.tests[endpointID]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("no A/B test found for endpoint %s", endpointID)
	}

	result := &TestResult{
		VariantResults: make([]VariantResult, 0, len(test.Variants)),
	}

	// Collect metrics for each variant
	for _, v := range test.Variants {
		key := fmt.Sprintf("abtest:%s:variant:%s", endpointID, v.ModelID)
		
		totalRequests, _ := m.rdb.HGet(ctx, key, "total_requests").Int64()
		_, _ = m.rdb.HGet(ctx, key, "total_tokens").Int64() // totalTokens - reserved for future use
		avgLatency, _ := m.rdb.HGet(ctx, key, "avg_latency").Float64()
		avgCost, _ := m.rdb.HGet(ctx, key, "avg_cost").Float64()
		successCount, _ := m.rdb.HGet(ctx, key, "success_count").Int64()
		_, _ = m.rdb.HGet(ctx, key, "error_count").Int64() // errorCount - reserved for future use

		successRate := 0.0
		if totalRequests > 0 {
			successRate = float64(successCount) / float64(totalRequests) * 100
		}

		result.VariantResults = append(result.VariantResults, VariantResult{
			ModelID:       v.ModelID,
			TotalRequests: totalRequests,
			AvgLatency:    avgLatency,
			AvgCost:       avgCost,
			SuccessRate:   successRate,
		})
	}

	// Check if we have enough samples
	for _, vr := range result.VariantResults {
		if vr.TotalRequests < test.MinSamples {
			result.IsSignificant = false
			result.Recommendation = fmt.Sprintf("Need more samples. Current: %d, Required: %d per variant", vr.TotalRequests, test.MinSamples)
			return result, nil
		}
	}

	// Perform simplified statistical analysis
	result.IsSignificant, result.WinnerModelID, result.PValue = calculateSignificance(result.VariantResults, test.Confidence)

	if result.IsSignificant {
		result.Recommendation = fmt.Sprintf("Statistically significant! %s is the winner (p=%.4f)", result.WinnerModelID, result.PValue)
	} else {
		result.Recommendation = "No statistically significant difference detected yet. Continue the test."
	}

	return result, nil
}

// SwitchTraffic switches 100% traffic to a specific variant.
func (m *Manager) SwitchTraffic(ctx context.Context, endpointID, modelID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	test, exists := m.tests[endpointID]
	if !exists {
		return fmt.Errorf("no A/B test found for endpoint %s", endpointID)
	}

	// Update weights
	for i := range test.Variants {
		if test.Variants[i].ModelID == modelID {
			test.Variants[i].Weight = 1.0
		} else {
			test.Variants[i].Weight = 0.0
		}
	}

	// Update Redis
	key := fmt.Sprintf("abtest:%s", endpointID)
	m.rdb.HSet(ctx, key, "variants", variantsToJSON(test.Variants))

	m.logger.Info("switched traffic",
		zap.String("endpoint_id", endpointID),
		zap.String("winner", modelID))

	return nil
}

// Stop stops an A/B test.
func (m *Manager) Stop(ctx context.Context, endpointID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	test, exists := m.tests[endpointID]
	if !exists {
		return fmt.Errorf("no A/B test found for endpoint %s", endpointID)
	}

	test.IsActive = false
	now := time.Now()
	test.EndTime = &now

	key := fmt.Sprintf("abtest:%s", endpointID)
	m.rdb.HSet(ctx, key, "is_active", false, "end_time", now.Unix())

	m.logger.Info("stopped A/B test",
		zap.String("endpoint_id", endpointID),
		zap.String("test_id", test.ID))

	return nil
}

// calculateSignificance performs a simplified statistical test.
func calculateSignificance(variants []VariantResult, confidence float64) (bool, string, float64) {
	if len(variants) < 2 {
		return false, "", 1.0
	}

	// Find best variant by success rate (simplified)
	bestIdx := 0
	bestRate := variants[0].SuccessRate

	for i, v := range variants {
		if v.SuccessRate > bestRate {
			bestRate = v.SuccessRate
			bestIdx = i
		}
	}

	// Simplified p-value calculation (in production, use proper statistical library)
	// This is a placeholder - real implementation would use chi-squared or t-test
	pValue := 0.05 // placeholder

	isSignificant := pValue < (1.0 - confidence)

	return isSignificant, variants[bestIdx].ModelID, pValue
}

func variantsToJSON(variants []Variant) string {
	// Simplified - in production use proper JSON marshaling
	result := ""
	for i, v := range variants {
		if i > 0 {
			result += ","
		}
		result += fmt.Sprintf(`{"model_id":"%s","weight":%.2f}`, v.ModelID, v.Weight)
	}
	return "[" + result + "]"
}
