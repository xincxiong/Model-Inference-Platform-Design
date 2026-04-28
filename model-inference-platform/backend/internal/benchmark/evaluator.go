// Package benchmark provides model evaluation benchmarking.
package benchmark

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// BenchmarkDataset represents a standard evaluation dataset.
type BenchmarkDataset string

const (
	DatasetMMLU      BenchmarkDataset = "mmlu"      // Massive Multitask Language Understanding
	DatasetHumanEval BenchmarkDataset = "humaneval" // HumanEval code generation
	DatasetGSM8K     BenchmarkDataset = "gsm8k"     // Grade school math
	DatasetMATH      BenchmarkDataset = "math"      // Math competition problems
	DatasetCEval     BenchmarkDataset = "ceval"     // Chinese evaluation
	DatasetCustom    BenchmarkDataset = "custom"    // User-defined benchmark
)

// BenchmarkConfig holds benchmark evaluation configuration.
type BenchmarkConfig struct {
	ID                string           `json:"id"`
	Name              string           `json:"name"`
	ModelID           string           `json:"model_id"`
	Datasets          []BenchmarkDataset `json:"datasets"`
	SamplesPerDataset int              `json:"samples_per_dataset"`
	Temperature       float64          `json:"temperature"`
	MaxTokens         int              `json:"max_tokens"`
}

// BenchmarkResult holds the evaluation results.
type BenchmarkResult struct {
	ID              string           `json:"id"`
	BenchmarkID     string           `json:"benchmark_id"`
	ModelID         string           `json:"model_id"`
	Dataset         BenchmarkDataset `json:"dataset"`
	Score           float64          `json:"score"`          // percentage correct
	TotalSamples    int              `json:"total_samples"`
	CorrectSamples  int              `json:"correct_samples"`
	AvgLatency      float64          `json:"avg_latency"`    // ms
	AvgTokens       float64          `json:"avg_tokens"`
	CostPerSample   float64          `json:"cost_per_sample"`
	EvaluatedAt     time.Time        `json:"evaluated_at"`
	Details         []SampleResult   `json:"details,omitempty"`
}

type SampleResult struct {
	Question  string  `json:"question"`
	Expected  string  `json:"expected"`
	Predicted string  `json:"predicted"`
	IsCorrect bool    `json:"is_correct"`
	Latency   float64 `json:"latency"`
	Tokens    int     `json:"tokens"`
}

// ComparisonResult holds before/after comparison.
type ComparisonResult struct {
	BenchmarkID   string            `json:"benchmark_id"`
	BeforeModelID string            `json:"before_model_id"`
	AfterModelID  string            `json:"after_model_id"`
	BeforeResults []BenchmarkResult `json:"before_results"`
	AfterResults  []BenchmarkResult `json:"after_results"`
	Improvements  map[BenchmarkDataset]float64 `json:"improvements"` // dataset -> percentage improvement
}

// Manager manages model benchmarking.
type Manager struct {
	rdb    *redis.Client
	logger *zap.Logger
}

// New creates a new benchmark manager.
func New(rdb *redis.Client, logger *zap.Logger) *Manager {
	return &Manager{
		rdb:    rdb,
		logger: logger,
	}
}

// RunBenchmark runs a benchmark evaluation (placeholder - actual evaluation requires model inference).
func (m *Manager) RunBenchmark(ctx context.Context, config BenchmarkConfig) (*BenchmarkResult, error) {
	if config.ID == "" {
		config.ID = "bench-" + strings.ReplaceAll(uuid.New().String(), "-", "")[:16]
	}

	results := make([]BenchmarkResult, 0, len(config.Datasets))

	for _, dataset := range config.Datasets {
		// Load dataset samples
		samples := loadDataset(dataset, config.SamplesPerDataset)
		
		// Evaluate each sample
		result := BenchmarkResult{
			ID:           fmt.Sprintf("%s-%s", config.ID, dataset),
			BenchmarkID:  config.ID,
			ModelID:      config.ModelID,
			Dataset:      dataset,
			TotalSamples: len(samples),
			EvaluatedAt:  time.Now(),
		}

		correctCount := 0
		totalLatency := 0.0
		totalTokens := 0

		for _, sample := range samples {
			// In production, this would call the model inference API
			// For now, we simulate evaluation
			sampleResult := evaluateSample(sample, config)
			result.Details = append(result.Details, sampleResult)
			
			if sampleResult.IsCorrect {
				correctCount++
			}
			totalLatency += sampleResult.Latency
			totalTokens += sampleResult.Tokens
		}

		result.CorrectSamples = correctCount
		if result.TotalSamples > 0 {
			result.Score = float64(correctCount) / float64(result.TotalSamples) * 100
		}
		result.AvgLatency = totalLatency / float64(len(samples))
		result.AvgTokens = float64(totalTokens) / float64(len(samples))

		results = append(results, result)

		// Store in Redis
		key := fmt.Sprintf("benchmark:%s:%s", config.ID, dataset)
		m.rdb.HSet(ctx, key, map[string]interface{}{
			"benchmark_id":    config.ID,
			"model_id":        config.ModelID,
			"dataset":         string(dataset),
			"score":           result.Score,
			"total_samples":   result.TotalSamples,
			"correct_samples": result.CorrectSamples,
			"avg_latency":     result.AvgLatency,
			"evaluated_at":    result.EvaluatedAt.Unix(),
		})
		
		// Also store as latest result for quick lookup
		latestKey := fmt.Sprintf("benchmark:latest:%s:%s", config.ModelID, dataset)
		m.rdb.HSet(ctx, latestKey, map[string]interface{}{
			"benchmark_id":    config.ID,
			"model_id":        config.ModelID,
			"dataset":         string(dataset),
			"score":           result.Score,
			"total_samples":   result.TotalSamples,
			"correct_samples": result.CorrectSamples,
			"avg_latency":     result.AvgLatency,
		})
	}

	m.logger.Info("benchmark completed",
		zap.String("benchmark_id", config.ID),
		zap.String("model_id", config.ModelID),
		zap.Int("datasets", len(results)))

	// Return first result for simplicity (in production, return all)
	if len(results) > 0 {
		return &results[0], nil
	}
	return nil, fmt.Errorf("no results generated")
}

