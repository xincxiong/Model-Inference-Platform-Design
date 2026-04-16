package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/xincxiong/model-inference-platform/backend/internal/middleware"
	"github.com/xincxiong/model-inference-platform/backend/internal/model"
	"github.com/xincxiong/model-inference-platform/backend/internal/store"
)

var validMethods = map[string]bool{
	"lora": true, "qlora": true, "full": true,
	"grpo": true, "gspo": true, "dapo": true, "vapo": true, "ppo": true, "dpo": true,
}

var validRolloutScenarios = map[string]bool{
	model.RolloutChat: true, model.RolloutCode: true,
	model.RolloutToolCall: true, model.RolloutAgentSingle: true,
	model.RolloutAgentMulti: true, model.RolloutMathReasoning: true,
	model.RolloutCustom: true,
	"": true, // optional
}

type FineTuningHandler struct {
	store *store.Store
}

func NewFineTuningHandler(s *store.Store) *FineTuningHandler {
	return &FineTuningHandler{store: s}
}

func isRLMethod(m string) bool {
	return m == "grpo" || m == "gspo" || m == "dapo" || m == "vapo" || m == "ppo" || m == "dpo"
}

func (h *FineTuningHandler) Create(c *gin.Context) {
	var req model.FineTuningJobCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": err.Error(), "type": "invalid_request_error"}})
		return
	}

	method := strings.ToLower(strings.TrimSpace(req.Method))
	if method == "" {
		method = "lora"
	}
	if !validMethods[method] {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"message": "method must be one of: lora, qlora, full, grpo, gspo, dapo, vapo, ppo, dpo",
			"type":    "invalid_request_error",
		}})
		return
	}

	rolloutScenario := req.RolloutScenario
	if !validRolloutScenarios[rolloutScenario] {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"message": "rollout_scenario must be one of: chat, code, tool_call, agent_single, agent_multi, math_reasoning, custom",
			"type":    "invalid_request_error",
		}})
		return
	}

	// Validate: RL methods require rollout_scenario
	if isRLMethod(method) && rolloutScenario == "" {
		rolloutScenario = model.RolloutChat // default
	}

	// Serialize JSON fields
	hp := req.Hyperparameters
	if hp == nil {
		hp = map[string]interface{}{}
	}
	rolloutCfg := req.RolloutConfig
	if rolloutCfg == nil {
		rolloutCfg = map[string]interface{}{}
	}
	// Merge slime engine_config and data_buffer_config into rollout_config for storage
	if req.EngineConfig != nil {
		b, _ := json.Marshal(req.EngineConfig)
		var ec map[string]interface{}
		_ = json.Unmarshal(b, &ec)
		rolloutCfg["engine_config"] = ec
	}
	if req.DataBufferConfig != nil {
		b, _ := json.Marshal(req.DataBufferConfig)
		var dbc map[string]interface{}
		_ = json.Unmarshal(b, &dbc)
		rolloutCfg["data_buffer_config"] = dbc
	}
	rewardCfg := map[string]interface{}{}
	if req.RewardConfig != nil {
		b, _ := json.Marshal(req.RewardConfig)
		_ = json.Unmarshal(b, &rewardCfg)
	}

	hpBytes, _ := json.Marshal(hp)
	rolloutBytes, _ := json.Marshal(rolloutCfg)
	rewardBytes, _ := json.Marshal(rewardCfg)

	auth := middleware.GetAuthInfo(c)
	jobID := "ftjob-" + strings.ReplaceAll(uuid.New().String(), "-", "")

	ctx := c.Request.Context()
	_, err := h.store.DB.Exec(ctx,
		`INSERT INTO fine_tuning_jobs
		 (id, user_id, base_model, training_file, method, hyperparameters, rollout_scenario, rollout_config, reward_config, status)
		 VALUES ($1,$2,$3,$4,$5,$6::jsonb,$7,$8::jsonb,$9::jsonb,'queued')`,
		jobID, auth.UserID, req.Model, req.TrainingFile, method,
		hpBytes, rolloutScenario, rolloutBytes, rewardBytes,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error(), "type": "api_error"}})
		return
	}

	go h.simulateJobFinish(jobID, req.Model)

	row := h.store.DB.QueryRow(ctx, selectJobSQL+` WHERE id = $1`, jobID)
	j, err := h.scanJobRow(row)
	if err != nil {
		c.JSON(http.StatusCreated, gin.H{"id": jobID, "object": "fine_tuning.job", "status": "queued"})
		return
	}
	c.JSON(http.StatusCreated, jobToAPI(j))
}

