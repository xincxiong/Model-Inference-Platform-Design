// Package queue provides async message passing via Kafka (or an in-process fallback).
//
// The package exposes two independent components:
//
//   - Producer – publishes InferenceEvent / UsageEvent messages to Kafka topics.
//   - Consumer – reads messages and drives callback handlers.
//
// When Kafka brokers are not configured the package falls back to an in-process
// channel so the rest of the codebase compiles and runs without a real Kafka cluster.
package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// ── Event types ───────────────────────────────────────────────────────────

// EventType identifies the message schema.
type EventType string

const (
	EventTypeInference EventType = "inference"
	EventTypeUsage     EventType = "usage"
	EventTypeAudit     EventType = "audit"
)

// InferenceEvent is published when an inference request is dispatched.
type InferenceEvent struct {
	RequestID  string    `json:"request_id"`
	UserID     string    `json:"user_id"`
	ModelID    string    `json:"model_id"`
	BackendKey string    `json:"backend_key"`
	Endpoint   string    `json:"endpoint"` // /v1/chat/completions etc.
	Streaming  bool      `json:"streaming"`
	EnqueuedAt time.Time `json:"enqueued_at"`
}

// UsageEvent is published after billing is recorded.
type UsageEvent struct {
	RequestID    string    `json:"request_id"`
	UserID       string    `json:"user_id"`
	APIKeyID     string    `json:"api_key_id"`
	ModelID      string    `json:"model_id"`
	InputTokens  int       `json:"input_tokens"`
	OutputTokens int       `json:"output_tokens"`
	Cost         float64   `json:"cost"`
	FinishedAt   time.Time `json:"finished_at"`
}

// Message is the envelope written to Kafka.
type Message struct {
	Type    EventType       `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// ── Producer ──────────────────────────────────────────────────────────────

// Producer sends messages to the configured Kafka topic.
// When brokers is empty it falls back to an in-process channel.
type Producer struct {
	brokers []string
	topic   string
	logger  *zap.Logger
	// fallback channel used when Kafka is not configured
	fallback chan Message
	// kafka writer (lazily initialised)
	writer kafkaWriter
}

// kafkaWriter is an interface so we can swap in the real kafka-go writer
// and a no-op stub without importing kafka-go directly in this file.
type kafkaWriter interface {
	WriteMessages(ctx context.Context, msgs ...kafkaMessage) error
	Close() error
}

type kafkaMessage struct {
	Key   []byte
	Value []byte
}

// NewProducer creates a Producer.
//   - brokers: Kafka broker addresses (empty → in-process fallback).
//   - topic: destination topic.
func NewProducer(brokers []string, topic string, logger *zap.Logger) *Producer {
	p := &Producer{
		brokers:  brokers,
		topic:    topic,
		logger:   logger,
		fallback: make(chan Message, 10000),
	}
	if len(brokers) > 0 {
		p.writer = newKafkaWriter(brokers, topic)
	}
	return p
}

// Publish serialises and sends a message.
func (p *Producer) Publish(ctx context.Context, evType EventType, payload interface{}) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}
	msg := Message{Type: evType, Payload: raw}
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	if p.writer != nil {
		err := p.writer.WriteMessages(ctx, kafkaMessage{Value: body})
		if err != nil {
			p.logger.Warn("kafka write failed, falling back", zap.Error(err))
		} else {
			return nil
		}
	}

	// Fallback: drop into in-process channel (non-blocking).
	select {
	case p.fallback <- msg:
	default:
		p.logger.Warn("queue fallback channel full, dropping message",
			zap.String("type", string(evType)))
	}
	return nil
}

// PublishInference is a convenience wrapper for InferenceEvent.
func (p *Producer) PublishInference(ctx context.Context, ev InferenceEvent) {
	if err := p.Publish(ctx, EventTypeInference, ev); err != nil {
		p.logger.Error("publish inference event", zap.Error(err))
	}
}

// PublishUsage is a convenience wrapper for UsageEvent.
func (p *Producer) PublishUsage(ctx context.Context, ev UsageEvent) {
	if err := p.Publish(ctx, EventTypeUsage, ev); err != nil {
		p.logger.Error("publish usage event", zap.Error(err))
	}
}

// Fallback returns the in-process fallback channel (read-only).
// Useful for testing or when Kafka is unavailable.
func (p *Producer) Fallback() <-chan Message {
	return p.fallback
}

// Close shuts down the underlying Kafka writer (if any).
func (p *Producer) Close() error {
	if p.writer != nil {
		return p.writer.Close()
	}
	return nil
}
