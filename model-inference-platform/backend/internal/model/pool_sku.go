package model

import "time"

// PoolSKU is a purchasable compute-pool plan (one row per GPU config × billing term).
type PoolSKU struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	GPUType         string    `json:"gpu_type"`
	GPUCount        int       `json:"gpu_count"`
	Region          string    `json:"region"`
	SharingMode     string    `json:"sharing_mode"` // "exclusive" | "shared-fifo"
	Term            string    `json:"term"`         // "monthly" | "quarterly" | "yearly"
	TermMonths      int       `json:"term_months"`
	HourlyListPrice float64   `json:"hourly_list_price"` // 标价折算每小时
	TermPrice       float64   `json:"term_price"`        // 订阅总价
	DiscountPct     float64   `json:"discount_pct"`      // 对比按量的折扣率 (0-100)
	SLAClass        string    `json:"sla_class"`          // "standard" | "enhanced"
	Active          bool      `json:"active"`
	SortOrder       int       `json:"sort_order"`
	CreatedAt       time.Time `json:"created_at"`
}

// ListPoolSKUsResponse is the response for GET /v0/pools/skus.
type ListPoolSKUsResponse struct {
	Data []PoolSKU `json:"data"`
}

// ListPoolSKUsQuery filters the SKU list.
type ListPoolSKUsQuery struct {
	GPUType     string `form:"gpu_type"`
	Region      string `form:"region"`
	SharingMode string `form:"sharing_mode"`
	Term        string `form:"term"`
	ActiveOnly  bool   `form:"active_only"`
}
