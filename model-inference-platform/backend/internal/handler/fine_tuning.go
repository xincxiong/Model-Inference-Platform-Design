package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/xincxiong/model-inference-platform/backend/internal/config"
	"github.com/xincxiong/model-inference-platform/backend/internal/middleware"
	"github.com/xincxiong/model-inference-platform/backend/internal/model"
	"github.com/xincxiong/model-inference-platform/backend/internal/store"
	"github.com/xincxiong/model-inference-platform/backend/internal/volcano"
	"go.uber.org/zap"
	batchv1alpha1 "volcano.sh/apis/pkg/apis/batch/v1alpha1"
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
	store         *store.Store
	volcanoClient *volcano.Client
	volcanoCfg    config.VolcanoConfig
	logger        *zap.Logger
}

func NewFineTuningHandler(s *store.Store, vc *volcano.Client, volcanoCfg config.VolcanoConfig, logger *zap.Logger) *FineTuningHandler {
	return &FineTuningHandler{store: s, volcanoClient: vc, volcanoCfg: volcanoCfg, logger: logger}
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

	if isRLMethod(method) && rolloutScenario == "" {
		rolloutScenario = model.RolloutChat
	}

	hp := req.Hyperparameters
	if hp == nil {
		hp = map[string]interface{}{}
	}
	rolloutCfg := req.RolloutConfig
	if rolloutCfg == nil {
		rolloutCfg = map[string]interface{}{}
	}
	// 调度元数据落盘，便于观察与审计
	rolloutCfg["volcano_queue"] = h.volcanoCfg.Queue
	rolloutCfg["volcano_namespace"] = h.volcanoCfg.Namespace
	if h.volcanoCfg.PriorityClass != "" {
		rolloutCfg["volcano_priority_class"] = h.volcanoCfg.PriorityClass
	}
	if h.volcanoCfg.SchedulerPolicy != "" {
		rolloutCfg["volcano_scheduler_policy"] = h.volcanoCfg.SchedulerPolicy
	}
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

	// ── Validate pool resources (P1) ──────────────────────────────────────
	if err := h.validatePoolResources(ctx, auth.UserID, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": err.Error(), "type": "invalid_request_error"}})
		return
	}

	_, err := h.store.DB.Exec(ctx,
		`INSERT INTO fine_tuning_jobs
		 (id, user_id, base_model, training_file, method, hyperparameters, rollout_scenario, rollout_config, reward_config,
		  pool_id, gpu_request, rollout_pool_id, status)
		 VALUES ($1,$2,$3,$4,$5,$6::jsonb,$7,$8::jsonb,$9::jsonb,$10,$11,$12,'queued')`,
		jobID, auth.UserID, req.Model, req.TrainingFile, method,
		hpBytes, rolloutScenario, rolloutBytes, rewardBytes,
		req.PoolID, req.GPURequest, nilIfEmpty(req.RolloutPoolID),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error(), "type": "api_error"}})
		return
	}

	// Submit VolcanoJob if Volcano is enabled, otherwise simulate
	if h.volcanoClient != nil && h.volcanoClient.IsEnabled() {
		go h.submitVolcanoJob(jobID, req.Model, method, rolloutScenario, rolloutCfg)
	} else {
		go h.simulateJobFinish(jobID, req.Model)
	}

	row := h.store.DB.QueryRow(ctx, selectJobSQL+` WHERE id = $1`, jobID)
	j, err := h.scanJobRow(row)
	if err != nil {
		c.JSON(http.StatusCreated, gin.H{"id": jobID, "object": "fine_tuning.job", "status": "queued"})
		return
	}
	c.JSON(http.StatusCreated, jobToAPI(j))
}

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

