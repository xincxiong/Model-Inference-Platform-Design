package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/xincxiong/model-inference-platform/backend/internal/middleware"
	"github.com/xincxiong/model-inference-platform/backend/internal/model"
	"github.com/xincxiong/model-inference-platform/backend/internal/store"
)

type PoolSubscriptionsHandler struct {
	store *store.Store
}

func NewPoolSubscriptionsHandler(s *store.Store) *PoolSubscriptionsHandler {
	return &PoolSubscriptionsHandler{store: s}
}

// ListSKUs GET /v0/pools/skus — public-ish; no auth required for browsing the catalog.
func (h *PoolSubscriptionsHandler) ListSKUs(c *gin.Context) {
	var q model.ListPoolSKUsQuery
	_ = c.ShouldBindQuery(&q)

	sql := `SELECT id, name, description, gpu_type, gpu_count, region, sharing_mode,
		term, term_months, hourly_list_price, term_price, discount_pct, sla_class,
		active, sort_order, created_at
		FROM pool_skus WHERE 1=1`
	args := []any{}
	i := 1
	if q.GPUType != "" {
		sql += " AND gpu_type = $" + strconv.Itoa(i)
		args = append(args, q.GPUType)
		i++
	}
	if q.Region != "" {
		sql += " AND region = $" + strconv.Itoa(i)
		args = append(args, q.Region)
		i++
	}
	if q.SharingMode != "" {
		sql += " AND sharing_mode = $" + strconv.Itoa(i)
		args = append(args, q.SharingMode)
		i++
	}
	if q.Term != "" {
		sql += " AND term = $" + strconv.Itoa(i)
		args = append(args, q.Term)
		i++
	}
	if q.ActiveOnly {
		sql += " AND active = TRUE"
	}
	sql += " ORDER BY sort_order, id"

	rows, err := h.store.DB.Query(c.Request.Context(), sql, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}
	defer rows.Close()

	var data []model.PoolSKU
	for rows.Next() {
		var s model.PoolSKU
		if err := rows.Scan(&s.ID, &s.Name, &s.Description, &s.GPUType, &s.GPUCount,
			&s.Region, &s.SharingMode, &s.Term, &s.TermMonths,
			&s.HourlyListPrice, &s.TermPrice, &s.DiscountPct, &s.SLAClass,
			&s.Active, &s.SortOrder, &s.CreatedAt); err != nil {
			continue
		}
		data = append(data, s)
	}
	c.JSON(http.StatusOK, model.ListPoolSKUsResponse{Data: data})
}

// ListSubscriptions GET /v0/pools/subscriptions — current user's subscriptions.
func (h *PoolSubscriptionsHandler) ListSubscriptions(c *gin.Context) {
	auth := middleware.GetAuthInfo(c)
	rows, err := h.store.DB.Query(c.Request.Context(), `
		SELECT s.id, s.user_id, s.sku_id, sk.name, sk.gpu_type, sk.gpu_count,
			s.pool_id, s.total_amount, s.currency, s.start_at, s.end_at,
			s.auto_renew, s.status, s.payment_status, s.renewed_from_id, s.trial,
			s.created_at, s.updated_at
		FROM pool_subscriptions s
		JOIN pool_skus sk ON sk.id = s.sku_id
		WHERE s.user_id = $1
		ORDER BY s.created_at DESC`, auth.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}
	defer rows.Close()

	var data []model.PoolSubscription
	for rows.Next() {
		var s model.PoolSubscription
		if err := rows.Scan(&s.ID, &s.UserID, &s.SKUID, &s.SKUName, &s.GPUType, &s.GPUCount,
			&s.PoolID, &s.TotalAmount, &s.Currency, &s.StartAt, &s.EndAt,
			&s.AutoRenew, &s.Status, &s.PaymentStatus, &s.RenewedFromID, &s.Trial,
			&s.CreatedAt, &s.UpdatedAt); err != nil {
			continue
		}
		data = append(data, s)
	}
	c.JSON(http.StatusOK, model.ListPoolSubscriptionsResponse{Data: data})
}

