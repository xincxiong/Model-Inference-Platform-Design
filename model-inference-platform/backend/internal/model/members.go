package model

import "time"

// MemberRole constants
const (
	MemberRoleOwner  = "owner"
	MemberRoleAdmin  = "admin"
	MemberRoleMember = "member"
	MemberRoleViewer = "viewer"
)

// MemberStatus constants
const (
	MemberStatusActive  = "active"
	MemberStatusPending = "pending" // 邀请已发出，未接受
)

// ─── Request / Response ────────────────────────────────────────────────────

type InviteMemberRequest struct {
	Email string `json:"email" binding:"required"`
	Role  string `json:"role"` // owner | admin | member | viewer，默认 member
}

type PatchMemberRequest struct {
	Role *string `json:"role"`
}

// ─── API Shape ──────────────────────────────────────────────────────────────

type OrgMember struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Role      string    `json:"role"`
	Status    string    `json:"status"`
	InvitedBy string    `json:"invited_by"`
	JoinedAt  *time.Time `json:"joined_at,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type OrgMemberListResponse struct {
	Object string      `json:"object"`
	Data   []OrgMember `json:"data"`
}
