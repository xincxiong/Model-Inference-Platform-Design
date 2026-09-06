package model

import "time"

// Subscription lifecycle (see PRD 4.x for full state machine).
const (
	SubStatusPending   = "pending"   // 等待支付
	SubStatusActive    = "active"    // 生效中
	SubStatusOverdue   = "overdue"   // 欠费
	SubStatusSuspended = "suspended" // 已冻结
	SubStatusExpired   = "expired"   // 到期未续
	SubStatusCancelled = "cancelled" // 主动取消
)

const (
	PaymentPending = "pending"
	PaymentPaid    = "paid"
	PaymentPartial = "partial"
	PaymentRefunded = "refunded"
)

// PurchasePoolRequest is the body for POST /v0/pools/subscriptions.
type PurchasePoolRequest struct {
	SKUID    string `json:"sku_id" binding:"required"`
	AutoRenew bool  `json:"auto_renew"`
	Trial    bool   `json:"trial"` // true = 7 天试用
}

// PoolSubscription is a user-purchased compute-pool plan.
type PoolSubscription struct {
	ID            string     `json:"id"`
	UserID        string     `json:"user_id"`
	SKUID         string     `json:"sku_id"`
	SKUName       string     `json:"sku_name,omitempty"`     // joined for display
	GPUType       string     `json:"gpu_type,omitempty"`     // joined
	GPUCount      int        `json:"gpu_count,omitempty"`    // joined
	PoolID        *string    `json:"pool_id,omitempty"`
	TotalAmount   float64    `json:"total_amount"`
	Currency      string     `json:"currency"`
	StartAt       time.Time  `json:"start_at"`
	EndAt         time.Time  `json:"end_at"`
	AutoRenew     bool       `json:"auto_renew"`
	Status        string     `json:"status"`
	PaymentStatus string     `json:"payment_status"`
	RenewedFromID *string    `json:"renewed_from_id,omitempty"`
	Trial         bool       `json:"trial"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// ListPoolSubscriptionsResponse is the response for GET /v0/pools/subscriptions.
type ListPoolSubscriptionsResponse struct {
	Data []PoolSubscription `json:"data"`
}

// CancelSubscriptionRequest is the body for POST /v0/pools/subscriptions/:id/cancel.
type CancelSubscriptionRequest struct {
	Reason string `json:"reason"`
	// If true, cancel takes effect immediately; otherwise at period end.
	Immediate bool `json:"immediate"`
}

// PoolInvoice is a billing record tied to a subscription period.
type PoolInvoice struct {
	ID             string     `json:"id"`
	SubscriptionID string     `json:"subscription_id"`
	PeriodStart    time.Time  `json:"period_start"`
	PeriodEnd      time.Time  `json:"period_end"`
	Amount         float64    `json:"amount"`
	Status         string     `json:"status"` // pending/paid/overdue/cancelled
	DueAt          time.Time  `json:"due_at"`
	PaidAt         *time.Time `json:"paid_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

// ListPoolInvoicesResponse is the response for GET /v0/pools/subscriptions/:id/invoices.
type ListPoolInvoicesResponse struct {
	Data []PoolInvoice `json:"data"`
}

// SubscriptionOverview gives the user a top-level summary for the /pools page header.
type SubscriptionOverview struct {
	ActiveCount      int     `json:"active_count"`
	MonthlySpend     float64 `json:"monthly_spend"`      // 当前所有 active 订阅的月折算总额
	UpcomingRenewals int     `json:"upcoming_renewals"`  // 7 天内到期的 active 订阅数
	TotalSavings     float64 `json:"total_savings"`      // 累计对比按量的节省
	Currency         string  `json:"currency"`
}
