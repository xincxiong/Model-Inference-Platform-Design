// Package prompttemplate provides prompt template management with variable interpolation and versioning.
package prompttemplate

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// TemplateScope defines the visibility scope of a template.
type TemplateScope string

const (
	ScopeSystem TemplateScope = "system" // Available to all users
	ScopeTeam   TemplateScope = "team"   // Available to team members only
	ScopeUser   TemplateScope = "user"   // Private to the creator
)

// PromptTemplate represents a reusable prompt template.
type PromptTemplate struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Scope       TemplateScope `json:"scope"`
	OwnerID     string        `json:"owner_id"`
	Content     string        `json:"content"`
	Variables   []string      `json:"variables"` // extracted from content
	Version     int           `json:"version"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
	Tags        []string      `json:"tags"`
}

// Manager manages prompt templates.
type Manager struct {
	rdb *redis.Client
}

// New creates a new prompt template manager.
func New(rdb *redis.Client) *Manager {
	return &Manager{rdb: rdb}
}

// Create creates a new prompt template.
func (m *Manager) Create(ctx context.Context, template *PromptTemplate) error {
	if template.ID == "" {
		template.ID = "tmpl-" + strings.ReplaceAll(uuid.New().String(), "-", "")[:16]
	}
	template.Version = 1
	template.CreatedAt = time.Now()
	template.UpdatedAt = time.Now()

	// Extract variables from content ({{variable_name}} pattern)
	template.Variables = extractVariables(template.Content)

	// Store in Redis
	key := fmt.Sprintf("prompt_template:%s", template.ID)
	data := map[string]interface{}{
		"id":          template.ID,
		"name":        template.Name,
		"description": template.Description,
		"scope":       string(template.Scope),
		"owner_id":    template.OwnerID,
		"content":     template.Content,
		"variables":   strings.Join(template.Variables, ","),
		"version":     template.Version,
		"created_at":  template.CreatedAt.Unix(),
		"updated_at":  template.UpdatedAt.Unix(),
		"tags":        strings.Join(template.Tags, ","),
	}

	if err := m.rdb.HSet(ctx, key, data).Err(); err != nil {
		return err
	}

	// Add to index
	indexKey := fmt.Sprintf("prompt_template_index:%s", template.Scope)
	m.rdb.SAdd(ctx, indexKey, template.ID)

	// Add to owner index
	ownerKey := fmt.Sprintf("prompt_template_owner:%s", template.OwnerID)
	m.rdb.SAdd(ctx, ownerKey, template.ID)

	return nil
}

// Get retrieves a template by ID.
func (m *Manager) Get(ctx context.Context, id string) (*PromptTemplate, error) {
	key := fmt.Sprintf("prompt_template:%s", id)
	vals, err := m.rdb.HGetAll(ctx, key).Result()
	if err != nil || len(vals) == 0 {
		return nil, fmt.Errorf("template %q not found", id)
	}

	t := &PromptTemplate{
		ID:          vals["id"],
		Name:        vals["name"],
		Description: vals["description"],
		Scope:       TemplateScope(vals["scope"]),
		OwnerID:     vals["owner_id"],
		Content:     vals["content"],
	}

	if vals["variables"] != "" {
		t.Variables = strings.Split(vals["variables"], ",")
	}
	if vals["tags"] != "" {
		t.Tags = strings.Split(vals["tags"], ",")
	}

	fmt.Sscan(vals["version"], &t.Version)
	fmt.Sscan(vals["created_at"], &t.CreatedAt)
	fmt.Sscan(vals["updated_at"], &t.UpdatedAt)

	return t, nil
}

// Update updates an existing template (creates a new version).
func (m *Manager) Update(ctx context.Context, id string, content string) (*PromptTemplate, error) {
	t, err := m.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	t.Content = content
	t.Version++
	t.UpdatedAt = time.Now()
	t.Variables = extractVariables(content)

	// Update in Redis
	key := fmt.Sprintf("prompt_template:%s", id)
	m.rdb.HSet(ctx, key,
		"content", t.Content,
		"version", t.Version,
		"variables", strings.Join(t.Variables, ","),
		"updated_at", t.UpdatedAt.Unix(),
	)

	return t, nil
}

// Delete removes a template.
func (m *Manager) Delete(ctx context.Context, id string) error {
	t, err := m.Get(ctx, id)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("prompt_template:%s", id)
	if err := m.rdb.Del(ctx, key).Err(); err != nil {
		return err
	}

	// Remove from indexes
	m.rdb.SRem(ctx, fmt.Sprintf("prompt_template_index:%s", t.Scope), id)
	m.rdb.SRem(ctx, fmt.Sprintf("prompt_template_owner:%s", t.OwnerID), id)

	return nil
}

// List retrieves templates by scope.
func (m *Manager) List(ctx context.Context, scope TemplateScope) ([]*PromptTemplate, error) {
	indexKey := fmt.Sprintf("prompt_template_index:%s", scope)
	ids, err := m.rdb.SMembers(ctx, indexKey).Result()
	if err != nil {
		return nil, err
	}

	var templates []*PromptTemplate
	for _, id := range ids {
		t, err := m.Get(ctx, id)
		if err != nil {
			continue
		}
		templates = append(templates, t)
	}

	return templates, nil
}

// RenderContent interpolates variables in template content.
func RenderContent(content string, values map[string]string) string {
	variablePattern := regexp.MustCompile(`\{\{(\w+)\}\}`)
	
	result := variablePattern.ReplaceAllStringFunc(content, func(match string) string {
		varName := variablePattern.FindStringSubmatch(match)[1]
		if val, ok := values[varName]; ok {
			return val
		}
		return match
	})

	return result
}

// extractVariables extracts variable names from template content.
func extractVariables(content string) []string {
	variablePattern := regexp.MustCompile(`\{\{(\w+)\}\}`)
	matches := variablePattern.FindAllStringSubmatch(content, -1)
	
	seen := make(map[string]bool)
	var variables []string
	for _, match := range matches {
		if len(match) > 1 && !seen[match[1]] {
			seen[match[1]] = true
			variables = append(variables, match[1])
		}
	}
	
	return variables
}
