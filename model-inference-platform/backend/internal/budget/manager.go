// Package budget provides cost budget alerting with multi-level thresholds.
package budget

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// AlertChannel defines where alerts are sent.
type AlertChannel string

const (
	ChannelEmail  AlertChannel = "email"
	ChannelSlack  AlertChannel = "slack"
	ChannelWebhook AlertChannel = "webhook"
)

// AlertThreshold defines a budget alert threshold.
type AlertThreshold struct {
	Percentage float64       // e.g., 50, 80, 100
	Channels   []AlertChannel // where to send the alert
	Message    string        // custom alert message
}

// BudgetConfig holds budget alert configuration for a user/project.
type BudgetConfig struct {
	ID         string           // budget ID (user_id or project_id)
	Scope      string           // "user" or "project"
	BudgetAmount float64        // total budget amount
	Thresholds []AlertThreshold // alert thresholds
	NotifyURL  string           // webhook/Slack URL for notifications
}

// DefaultBudgetThresholds returns standard alert thresholds.
func DefaultBudgetThresholds() []AlertThreshold {
	return []AlertThreshold{
		{Percentage: 50, Channels: []AlertChannel{ChannelEmail}, Message: "Budget usage reached 50%"},
		{Percentage: 80, Channels: []AlertChannel{ChannelEmail, ChannelSlack}, Message: "Budget usage reached 80%"},
		{Percentage: 100, Channels: []AlertChannel{ChannelEmail, ChannelSlack}, Message: "Budget exhausted (100%)"},
	}
}

// AlertRecord stores a sent alert for history.
type AlertRecord struct {
	ID        string    `json:"id"`
	BudgetID  string    `json:"budget_id"`
	Threshold float64   `json:"threshold"`
	SentAt    time.Time `json:"sent_at"`
	Channel   string    `json:"channel"`
	Message   string    `json:"message"`
}

// Manager manages budget alerts.
type Manager struct {
	rdb      *redis.Client
	logger   *zap.Logger
	configs  map[string]*BudgetConfig
	mu       sync.RWMutex
	alertedThresholds map[string]map[float64]bool // budgetID -> threshold% -> alerted
	notifyFn func(channel AlertChannel, config BudgetConfig, threshold float64, currentSpend float64) error
}

// New creates a new budget alert manager.
func New(rdb *redis.Client, logger *zap.Logger, notifyFn func(channel AlertChannel, config BudgetConfig, threshold float64, currentSpend float64) error) *Manager {
	return &Manager{
		rdb:               rdb,
		logger:            logger,
		configs:           make(map[string]*BudgetConfig),
		alertedThresholds: make(map[string]map[float64]bool),
		notifyFn:          notifyFn,
	}
}

// Register registers a budget for alerting.
func (m *Manager) Register(ctx context.Context, config BudgetConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.configs[config.ID] = &config
	m.alertedThresholds[config.ID] = make(map[float64]bool)
	
	// Restore already-alerted state from Redis
	for _, t := range config.Thresholds {
		key := fmt.Sprintf("budget:%s:alerted:%.0f", config.ID, t.Percentage)
		alerted, _ := m.rdb.Get(ctx, key).Bool()
		if alerted {
			m.alertedThresholds[config.ID][t.Percentage] = true
		}
	}
	
	m.logger.Info("registered budget alert",
		zap.String("budget_id", config.ID),
		zap.Float64("budget_amount", config.BudgetAmount),
		zap.Int("thresholds", len(config.Thresholds)))
}

// CheckBudget checks current spend against budget thresholds and sends alerts if needed.
func (m *Manager) CheckBudget(ctx context.Context, budgetID string, currentSpend float64) error {
	m.mu.RLock()
	config, exists := m.configs[budgetID]
	alerted := m.alertedThresholds[budgetID]
	m.mu.RUnlock()
	
	if !exists {
		return nil // no budget configured
	}
	
	if config.BudgetAmount == 0 {
		return nil
	}
	
	usagePercentage := (currentSpend / config.BudgetAmount) * 100
	
	for _, threshold := range config.Thresholds {
		if usagePercentage >= threshold.Percentage && !alerted[threshold.Percentage] {
			// Send alerts
			for _, channel := range threshold.Channels {
				msg := threshold.Message
				if msg == "" {
					msg = fmt.Sprintf("Budget alert: %.1f%% used ($%.2f / $%.2f)", 
						threshold.Percentage, currentSpend, config.BudgetAmount)
				}
				
				if m.notifyFn != nil {
					if err := m.notifyFn(channel, *config, threshold.Percentage, currentSpend); err != nil {
						m.logger.Error("failed to send budget alert",
							zap.String("budget_id", budgetID),
							zap.String("channel", string(channel)),
							zap.Error(err))
						continue
					}
				}
				
				m.logger.Warn("budget alert triggered",
					zap.String("budget_id", budgetID),
					zap.Float64("threshold", threshold.Percentage),
					zap.Float64("current_spend", currentSpend),
					zap.String("channel", string(channel)))
			}
			
			// Mark as alerted
			alerted[threshold.Percentage] = true
			
			// Persist to Redis
			key := fmt.Sprintf("budget:%s:alerted:%.0f", budgetID, threshold.Percentage)
			m.rdb.Set(ctx, key, true, 30*24*time.Hour) // keep for 30 days
			
			// Record alert history
			m.recordAlert(ctx, AlertRecord{
				ID:        fmt.Sprintf("alert-%s-%.0f-%d", budgetID, threshold.Percentage, time.Now().Unix()),
				BudgetID:  budgetID,
				Threshold: threshold.Percentage,
				SentAt:    time.Now(),
				Channel:   string(threshold.Channels[0]),
				Message:   threshold.Message,
			})
		}
	}
	
	return nil
}

// recordAlert stores an alert record for history.
func (m *Manager) recordAlert(ctx context.Context, record AlertRecord) {
	key := fmt.Sprintf("budget:%s:alerts", record.BudgetID)
	data := fmt.Sprintf("%s:%.0f:%s:%s", record.SentAt.Format(time.RFC3339), record.Threshold, record.Channel, record.Message)
	m.rdb.LPush(ctx, key, data)
	m.rdb.LTrim(ctx, key, 0, 99) // keep last 100 alerts
}

// GetAlertHistory retrieves recent alert history for a budget.
func (m *Manager) GetAlertHistory(ctx context.Context, budgetID string) ([]string, error) {
	key := fmt.Sprintf("budget:%s:alerts", budgetID)
	return m.rdb.LRange(ctx, key, 0, -1).Result()
}

// ResetBudget clears alerted thresholds for a budget.
func (m *Manager) ResetBudget(ctx context.Context, budgetID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if alerted, exists := m.alertedThresholds[budgetID]; exists {
		for threshold := range alerted {
			key := fmt.Sprintf("budget:%s:alerted:%.0f", budgetID, threshold)
			m.rdb.Del(ctx, key)
		}
		m.alertedThresholds[budgetID] = make(map[float64]bool)
	}
}