// CompareModels compares benchmark results between two models.
func (m *Manager) CompareModels(ctx context.Context, beforeModelID, afterModelID string, datasets []BenchmarkDataset) (*ComparisonResult, error) {
	comparison := &ComparisonResult{
		BeforeModelID: beforeModelID,
		AfterModelID:  afterModelID,
		Improvements:  make(map[BenchmarkDataset]float64),
	}

	for _, dataset := range datasets {
		// Find latest results for both models
		beforeResult, _ := m.getLatestResult(ctx, beforeModelID, dataset)
		afterResult, _ := m.getLatestResult(ctx, afterModelID, dataset)

		if beforeResult != nil {
			comparison.BeforeResults = append(comparison.BeforeResults, *beforeResult)
		}
		if afterResult != nil {
			comparison.AfterResults = append(comparison.AfterResults, *afterResult)
		}

		// Calculate improvement
		if beforeResult != nil && afterResult != nil {
			improvement := afterResult.Score - beforeResult.Score
			comparison.Improvements[dataset] = improvement
		}
	}

	return comparison, nil
}

// GetResults retrieves benchmark results for a model.
func (m *Manager) GetResults(ctx context.Context, modelID string) ([]BenchmarkResult, error) {
	// Scan for all benchmark results for this model
	var results []BenchmarkResult
	
	cursor := uint64(0)
	for {
		keys, next, err := m.rdb.Scan(ctx, cursor, "benchmark:*", 100).Result()
		if err != nil {
			return nil, err
		}
		
		for _, key := range keys {
			vals, err := m.rdb.HGetAll(ctx, key).Result()
			if err != nil {
				continue
			}
			
			if vals["model_id"] != modelID {
				continue
			}
			
			result := BenchmarkResult{
				BenchmarkID: vals["benchmark_id"],
				ModelID:     vals["model_id"],
				Dataset:     BenchmarkDataset(vals["dataset"]),
			}
			
			fmt.Sscanf(vals["score"], "%f", &result.Score)
			fmt.Sscanf(vals["total_samples"], "%d", &result.TotalSamples)
			fmt.Sscanf(vals["correct_samples"], "%d", &result.CorrectSamples)
			fmt.Sscanf(vals["avg_latency"], "%f", &result.AvgLatency)
			
			results = append(results, result)
		}
		
		cursor = next
		if cursor == 0 {
			break
		}
	}
	
	return results, nil
}

func (m *Manager) getLatestResult(ctx context.Context, modelID string, dataset BenchmarkDataset) (*BenchmarkResult, error) {
	key := fmt.Sprintf("benchmark:latest:%s:%s", modelID, dataset)
	vals, err := m.rdb.HGetAll(ctx, key).Result()
	if err != nil || len(vals) == 0 {
		return nil, err
	}
	
	result := &BenchmarkResult{
		BenchmarkID: vals["benchmark_id"],
		ModelID:     vals["model_id"],
		Dataset:     BenchmarkDataset(vals["dataset"]),
	}
	
	fmt.Sscanf(vals["score"], "%f", &result.Score)
	fmt.Sscanf(vals["total_samples"], "%d", &result.TotalSamples)
	fmt.Sscanf(vals["correct_samples"], "%d", &result.CorrectSamples)
	fmt.Sscanf(vals["avg_latency"], "%f", &result.AvgLatency)
	
	return result, nil
}

// loadDataset loads benchmark samples (placeholder - in production, load from S3 or database).
func loadDataset(dataset BenchmarkDataset, count int) []BenchmarkSample {
	// Placeholder implementation
	samples := make([]BenchmarkSample, count)
	for i := 0; i < count; i++ {
		samples[i] = BenchmarkSample{
			Question: fmt.Sprintf("Sample question %d for %s", i, dataset),
			Expected: fmt.Sprintf("Expected answer %d", i),
		}
	}
	return samples
}

// BenchmarkSample represents a single evaluation sample.
type BenchmarkSample struct {
	Question string
	Expected string
}

// evaluateSample evaluates a single sample (placeholder).
func evaluateSample(sample BenchmarkSample, config BenchmarkConfig) SampleResult {
	// In production, this would:
	// 1. Send question to model API
	// 2. Measure latency
	// 3. Compare predicted vs expected
	// 4. Return result
	
	return SampleResult{
		Question:  sample.Question,
		Expected:  sample.Expected,
		Predicted: "Simulated answer",
		IsCorrect: true, // placeholder
		Latency:   150.0, // placeholder ms
		Tokens:    50,    // placeholder
	}
}