// Overview GET /v0/pools/subscriptions/overview — top summary card.
func (h *PoolSubscriptionsHandler) Overview(c *gin.Context) {
	auth := middleware.GetAuthInfo(c)
	var ov model.SubscriptionOverview
	ov.Currency = "CNY"

	// Active count + monthly spend
	row := h.store.DB.QueryRow(c.Request.Context(), `
		SELECT COUNT(*), COALESCE(SUM(
			CASE WHEN sk.term='monthly'  THEN s.total_amount
			     WHEN sk.term='quarterly' THEN s.total_amount / 3
			     WHEN sk.term='yearly'    THEN s.total_amount / 12
			     ELSE 0 END
		), 0)
		FROM pool_subscriptions s
		JOIN pool_skus sk ON sk.id = s.sku_id
		WHERE s.user_id = $1 AND s.status = 'active'`, auth.UserID)
	_ = row.Scan(&ov.ActiveCount, &ov.MonthlySpend)

	// Upcoming renewals (within 7 days)
	_ = h.store.DB.QueryRow(c.Request.Context(), `
		SELECT COUNT(*) FROM pool_subscriptions
		WHERE user_id = $1 AND status = 'active'
		  AND end_at BETWEEN NOW() AND NOW() + INTERVAL '7 days'`, auth.UserID).
		Scan(&ov.UpcomingRenewals)

	// Total savings = sum(term_price vs hourly_list_price*hours)
	_ = h.store.DB.QueryRow(c.Request.Context(), `
		SELECT COALESCE(SUM(
			(sk.hourly_list_price * sk.gpu_count * sk.term_months * 30 * 24) - s.total_amount
		), 0)
		FROM pool_subscriptions s
		JOIN pool_skus sk ON sk.id = s.sku_id
		WHERE s.user_id = $1 AND s.payment_status = 'paid'`, auth.UserID).
		Scan(&ov.TotalSavings)

	c.JSON(http.StatusOK, ov)
}

// Purchase POST /v0/pools/subscriptions — create a subscription.
// In P0 the payment is mocked (auto-paid); P1 can swap in real payment gateway.
func (h *PoolSubscriptionsHandler) Purchase(c *gin.Context) {
	var req model.PurchasePoolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}
	auth := middleware.GetAuthInfo(c)

	ctx := c.Request.Context()
	var sku model.PoolSKU
	err := h.store.DB.QueryRow(ctx, `
		SELECT id, name, term_price, term_months, active FROM pool_skus WHERE id = $1`, req.SKUID).
		Scan(&sku.ID, &sku.Name, &sku.TermPrice, &sku.TermMonths, &sku.Active)
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "sku not found"}})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}
	if !sku.Active {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "sku is not available for purchase"}})
		return
	}

	// Trial = 7 days, free
	totalAmount := sku.TermPrice
	if req.Trial {
		totalAmount = 0
	}
	now := time.Now()
	endAt := now.AddDate(0, sku.TermMonths, 0)
	if req.Trial {
		endAt = now.Add(7 * 24 * time.Hour)
	}

	// Mock payment: in P0 we mark paid immediately when not trial.
	paymentStatus := model.PaymentPending
	if req.Trial {
		paymentStatus = model.PaymentPaid // trial is "free"
	} else {
		paymentStatus = model.PaymentPaid // mock: pay succeeds instantly
	}

	tx, err := h.store.DB.Begin(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}
	defer tx.Rollback(ctx)

	var subID string
	err = tx.QueryRow(ctx, `
		INSERT INTO pool_subscriptions
			(user_id, sku_id, total_amount, currency, start_at, end_at, auto_renew,
			 status, payment_status, trial)
		VALUES ($1, $2, $3, 'CNY', $4, $5, $6, 'active', $7, $8)
		RETURNING id`,
		auth.UserID, req.SKUID, totalAmount, now, endAt, req.AutoRenew, paymentStatus, req.Trial,
	).Scan(&subID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}

	// Issue invoice for non-trial purchases
	if !req.Trial {
		_, err = tx.Exec(ctx, `
			INSERT INTO pool_invoices (subscription_id, period_start, period_end, amount, status, due_at, paid_at)
			VALUES ($1, $2, $3, $4, 'paid', $2, $2)`,
			subID, now, endAt, totalAmount)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
			return
		}
	}

	if err := tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}

	// Return the newly created subscription
	sub, err := h.fetchSubscription(ctx, subID, auth.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}
	c.JSON(http.StatusCreated, sub)
}

// Cancel POST /v0/pools/subscriptions/:id/cancel
func (h *PoolSubscriptionsHandler) Cancel(c *gin.Context) {
	id := c.Param("id")
	var req model.CancelSubscriptionRequest
	_ = c.ShouldBindJSON(&req)
	auth := middleware.GetAuthInfo(c)
	ctx := c.Request.Context()

	var currentStatus string
	err := h.store.DB.QueryRow(ctx,
		`SELECT status FROM pool_subscriptions WHERE id = $1 AND user_id = $2`,
		id, auth.UserID).Scan(&currentStatus)
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "subscription not found"}})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}
	if currentStatus == model.SubStatusCancelled || currentStatus == model.SubStatusExpired {
		c.JSON(http.StatusConflict, gin.H{"error": gin.H{"message": "subscription is already " + currentStatus}})
		return
	}

	newStatus := model.SubStatusCancelled
	if !req.Immediate {
		// Schedule cancel at period end — leave status as active, system job flips it later.
		// For P0 we set a flag via auto_renew=false and keep status active until end_at.
		_, err = h.store.DB.Exec(ctx,
			`UPDATE pool_subscriptions SET auto_renew = FALSE, updated_at = NOW() WHERE id = $1`, id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "cancel scheduled at period end", "id": id})
		return
	}

	_, err = h.store.DB.Exec(ctx,
		`UPDATE pool_subscriptions SET status = $1, updated_at = NOW() WHERE id = $2`,
		newStatus, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "subscription cancelled", "id": id, "status": newStatus})
}

