// Package datavalidator provides data quality checks for fine-tuning datasets.
package datavalidator

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ValidationResult holds the result of data validation.
type ValidationResult struct {
	IsValid       bool                `json:"is_valid"`
	TotalRows     int                 `json:"total_rows"`
	ValidRows     int                 `json:"valid_rows"`
	InvalidRows   int                 `json:"invalid_rows"`
	Errors        []ValidationError   `json:"errors,omitempty"`
	Warnings      []ValidationWarning `json:"warnings,omitempty"`
	QualityScore  float64             `json:"quality_score"` // 0-100
	Report        string              `json:"report"`
}

// ValidationError represents a row-level validation error.
type ValidationError struct {
	Row     int    `json:"row"`
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationWarning represents a non-fatal warning.
type ValidationWarning struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// DataFormat defines the expected format of the dataset.
type DataFormat string

const (
	FormatChatML     DataFormat = "chatml"     // {"messages": [{"role": "...", "content": "..."}]}
	FormatCompletion DataFormat = "completion" // {"prompt": "...", "completion": "..."}
	FormatInstruction DataFormat = "instruction" // {"instruction": "...", "input": "...", "output": "..."}
)

// ValidateDataset validates a fine-tuning dataset for quality issues.
func ValidateDataset(data []byte, format DataFormat) (*ValidationResult, error) {
	result := &ValidationResult{
		IsValid: true,
	}

	// Parse lines
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	result.TotalRows = len(lines)

	var roleCounts map[string]int
	var contentLengths []int

	switch format {
	case FormatChatML:
		_, roleCounts, contentLengths = validateChatML(lines, result)
	case FormatCompletion:
		validateCompletion(lines, result)
	case FormatInstruction:
		validateInstruction(lines, result)
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}

	// Check for data imbalance
	if roleCounts != nil {
		checkRoleImbalance(roleCounts, result)
	}

	// Check content length distribution
	if len(contentLengths) > 0 {
		checkContentLength(contentLengths, result)
	}

	// Calculate quality score
	result.QualityScore = calculateQualityScore(result)
	result.IsValid = len(result.Errors) == 0

	// Generate report
	result.Report = generateReport(result)

	return result, nil
}

func validateChatML(lines []string, result *ValidationResult) ([]map[string]interface{}, map[string]int, []int) {
	roleCounts := make(map[string]int)
	var contentLengths []int

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var row map[string]interface{}
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			result.Errors = append(result.Errors, ValidationError{
				Row:     i + 1,
				Field:   "json",
				Message: fmt.Sprintf("Invalid JSON: %v", err),
			})
			result.InvalidRows++
			continue
		}

		// Check for messages field
		messagesRaw, ok := row["messages"]
		if !ok {
			result.Errors = append(result.Errors, ValidationError{
				Row:     i + 1,
				Field:   "messages",
				Message: "Missing 'messages' field",
			})
			result.InvalidRows++
			continue
		}

		messages, ok := messagesRaw.([]interface{})
		if !ok {
			result.Errors = append(result.Errors, ValidationError{
				Row:     i + 1,
				Field:   "messages",
				Message: "'messages' field must be an array",
			})
			result.InvalidRows++
			continue
		}

		result.ValidRows++

		for _, msgRaw := range messages {
			msg, ok := msgRaw.(map[string]interface{})
			if !ok {
				continue
			}

			role, _ := msg["role"].(string)
			content, _ := msg["content"].(string)

			roleCounts[role]++
			contentLengths = append(contentLengths, len(content))

			// Validate required fields
			if role == "" {
				result.Errors = append(result.Errors, ValidationError{
					Row:     i + 1,
					Field:   "role",
					Message: "Empty role in message",
				})
			}
			if content == "" {
				result.Warnings = append(result.Warnings, ValidationWarning{
					Type:    "empty_content",
					Message: fmt.Sprintf("Row %d: Empty content in message", i+1),
				})
			}
		}
	}

	return nil, roleCounts, contentLengths
}

func validateCompletion(lines []string, result *ValidationResult) {
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var row map[string]interface{}
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			result.Errors = append(result.Errors, ValidationError{
				Row:     i + 1,
				Field:   "json",
				Message: fmt.Sprintf("Invalid JSON: %v", err),
			})
			result.InvalidRows++
			continue
		}

		if _, ok := row["prompt"]; !ok {
			result.Errors = append(result.Errors, ValidationError{
				Row:     i + 1,
				Field:   "prompt",
				Message: "Missing 'prompt' field",
			})
			result.InvalidRows++
			continue
		}
		if _, ok := row["completion"]; !ok {
			result.Errors = append(result.Errors, ValidationError{
				Row:     i + 1,
				Field:   "completion",
				Message: "Missing 'completion' field",
			})
			result.InvalidRows++
			continue
		}

		result.ValidRows++
	}
}

