package cognito

import (
	"fmt"
)

// Error types
const (
	ErrorTypeInvalidRequest   = "InvalidRequest"
	ErrorTypeRequestFailed    = "RequestFailed"
	ErrorTypeParsingFailed    = "ParsingFailed"
	ErrorTypeInvalidToken     = "InvalidToken"
	ErrorTypeTokenExpired     = "TokenExpired"
	ErrorTypeAuthFailed       = "AuthenticationFailed"
	ErrorTypeUserPoolError    = "UserPoolError"
	ErrorTypeValidationFailed = "ValidationFailed"
)

// CognitoError represents a custom error in the cognito package
type CognitoError struct {
	Type    string
	Message string
	Err     error
}

// Error implements the error interface
func (e *CognitoError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Type, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// Unwrap returns the wrapped error
func (e *CognitoError) Unwrap() error {
	return e.Err
}

// NewInvalidRequestError creates a new invalid request error
func NewInvalidRequestError(msg string, err error) *CognitoError {
	return &CognitoError{
		Type:    ErrorTypeInvalidRequest,
		Message: msg,
		Err:     err,
	}
}

// NewRequestFailedError creates a new request failed error
func NewRequestFailedError(msg string, err error) *CognitoError {
	return &CognitoError{
		Type:    ErrorTypeRequestFailed,
		Message: msg,
		Err:     err,
	}
}

// NewParsingFailedError creates a new parsing failed error
func NewParsingFailedError(msg string, err error) *CognitoError {
	return &CognitoError{
		Type:    ErrorTypeParsingFailed,
		Message: msg,
		Err:     err,
	}
}

// NewInvalidTokenError creates a new invalid token error
func NewInvalidTokenError(msg string, err error) *CognitoError {
	return &CognitoError{
		Type:    ErrorTypeInvalidToken,
		Message: msg,
		Err:     err,
	}
}

// NewTokenExpiredError creates a new token expired error
func NewTokenExpiredError() *CognitoError {
	return &CognitoError{
		Type:    ErrorTypeTokenExpired,
		Message: "token has expired",
	}
}

// NewAuthFailedError creates a new authentication failed error
func NewAuthFailedError(msg string, err error) *CognitoError {
	return &CognitoError{
		Type:    ErrorTypeAuthFailed,
		Message: msg,
		Err:     err,
	}
}

// NewUserPoolError creates a new user pool related error
func NewUserPoolError(msg string) *CognitoError {
	return &CognitoError{
		Type:    ErrorTypeUserPoolError,
		Message: msg,
	}
}

// NewValidationFailedError creates a new validation failed error
func NewValidationFailedError(msg string) *CognitoError {
	return &CognitoError{
		Type:    ErrorTypeValidationFailed,
		Message: msg,
	}
}