func (h *FineTuningHandler) submitVolcanoJob(jobID, baseModel, method, rolloutScenario string, rolloutCfg map[string]interface{}) {
	ctx := context.Background()
	gpuCount := int32(4)
	if method == "full" {
		gpuCount = 8
	}

	namespace := h.volcanoCfg.Namespace
	if namespace == "" {
		namespace = "inference-platform"
	}
	if ns := h.volcanoClient.Namespace(); ns != "" {
		namespace = ns
	}

	queueName := h.volcanoCfg.Queue
	if queueName == "" {
		queueName = "inference-platform-queue"
	}

	imageName := h.volcanoCfg.JobImage
	if imageName == "" {
		imageName = "pytorch/pytorch:2.1.0-cuda12.1-cudnn8-runtime"
	}

	minAvailable := gpuCount
	if h.volcanoCfg.MinAvailableOverride > 0 {
		minAvailable = int32(h.volcanoCfg.MinAvailableOverride)
	}

	schedulerPolicy := h.volcanoCfg.SchedulerPolicy
	if schedulerPolicy == "" {
		schedulerPolicy = "spread"
	}

	vJob := volcano.BuildFineTuningJob(
		jobID,
		namespace,
		queueName,
		imageName,
		gpuCount,
		40960,
		80,
		h.buildTrainingCommand(baseModel, method, rolloutScenario),
		volcano.JobOptions{
			MinAvailable:           minAvailable,
			TTLSecondsAfterFinish:  int32(h.volcanoCfg.TTLSecondsAfterFinish),
			PriorityClass:          h.volcanoCfg.PriorityClass,
			SchedulerPolicy:        schedulerPolicy,
		},
	)

	created, err := h.volcanoClient.CreateJob(ctx, vJob)
	if err != nil {
		h.logger.Error("failed to create volcano job", zap.String("job_id", jobID), zap.Error(err))
		_, _ = h.store.DB.Exec(ctx,
			`UPDATE fine_tuning_jobs SET status = 'failed', error_message = $2, updated_at = $3 WHERE id = $1`,
			jobID, err.Error(), time.Now().UTC())
		return
	}

	h.logger.Info("volcano job submitted",
		zap.String("job_id", jobID),
		zap.String("volcano_job_name", created.Name),
		zap.String("queue", queueName),
		zap.String("namespace", namespace),
		zap.Int32("gpu_count", gpuCount))

	_, _ = h.store.DB.Exec(ctx,
		`UPDATE fine_tuning_jobs SET status = 'running', updated_at = $2 WHERE id = $1 AND status = 'queued'`,
		jobID, time.Now().UTC())

	go h.monitorVolcanoJob(jobID)
}

func (h *FineTuningHandler) monitorVolcanoJob(jobID string) {
	ctx := context.Background()
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		phase, err := h.volcanoClient.GetJobStatus(ctx, jobID)
		if err != nil {
			h.logger.Warn("failed to get volcano job status", zap.String("job_id", jobID), zap.Error(err))
			continue
		}

		switch phase {
		case batchv1alpha1.Completed:
			_, _ = h.store.DB.Exec(ctx,
				`UPDATE fine_tuning_jobs SET status = 'succeeded', fine_tuned_model = $2, updated_at = $3 WHERE id = $1 AND status = 'running'`,
				jobID, jobID+"-completed", time.Now().UTC())
			return
		case batchv1alpha1.Failed:
			_, _ = h.store.DB.Exec(ctx,
				`UPDATE fine_tuning_jobs SET status = 'failed', error_message = 'volcano job failed', updated_at = $2 WHERE id = $1 AND status = 'running'`,
				jobID, time.Now().UTC())
			return
		case batchv1alpha1.Terminated, batchv1alpha1.Terminating:
			_, _ = h.store.DB.Exec(ctx,
				`UPDATE fine_tuning_jobs SET status = 'cancelled', updated_at = $2 WHERE id = $1 AND status = 'running'`,
				jobID, time.Now().UTC())
			return
		case batchv1alpha1.Running:
			h.logger.Debug("volcano job running", zap.String("job_id", jobID))
		default:
			h.logger.Debug("volcano job phase", zap.String("job_id", jobID), zap.String("phase", string(phase)))
		}
	}
}