func validateInstruction(lines []string, result *ValidationResult) {
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var row map[string]interface{}
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			result.Errors = append(result.Errors, ValidationError{
				Row:     i + 1,
				Field:   "json",
				Message: fmt.Sprintf("Invalid JSON: %v", err),
			})
			result.InvalidRows++
			continue
		}

		if _, ok := row["instruction"]; !ok {
			result.Errors = append(result.Errors, ValidationError{
				Row:     i + 1,
				Field:   "instruction",
				Message: "Missing 'instruction' field",
			})
			result.InvalidRows++
			continue
		}
		if _, ok := row["output"]; !ok {
			result.Errors = append(result.Errors, ValidationError{
				Row:     i + 1,
				Field:   "output",
				Message: "Missing 'output' field",
			})
			result.InvalidRows++
			continue
		}

		result.ValidRows++
	}
}

func checkRoleImbalance(roleCounts map[string]int, result *ValidationResult) {
	total := 0
	for _, count := range roleCounts {
		total += count
	}

	if total == 0 {
		return
	}

	// Check if assistant messages are less than 10% of total
	if assistantCount, ok := roleCounts["assistant"]; ok {
		assistantRatio := float64(assistantCount) / float64(total)
		if assistantRatio < 0.1 {
			result.Warnings = append(result.Warnings, ValidationWarning{
				Type:    "role_imbalance",
				Message: fmt.Sprintf("Assistant messages are only %.1f%% of total messages. Consider adding more assistant responses.", assistantRatio*100),
			})
		}
	}

	// Check if system messages dominate
	if systemCount, ok := roleCounts["system"]; ok {
		systemRatio := float64(systemCount) / float64(total)
		if systemRatio > 0.5 {
			result.Warnings = append(result.Warnings, ValidationWarning{
				Type:    "role_imbalance",
				Message: fmt.Sprintf("System messages are %.1f%% of total messages. Too many system prompts may reduce training effectiveness.", systemRatio*100),
			})
		}
	}
}

func checkContentLength(lengths []int, result *ValidationResult) {
	if len(lengths) == 0 {
		return
	}

	// Check for very short content
	shortCount := 0
	for _, l := range lengths {
		if l < 10 {
			shortCount++
		}
	}

	if float64(shortCount)/float64(len(lengths)) > 0.2 {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Type:    "short_content",
			Message: fmt.Sprintf("%.0f%% of messages have less than 10 characters. Consider filtering out very short examples.", float64(shortCount)/float64(len(lengths))*100),
		})
	}

	// Check for very long content
	longCount := 0
	for _, l := range lengths {
		if l > 4000 {
			longCount++
		}
	}

	if float64(longCount)/float64(len(lengths)) > 0.1 {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Type:    "long_content",
			Message: fmt.Sprintf("%.0f%% of messages exceed 4000 characters. Very long examples may cause context overflow during training.", float64(longCount)/float64(len(lengths))*100),
		})
	}
}

func calculateQualityScore(result *ValidationResult) float64 {
	if result.TotalRows == 0 {
		return 0
	}

	// Start with 100 and deduct for issues
	score := 100.0

	// Deduct for errors (5 points per error, max 50)
	errorDeduction := float64(len(result.Errors)) * 5
	if errorDeduction > 50 {
		errorDeduction = 50
	}
	score -= errorDeduction

	// Deduct for warnings (2 points per warning, max 30)
	warningDeduction := float64(len(result.Warnings)) * 2
	if warningDeduction > 30 {
		warningDeduction = 30
	}
	score -= warningDeduction

	// Deduct for invalid rows ratio
	if result.TotalRows > 0 {
		invalidRatio := float64(result.InvalidRows) / float64(result.TotalRows)
		score -= invalidRatio * 20
	}

	if score < 0 {
		score = 0
	}

	return score
}

func generateReport(result *ValidationResult) string {
	var sb strings.Builder

	sb.WriteString("=== Data Quality Report ===\n\n")
	sb.WriteString(fmt.Sprintf("Total rows: %d\n", result.TotalRows))
	sb.WriteString(fmt.Sprintf("Valid rows: %d\n", result.ValidRows))
	sb.WriteString(fmt.Sprintf("Invalid rows: %d\n", result.InvalidRows))
	sb.WriteString(fmt.Sprintf("Quality score: %.0f/100\n\n", result.QualityScore))

	if len(result.Errors) > 0 {
		sb.WriteString(fmt.Sprintf("Errors (%d):\n", len(result.Errors)))
		for _, err := range result.Errors {
			sb.WriteString(fmt.Sprintf("  - Row %d, %s: %s\n", err.Row, err.Field, err.Message))
		}
		sb.WriteString("\n")
	}

	if len(result.Warnings) > 0 {
		sb.WriteString(fmt.Sprintf("Warnings (%d):\n", len(result.Warnings)))
		for _, warn := range result.Warnings {
			sb.WriteString(fmt.Sprintf("  - [%s] %s\n", warn.Type, warn.Message))
		}
		sb.WriteString("\n")
	}

	if result.QualityScore >= 90 {
		sb.WriteString("✅ Data quality is excellent. Ready for training.")
	} else if result.QualityScore >= 70 {
		sb.WriteString("⚠️ Data quality is acceptable but could be improved.")
	} else {
		sb.WriteString("❌ Data quality is poor. Please fix issues before training.")
	}

	return sb.String()
}
