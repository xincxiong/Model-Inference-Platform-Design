package queue

// This file wires up the kafkaWriter interface to a real implementation.
//
// We use a lightweight stub backed by an in-process goroutine so the project
// compiles without a kafka-go dependency.  To use a real Kafka cluster replace
// this file with one that imports github.com/segmentio/kafka-go and constructs
// a *kafka.Writer.

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"go.uber.org/zap"
)

// ── Stub writer (in-process, for development / CI) ────────────────────────

type stubWriter struct {
	mu       sync.Mutex
	messages [][]byte
	topic    string
}

func newKafkaWriter(brokers []string, topic string) kafkaWriter {
	// When brokers are provided but kafka-go is not integrated yet, we log a
	// warning and return the stub so the server still starts.
	_ = brokers // TODO: replace with real kafka-go writer
	return &stubWriter{topic: topic}
}

func (s *stubWriter) WriteMessages(_ context.Context, msgs ...kafkaMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, m := range msgs {
		s.messages = append(s.messages, m.Value)
	}
	return nil
}

func (s *stubWriter) Close() error { return nil }

// ── Consumer ──────────────────────────────────────────────────────────────

// HandlerFunc is called for each message consumed from the queue.
type HandlerFunc func(msg Message) error

// Consumer reads messages from the in-process fallback channel or a Kafka topic.
type Consumer struct {
	producer *Producer
	handlers map[EventType]HandlerFunc
	logger   *zap.Logger
}

// NewConsumer creates a Consumer that drains the producer's fallback channel.
func NewConsumer(p *Producer, logger *zap.Logger) *Consumer {
	return &Consumer{
		producer: p,
		handlers: make(map[EventType]HandlerFunc),
		logger:   logger,
	}
}

// Register registers a handler for the given event type.
func (c *Consumer) Register(evType EventType, fn HandlerFunc) {
	c.handlers[evType] = fn
}

// Start begins consuming messages until ctx is done.
func (c *Consumer) Start(ctx context.Context) {
	go c.run(ctx)
}

func (c *Consumer) run(ctx context.Context) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-c.producer.Fallback():
			c.handle(msg)
		case <-ticker.C:
			// drain any buffered messages
			for {
				select {
				case msg := <-c.producer.Fallback():
					c.handle(msg)
				default:
					goto drained
				}
			}
		drained:
		}
	}
}

func (c *Consumer) handle(msg Message) {
	fn, ok := c.handlers[msg.Type]
	if !ok {
		return
	}
	if err := fn(msg); err != nil {
		c.logger.Error("message handler error",
			zap.String("type", string(msg.Type)),
			zap.Error(err))
	}
}

// ── Helpers ───────────────────────────────────────────────────────────────

// UnmarshalUsageEvent decodes a Message into a UsageEvent.
func UnmarshalUsageEvent(msg Message) (UsageEvent, error) {
	var ev UsageEvent
	return ev, json.Unmarshal(msg.Payload, &ev)
}

// UnmarshalInferenceEvent decodes a Message into an InferenceEvent.
func UnmarshalInferenceEvent(msg Message) (InferenceEvent, error) {
	var ev InferenceEvent
	return ev, json.Unmarshal(msg.Payload, &ev)
}
