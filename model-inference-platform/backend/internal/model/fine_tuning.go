package model

import "time"

// FineTuningJobCreateRequest matches OpenAI-style POST /v1/fine_tuning/jobs (subset).
type FineTuningJobCreateRequest struct {
	Model            string                 `json:"model" binding:"required"`
	TrainingFile     string                 `json:"training_file"`
	ValidationFile   string                 `json:"validation_file"`
	Method           string                 `json:"method"` // lora, full
	Hyperparameters  map[string]interface{} `json:"hyperparameters"`
}

// FineTuningJob is returned by fine-tuning APIs.
type FineTuningJob struct {
	ID              string                 `json:"id"`
	Object          string                 `json:"object"`
	Model           string                 `json:"model"`
	TrainingFile    string                 `json:"training_file,omitempty"`
	Method          string                 `json:"method"`
	Hyperparameters map[string]interface{} `json:"hyperparameters"`
	Status          string                 `json:"status"`
	FineTunedModel  *string                `json:"fine_tuned_model,omitempty"`
	Error           *FineTuningJobError    `json:"error,omitempty"`
	CreatedAt       int64                  `json:"created_at"`
	UpdatedAt       int64                  `json:"updated_at"`
}

type FineTuningJobError struct {
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

// FineTuningJobRow is the DB row shape (internal).
type FineTuningJobRow struct {
	ID              string
	UserID          string
	BaseModel       string
	TrainingFile    string
	Method          string
	Hyperparameters map[string]interface{}
	Status          string
	FineTunedModel  *string
	ErrorMessage    *string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type FineTuningJobListResponse struct {
	Object string          `json:"object"`
	Data   []FineTuningJob `json:"data"`
}