func (h *FineTuningHandler) buildTrainingCommand(baseModel, method, rolloutScenario string) string {
	if method == "lora" || method == "qlora" {
		return "pip install transformers peft accelerate && python train_lora.py --model " + baseModel
	}
	if method == "full" {
		return "pip install transformers deepspeed && deepspeed train_full.py --model " + baseModel
	}
	return "pip install transformers && python train_rl.py --model " + baseModel + " --method " + method
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

	// Cancel VolcanoJob if enabled
	if h.volcanoClient != nil && h.volcanoClient.IsEnabled() {
		_ = h.volcanoClient.CancelJob(ctx, id)
	}

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
       pool_id, gpu_request, rollout_pool_id,
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
		&j.PoolID, &j.GPURequest, &j.RolloutPoolID,
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
		PoolID:          j.PoolID,
		GPURequest:      j.GPURequest,
		RolloutPoolID:   j.RolloutPoolID,
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

// ─── P1: pool resource validation ───────────────────────────────────────────

// validatePoolResources checks that the request's pool binding is feasible:
//   1. Pool exists, owned by user, status=active
//   2. Pool has enough free GPU capacity (sum of running+queued gpu_request + new ≤ pool.gpu_count)
//   3. Exclusive pool: can't have any other in-flight jobs
//   4. TP × PP × DP ≤ gpu_request
//   5. RolloutPoolID (if set) is valid, active, and (if exclusive) empty
func (h *FineTuningHandler) validatePoolResources(ctx context.Context, userID string, req *model.FineTuningJobCreateRequest) error {
	if req.PoolID == "" {
		return errors.New("pool_id is required")
	}
	if req.GPURequest < 1 {
		return errors.New("gpu_request must be ≥ 1")
	}

	if err := h.checkPoolCapacity(ctx, userID, req.PoolID, req.GPURequest, "training"); err != nil {
		return err
	}

	// Rollout pool: if empty, fall back to training pool
	rolloutID := req.RolloutPoolID
	if rolloutID == "" {
		rolloutID = req.PoolID
	}
	rolloutGPU := req.RolloutGPURequest
	if rolloutGPU == 0 {
		rolloutGPU = req.GPURequest
	}
	// Only check separate capacity if rollout pool differs from training pool
	if rolloutID != req.PoolID {
		if err := h.checkPoolCapacity(ctx, userID, rolloutID, rolloutGPU, "rollout"); err != nil {
			return err
		}
	}

	// Parallelism topology constraint
	if req.EngineConfig != nil {
		tp := req.EngineConfig.TensorModelParallelSize
		pp := req.EngineConfig.PipelineModelParallelSize
		dp := req.EngineConfig.DataParallelSize
		if tp == 0 { tp = 1 }
		if pp == 0 { pp = 1 }
		if dp == 0 { dp = 1 }
		required := tp * pp * dp
		if required > req.GPURequest {
			return fmt.Errorf("TP(%d) × PP(%d) × DP(%d) = %d must be ≤ gpu_request(%d)", tp, pp, dp, required, req.GPURequest)
		}
	}
	return nil
}

// checkPoolCapacity loads the pool and validates that it has room for the requested GPUs.
func (h *FineTuningHandler) checkPoolCapacity(ctx context.Context, userID, poolID string, gpuRequest int, role string) error {
	var (
		ownerID  string
		status   string
		gpuTotal int
		sharing  string
		endAt    *time.Time
	)
	err := h.store.DB.QueryRow(ctx, `
		SELECT user_id, status, gpu_count, sharing_mode, service_end_at
		FROM compute_pools WHERE id = $1`, poolID).
		Scan(&ownerID, &status, &gpuTotal, &sharing, &endAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("%s pool not found: %s", role, poolID)
	}
	if err != nil {
		return err
	}
	if ownerID != userID {
		return fmt.Errorf("%s pool not owned by user", role)
	}
	if status != model.PoolStatusActive {
		return fmt.Errorf("%s pool is %s, not active", role, status)
	}
	if endAt != nil && endAt.Before(time.Now()) {
		return fmt.Errorf("%s pool's subscription has ended", role)
	}

	// Sum current allocation from in-flight jobs on this pool
	var allocated int
	err = h.store.DB.QueryRow(ctx, `
		SELECT COALESCE(SUM(gpu_request), 0)
		FROM fine_tuning_jobs
		WHERE pool_id = $1 AND status IN ('queued','running','validating')`, poolID).Scan(&allocated)
	if err != nil {
		return err
	}

	// Exclusive pool: any in-flight job blocks new submissions
	if sharing == model.PoolSharingExclusive && allocated > 0 {
		return fmt.Errorf("%s pool is exclusive and currently busy (%d GPUs in use); try a shared pool or wait", role, allocated)
	}
	if allocated+gpuRequest > gpuTotal {
		free := gpuTotal - allocated
		if free < 0 {
			free = 0
		}
		return fmt.Errorf("%s pool insufficient capacity: requested %d, free %d (of %d); job will be queued when Volcano scheduler is connected", role, gpuRequest, free, gpuTotal)
	}
	return nil
}

// nilIfEmpty returns nil for an empty string so the SQL column stays NULL.
func nilIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
