// Package errors provides standardized error codes compatible with OpenAI API spec.
package errors

import (
	"fmt"
	"net/http"
)

// ErrorCode defines standardized error types.
type ErrorCode string

const (
	// Client errors (4xx)
	ErrInvalidRequest     ErrorCode = "invalid_request_error"
	ErrAuthentication     ErrorCode = "authentication_error"
	ErrPermission         ErrorCode = "permission_error"
	ErrNotFound           ErrorCode = "not_found_error"
	ErrRateLimit          ErrorCode = "rate_limit_error"
	ErrConflict           ErrorCode = "conflict_error"
	
	// Server errors (5xx)
	ErrInternal           ErrorCode = "api_error"
	ErrServiceUnavailable ErrorCode = "service_unavailable_error"
	ErrTimeout            ErrorCode = "timeout_error"
	ErrCircuitBreakerOpen ErrorCode = "circuit_breaker_open"
	ErrNoWorkerAvailable  ErrorCode = "no_worker_available"
)

// APIError represents a standardized API error response.
type APIError struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
	Type    string    `json:"type"`
	Param   *string   `json:"param,omitempty"`
}

// ErrorResponse wraps APIError for OpenAI-compatible responses.
type ErrorResponse struct {
	Error APIError `json:"error"`
}

// NewAPIError creates a new APIError.
func NewAPIError(code ErrorCode, message string) APIError {
	return APIError{
		Code:    code,
		Message: message,
		Type:    string(code),
	}
}

// NewWithParam creates an APIError with a parameter reference.
func NewWithParam(code ErrorCode, message string, param string) APIError {
	return APIError{
		Code:    code,
		Message: message,
		Type:    string(code),
		Param:   &param,
	}
}

// ToResponse wraps the error in ErrorResponse format.
func (e APIError) ToResponse() ErrorResponse {
	return ErrorResponse{Error: e}
}

// HTTPStatus returns the appropriate HTTP status code for the error.
func (e APIError) HTTPStatus() int {
	switch e.Code {
	case ErrInvalidRequest:
		return http.StatusBadRequest
	case ErrAuthentication:
		return http.StatusUnauthorized
	case ErrPermission:
		return http.StatusForbidden
	case ErrNotFound:
		return http.StatusNotFound
	case ErrRateLimit:
		return http.StatusTooManyRequests
	case ErrConflict:
		return http.StatusConflict
	case ErrServiceUnavailable:
		return http.StatusServiceUnavailable
	case ErrTimeout:
		return http.StatusGatewayTimeout
	case ErrCircuitBreakerOpen, ErrNoWorkerAvailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

// Error implements the error interface.
func (e APIError) Error() string {
	if e.Param != nil {
		return fmt.Sprintf("%s: %s (param: %s)", e.Code, e.Message, *e.Param)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Helper functions for common errors

func ErrInvalidRequestf(msg string, args ...interface{}) APIError {
	return NewAPIError(ErrInvalidRequest, fmt.Sprintf(msg, args...))
}

func ErrAuthenticationf(msg string, args ...interface{}) APIError {
	return NewAPIError(ErrAuthentication, fmt.Sprintf(msg, args...))
}

func ErrNotFoundf(msg string, args ...interface{}) APIError {
	return NewAPIError(ErrNotFound, fmt.Sprintf(msg, args...))
}

func ErrRateLimitf(msg string, args ...interface{}) APIError {
	return NewAPIError(ErrRateLimit, fmt.Sprintf(msg, args...))
}

func ErrInternalf(msg string, args ...interface{}) APIError {
	return NewAPIError(ErrInternal, fmt.Sprintf(msg, args...))
}

func ErrTimeoutf(msg string, args ...interface{}) APIError {
	return NewAPIError(ErrTimeout, fmt.Sprintf(msg, args...))
}