// simulateJobFinish: queued → running → succeeded (async, ~1.5s, MVP placeholder)
func (h *FineTuningHandler) simulateJobFinish(jobID, baseModel string) {
	ctx := context.Background()
	pool := h.store.DB
	tag, _ := pool.Exec(ctx,
		`UPDATE fine_tuning_jobs SET status = 'running', updated_at = $2 WHERE id = $1 AND status = 'queued'`,
		jobID, time.Now().UTC())
	if tag.RowsAffected() == 0 {
		return
	}
	time.Sleep(1500 * time.Millisecond)
	suffix := jobID
	if len(suffix) > 8 {
		suffix = suffix[len(suffix)-8:]
	}
	ftName := baseModel + "-ft-" + suffix
	_, _ = pool.Exec(ctx,
		`UPDATE fine_tuning_jobs SET status = 'succeeded', fine_tuned_model = $2, updated_at = $3
		 WHERE id = $1 AND status = 'running'`,
		jobID, ftName, time.Now().UTC())
}

func (h *FineTuningHandler) List(c *gin.Context) {
	auth := middleware.GetAuthInfo(c)
	rows, err := h.store.DB.Query(c.Request.Context(),
		selectJobSQL+` WHERE user_id = $1 ORDER BY created_at DESC LIMIT 100`, auth.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}
	defer rows.Close()

	var jobs []model.FineTuningJob
	for rows.Next() {
		j, err := h.scanJobRow(rows)
		if err != nil {
			continue
		}
		jobs = append(jobs, jobToAPI(j))
	}
	c.JSON(http.StatusOK, model.FineTuningJobListResponse{Object: "list", Data: jobs})
}

func (h *FineTuningHandler) Get(c *gin.Context) {
	id := c.Param("id")
	auth := middleware.GetAuthInfo(c)
	row := h.store.DB.QueryRow(c.Request.Context(),
		selectJobSQL+` WHERE id = $1 AND user_id = $2`, id, auth.UserID)
	j, err := h.scanJobRow(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "job not found", "type": "invalid_request_error"}})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}
	c.JSON(http.StatusOK, jobToAPI(j))
}

func (h *FineTuningHandler) Cancel(c *gin.Context) {
	id := c.Param("id")
	auth := middleware.GetAuthInfo(c)
	ctx := c.Request.Context()
	res, err := h.store.DB.Exec(ctx,
		`UPDATE fine_tuning_jobs SET status = 'cancelled', updated_at = $2
		 WHERE id = $1 AND user_id = $3 AND status IN ('queued','running')`,
		id, time.Now().UTC(), auth.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}
	if res.RowsAffected() == 0 {
		row := h.store.DB.QueryRow(ctx,
			`SELECT id FROM fine_tuning_jobs WHERE id = $1 AND user_id = $2`, id, auth.UserID)
		var dummy string
		if err := row.Scan(&dummy); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "job not found"}})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "job cannot be cancelled in current status"}})
		return
	}
	h.Get(c)
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

const selectJobSQL = `
SELECT id, user_id, base_model, training_file, method, hyperparameters,
       rollout_scenario, rollout_config, reward_config,
       status, fine_tuned_model, error_message, created_at, updated_at
FROM fine_tuning_jobs`

func (h *FineTuningHandler) scanJobRow(rows interface {
	Scan(dest ...any) error
}) (model.FineTuningJobRow, error) {
	var j model.FineTuningJobRow
	var hpJSON, rolloutCfgJSON, rewardCfgJSON []byte
	err := rows.Scan(
		&j.ID, &j.UserID, &j.BaseModel, &j.TrainingFile, &j.Method,
		&hpJSON, &j.RolloutScenario, &rolloutCfgJSON, &rewardCfgJSON,
		&j.Status, &j.FineTunedModel, &j.ErrorMessage, &j.CreatedAt, &j.UpdatedAt,
	)
	if err != nil {
		return j, err
	}
	_ = json.Unmarshal(hpJSON, &j.Hyperparameters)
	if j.Hyperparameters == nil {
		j.Hyperparameters = map[string]interface{}{}
	}
	_ = json.Unmarshal(rolloutCfgJSON, &j.RolloutConfig)
	if j.RolloutConfig == nil {
		j.RolloutConfig = map[string]interface{}{}
	}
	_ = json.Unmarshal(rewardCfgJSON, &j.RewardConfig)
	if j.RewardConfig == nil {
		j.RewardConfig = map[string]interface{}{}
	}
	return j, nil
}

func jobToAPI(j model.FineTuningJobRow) model.FineTuningJob {
	out := model.FineTuningJob{
		ID:              j.ID,
		Object:          "fine_tuning.job",
		Model:           j.BaseModel,
		TrainingFile:    j.TrainingFile,
		Method:          j.Method,
		Hyperparameters: j.Hyperparameters,
		RolloutScenario: j.RolloutScenario,
		RolloutConfig:   j.RolloutConfig,
		RewardConfig:    j.RewardConfig,
		Status:          j.Status,
		FineTunedModel:  j.FineTunedModel,
		CreatedAt:       j.CreatedAt.Unix(),
		UpdatedAt:       j.UpdatedAt.Unix(),
	}
	if j.ErrorMessage != nil && *j.ErrorMessage != "" {
		out.Error = &model.FineTuningJobError{Message: *j.ErrorMessage}
	}
	return out
}