// Renew POST /v0/pools/subscriptions/:id/renew — create a new subscription chained to the old one.
func (h *PoolSubscriptionsHandler) Renew(c *gin.Context) {
	id := c.Param("id")
	auth := middleware.GetAuthInfo(c)
	ctx := c.Request.Context()

	// Load the old subscription
	var skuID string
	var oldEndAt time.Time
	var oldStatus string
	err := h.store.DB.QueryRow(ctx,
		`SELECT sku_id, end_at, status FROM pool_subscriptions WHERE id = $1 AND user_id = $2`,
		id, auth.UserID).Scan(&skuID, &oldEndAt, &oldStatus)
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "subscription not found"}})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}

	// Load the SKU
	var termPrice float64
	var termMonths int
	err = h.store.DB.QueryRow(ctx,
		`SELECT term_price, term_months FROM pool_skus WHERE id = $1 AND active = TRUE`, skuID).
		Scan(&termPrice, &termMonths)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "sku not available"}})
		return
	}

	now := time.Now()
	// New period starts where the old one ended (or now if already expired)
	startAt := now
	if oldStatus == model.SubStatusActive && oldEndAt.After(now) {
		startAt = oldEndAt
	}
	endAt := startAt.AddDate(0, termMonths, 0)

	tx, err := h.store.DB.Begin(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}
	defer tx.Rollback(ctx)

	var newID string
	err = tx.QueryRow(ctx, `
		INSERT INTO pool_subscriptions
			(user_id, sku_id, total_amount, currency, start_at, end_at, auto_renew,
			 status, payment_status, renewed_from_id)
		VALUES ($1, $2, $3, 'CNY', $4, $5, TRUE, 'active', 'paid', $6)
		RETURNING id`,
		auth.UserID, skuID, termPrice, startAt, endAt, id,
	).Scan(&newID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO pool_invoices (subscription_id, period_start, period_end, amount, status, due_at, paid_at)
		VALUES ($1, $2, $3, $4, 'paid', $2, $2)`, newID, startAt, endAt, termPrice)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}
	if err := tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}

	sub, err := h.fetchSubscription(ctx, newID, auth.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}
	c.JSON(http.StatusCreated, sub)
}

// ListInvoices GET /v0/pools/subscriptions/:id/invoices
func (h *PoolSubscriptionsHandler) ListInvoices(c *gin.Context) {
	id := c.Param("id")
	auth := middleware.GetAuthInfo(c)
	rows, err := h.store.DB.Query(c.Request.Context(), `
		SELECT i.id, i.subscription_id, i.period_start, i.period_end, i.amount, i.status, i.due_at, i.paid_at, i.created_at
		FROM pool_invoices i
		JOIN pool_subscriptions s ON s.id = i.subscription_id
		WHERE i.subscription_id = $1 AND s.user_id = $2
		ORDER BY i.period_start DESC`, id, auth.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}
	defer rows.Close()

	var data []model.PoolInvoice
	for rows.Next() {
		var inv model.PoolInvoice
		if err := rows.Scan(&inv.ID, &inv.SubscriptionID, &inv.PeriodStart, &inv.PeriodEnd,
			&inv.Amount, &inv.Status, &inv.DueAt, &inv.PaidAt, &inv.CreatedAt); err != nil {
			continue
		}
		data = append(data, inv)
	}
	c.JSON(http.StatusOK, model.ListPoolInvoicesResponse{Data: data})
}

func (h *PoolSubscriptionsHandler) fetchSubscription(ctx context.Context, id, userID string) (*model.PoolSubscription, error) {
	var s model.PoolSubscription
	err := h.store.DB.QueryRow(ctx, `
		SELECT s.id, s.user_id, s.sku_id, sk.name, sk.gpu_type, sk.gpu_count,
			s.pool_id, s.total_amount, s.currency, s.start_at, s.end_at,
			s.auto_renew, s.status, s.payment_status, s.renewed_from_id, s.trial,
			s.created_at, s.updated_at
		FROM pool_subscriptions s
		JOIN pool_skus sk ON sk.id = s.sku_id
		WHERE s.id = $1 AND s.user_id = $2`, id, userID).
		Scan(&s.ID, &s.UserID, &s.SKUID, &s.SKUName, &s.GPUType, &s.GPUCount,
			&s.PoolID, &s.TotalAmount, &s.Currency, &s.StartAt, &s.EndAt,
			&s.AutoRenew, &s.Status, &s.PaymentStatus, &s.RenewedFromID, &s.Trial,
			&s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}
