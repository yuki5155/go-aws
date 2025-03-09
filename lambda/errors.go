package lambda

import (
	"fmt"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
)

// ErrorType represents the type of error
type ErrorType string

const (
	// ErrorTypeInvalidRequest represents an invalid request error
	ErrorTypeInvalidRequest ErrorType = "InvalidRequest"
	// ErrorTypeUnauthorized represents an unauthorized error
	ErrorTypeUnauthorized ErrorType = "Unauthorized"
	// ErrorTypeForbidden represents a forbidden error
	ErrorTypeForbidden ErrorType = "Forbidden"
	// ErrorTypeNotFound represents a not found error
	ErrorTypeNotFound ErrorType = "NotFound"
	// ErrorTypeMethodNotAllowed represents a method not allowed error
	ErrorTypeMethodNotAllowed ErrorType = "MethodNotAllowed"
	// ErrorTypeConflict represents a conflict error
	ErrorTypeConflict ErrorType = "Conflict"
	// ErrorTypeInternalServer represents an internal server error
	ErrorTypeInternalServer ErrorType = "InternalServerError"
	// ErrorTypeServiceUnavailable represents a service unavailable error
	ErrorTypeServiceUnavailable ErrorType = "ServiceUnavailable"
	// ErrorTypeValidationFailed represents a validation failed error
	ErrorTypeValidationFailed ErrorType = "ValidationFailed"
	// ErrorTypeRequestFailed represents a request failed error (typically for downstream requests)
	ErrorTypeRequestFailed ErrorType = "RequestFailed"
)

// LambdaError represents a custom error for Lambda functions
type LambdaError struct {
	Type       ErrorType
	Message    string
	StatusCode int
	Err        error
}

// Error implements the error interface
func (e *LambdaError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Type, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// Unwrap returns the wrapped error
func (e *LambdaError) Unwrap() error {
	return e.Err
}

// ToAPIGatewayResponse converts a LambdaError to an APIGatewayProxyResponse
func (e *LambdaError) ToAPIGatewayResponse() events.APIGatewayProxyResponse {
	return events.APIGatewayProxyResponse{
		StatusCode: e.StatusCode,
		Body:       e.Message,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}
}

// NewInvalidRequestError creates a new invalid request error
func NewInvalidRequestError(msg string, err error) *LambdaError {
	return &LambdaError{
		Type:       ErrorTypeInvalidRequest,
		Message:    msg,
		StatusCode: http.StatusBadRequest,
		Err:        err,
	}
}

// NewUnauthorizedError creates a new unauthorized error
func NewUnauthorizedError(msg string, err error) *LambdaError {
	return &LambdaError{
		Type:       ErrorTypeUnauthorized,
		Message:    msg,
		StatusCode: http.StatusUnauthorized,
		Err:        err,
	}
}

// NewForbiddenError creates a new forbidden error
func NewForbiddenError(msg string, err error) *LambdaError {
	return &LambdaError{
		Type:       ErrorTypeForbidden,
		Message:    msg,
		StatusCode: http.StatusForbidden,
		Err:        err,
	}
}

// NewNotFoundError creates a new not found error
func NewNotFoundError(msg string, err error) *LambdaError {
	return &LambdaError{
		Type:       ErrorTypeNotFound,
		Message:    msg,
		StatusCode: http.StatusNotFound,
		Err:        err,
	}
}

// NewMethodNotAllowedError creates a new method not allowed error
func NewMethodNotAllowedError() *LambdaError {
	return &LambdaError{
		Type:       ErrorTypeMethodNotAllowed,
		Message:    "Method Not Allowed",
		StatusCode: http.StatusMethodNotAllowed,
	}
}

// NewConflictError creates a new conflict error
func NewConflictError(msg string, err error) *LambdaError {
	return &LambdaError{
		Type:       ErrorTypeConflict,
		Message:    msg,
		StatusCode: http.StatusConflict,
		Err:        err,
	}
}

// NewInternalServerError creates a new internal server error
func NewInternalServerError(msg string, err error) *LambdaError {
	return &LambdaError{
		Type:       ErrorTypeInternalServer,
		Message:    msg,
		StatusCode: http.StatusInternalServerError,
		Err:        err,
	}
}

// NewServiceUnavailableError creates a new service unavailable error
func NewServiceUnavailableError(msg string, err error) *LambdaError {
	return &LambdaError{
		Type:       ErrorTypeServiceUnavailable,
		Message:    msg,
		StatusCode: http.StatusServiceUnavailable,
		Err:        err,
	}
}

// NewValidationFailedError creates a new validation failed error
func NewValidationFailedError(msg string, err error) *LambdaError {
	return &LambdaError{
		Type:       ErrorTypeValidationFailed,
		Message:    msg,
		StatusCode: http.StatusBadRequest,
		Err:        err,
	}
}

// NewRequestFailedError creates a new request failed error
func NewRequestFailedError(msg string, err error) *LambdaError {
	return &LambdaError{
		Type:       ErrorTypeRequestFailed,
		Message:    msg,
		StatusCode: http.StatusBadGateway,
		Err:        err,
	}
}
