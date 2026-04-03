package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/xincxiong/model-inference-platform/backend/internal/middleware"
	"github.com/xincxiong/model-inference-platform/backend/internal/model"
	"github.com/xincxiong/model-inference-platform/backend/internal/store"
)

type BillingHandler struct {
	store *store.Store
}

func NewBillingHandler(s *store.Store) *BillingHandler {
	return &BillingHandler{store: s}
}

func (h *BillingHandler) GetUsage(c *gin.Context) {
	auth := middleware.GetAuthInfo(c)

	var balance float64
	_ = h.store.DB.QueryRow(context.Background(),
		`SELECT COALESCE(balance, 0) FROM billing_accounts WHERE user_id = $1`, auth.UserID).
		Scan(&balance)

	var totalSpent float64
	var totalTokens int64
	_ = h.store.DB.QueryRow(context.Background(),
		`SELECT COALESCE(SUM(cost), 0), COALESCE(SUM(input_tokens + output_tokens), 0)
		 FROM usage_records WHERE user_id = $1`, auth.UserID).
		Scan(&totalSpent, &totalTokens)

	rows, err := h.store.DB.Query(context.Background(),
		`SELECT DATE(created_at) as date,
		        SUM(input_tokens) as input_tokens,
		        SUM(output_tokens) as output_tokens,
		        SUM(cost) as cost
		 FROM usage_records WHERE user_id = $1
		 GROUP BY DATE(created_at) ORDER BY date DESC LIMIT 30`, auth.UserID)

	var daily []model.DailyUsage
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var d model.DailyUsage
			if err := rows.Scan(&d.Date, &d.InputTokens, &d.OutputTokens, &d.Cost); err != nil {
				continue
			}
			daily = append(daily, d)
		}
	}

	if daily == nil {
		daily = []model.DailyUsage{}
	}

	c.JSON(http.StatusOK, model.UsageSummary{
		Balance:        balance,
		TotalSpent:     totalSpent,
		TotalTokens:    totalTokens,
		DailyBreakdown: daily,
	})
}

func (h *BillingHandler) RedeemPromo(c *gin.Context) {
	var req model.RedeemPromoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}

	auth := middleware.GetAuthInfo(c)

	var amount float64
	err := h.store.DB.QueryRow(context.Background(),
		`UPDATE promo_codes SET used = TRUE, used_by_id = $1
		 WHERE code = $2 AND used = FALSE RETURNING amount`,
		auth.UserID, req.Code).Scan(&amount)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "Invalid or already used promo code"}})
		return
	}

	_, _ = h.store.DB.Exec(context.Background(),
		`UPDATE billing_accounts SET balance = balance + $1 WHERE user_id = $2`,
		amount, auth.UserID)

	c.JSON(http.StatusOK, gin.H{"message": "Promo code redeemed", "amount": amount})
}
