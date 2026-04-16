package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/xincxiong/model-inference-platform/backend/internal/middleware"
	"github.com/xincxiong/model-inference-platform/backend/internal/model"
	"github.com/xincxiong/model-inference-platform/backend/internal/store"
)

type MembersHandler struct {
	store *store.Store
}

func NewMembersHandler(s *store.Store) *MembersHandler {
	return &MembersHandler{store: s}
}

// List GET /api/members
func (h *MembersHandler) List(c *gin.Context) {
	auth := middleware.GetAuthInfo(c)
	rows, err := h.store.DB.Query(c.Request.Context(),
		`SELECT m.id, m.user_id, COALESCE(u.email, m.invite_email) as email,
		        COALESCE(u.name, '') as name, m.role, m.status, m.invited_by,
		        m.joined_at, m.created_at
		 FROM org_members m
		 LEFT JOIN users u ON u.id = m.user_id
		 WHERE m.org_owner_id = $1
		 ORDER BY m.created_at ASC`, auth.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}
	defer rows.Close()

	var data []model.OrgMember
	for rows.Next() {
		var m model.OrgMember
		if err := rows.Scan(
			&m.ID, &m.UserID, &m.Email, &m.Name, &m.Role,
			&m.Status, &m.InvitedBy, &m.JoinedAt, &m.CreatedAt,
		); err != nil {
			continue
		}
		data = append(data, m)
	}
	if data == nil {
		data = []model.OrgMember{}
	}
	c.JSON(http.StatusOK, model.OrgMemberListResponse{Object: "list", Data: data})
}

// Invite POST /api/members
func (h *MembersHandler) Invite(c *gin.Context) {
	var req model.InviteMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}

	if req.Role == "" {
		req.Role = model.MemberRoleMember
	}
	switch req.Role {
	case model.MemberRoleOwner, model.MemberRoleAdmin, model.MemberRoleMember, model.MemberRoleViewer:
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "role must be one of: owner, admin, member, viewer"}})
		return
	}

	auth := middleware.GetAuthInfo(c)
	ctx := c.Request.Context()

	// check for duplicate invite
	var exists bool
	_ = h.store.DB.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM org_members WHERE org_owner_id=$1 AND invite_email=$2)`,
		auth.UserID, req.Email).Scan(&exists)
	if exists {
		c.JSON(http.StatusConflict, gin.H{"error": gin.H{"message": "member already invited"}})
		return
	}

	// look up user by email (may not exist yet)
	var userID *string
	var userName string
	var uid string
	err := h.store.DB.QueryRow(ctx,
		`SELECT id, name FROM users WHERE email=$1`, req.Email).Scan(&uid, &userName)
	if err == nil {
		userID = &uid
	} else if !errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}

	var id string
	err = h.store.DB.QueryRow(ctx,
		`INSERT INTO org_members (org_owner_id, user_id, invite_email, role, status, invited_by)
		 VALUES ($1, $2, $3, $4, 'pending', $5) RETURNING id`,
		auth.UserID, userID, req.Email, req.Role, auth.UserID,
	).Scan(&id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}

	status := model.MemberStatusPending
	joinedAt := (*time.Time)(nil)
	if userID != nil {
		// user exists → auto-accept
		status = model.MemberStatusActive
		now := time.Now().UTC()
		joinedAt = &now
		_, _ = h.store.DB.Exec(ctx,
			`UPDATE org_members SET status='active', joined_at=$2 WHERE id=$1`, id, now)
	}

	c.JSON(http.StatusCreated, model.OrgMember{
		ID:        id,
		UserID:    func() string { if userID != nil { return *userID }; return "" }(),
		Email:     req.Email,
		Name:      userName,
		Role:      req.Role,
		Status:    status,
		InvitedBy: auth.UserID,
		JoinedAt:  joinedAt,
	})
}

// PatchRole PATCH /api/members/:id
func (h *MembersHandler) PatchRole(c *gin.Context) {
	id := c.Param("id")
	var req model.PatchMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}
	if req.Role == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "role is required"}})
		return
	}
	switch *req.Role {
	case model.MemberRoleOwner, model.MemberRoleAdmin, model.MemberRoleMember, model.MemberRoleViewer:
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "invalid role"}})
		return
	}

	auth := middleware.GetAuthInfo(c)
	cmd, err := h.store.DB.Exec(c.Request.Context(),
		`UPDATE org_members SET role=$2 WHERE id=$1 AND org_owner_id=$3`,
		id, *req.Role, auth.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}
	if cmd.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "member not found"}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "role": *req.Role})
}

// Remove DELETE /api/members/:id
func (h *MembersHandler) Remove(c *gin.Context) {
	id := c.Param("id")
	auth := middleware.GetAuthInfo(c)
	cmd, err := h.store.DB.Exec(c.Request.Context(),
		`DELETE FROM org_members WHERE id=$1 AND org_owner_id=$2`, id, auth.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}
	if cmd.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "member not found"}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": true})
}
