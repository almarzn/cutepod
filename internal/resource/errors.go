package resource

import (
	"fmt"
)

// ErrorType represents the category of reconciliation error
type ErrorType string

const (
	ErrorTypeDependency    ErrorType = "dependency"
	ErrorTypeValidation    ErrorType = "validation"
	ErrorTypePodmanAPI     ErrorType = "podman_api"
	ErrorTypeConfiguration ErrorType = "configuration"
)

// ReconciliationError represents an error that occurred during reconciliation
type ReconciliationError struct {
	Type        ErrorType         `json:"type"`
	Resource    ResourceReference `json:"resource"`
	Message     string            `json:"message"`
	Cause       error             `json:"cause,omitempty"`
	Recoverable bool              `json:"recoverable"`
}

// Error implements the error interface
func (r *ReconciliationError) Error() string {
	resourceInfo := fmt.Sprintf("%s/%s", r.Resource.Type, r.Resource.Name)
	return fmt.Sprintf("[%s] %s: %s", r.Type, resourceInfo, r.Message)
}

// Unwrap returns the underlying cause error
func (r *ReconciliationError) Unwrap() error {
	return r.Cause
}

// NewReconciliationError creates a new ReconciliationError
func NewReconciliationError(errorType ErrorType, resource ResourceReference, message string, cause error, recoverable bool) *ReconciliationError {
	return &ReconciliationError{
		Type:        errorType,
		Resource:    resource,
		Message:     message,
		Cause:       cause,
		Recoverable: recoverable,
	}
}

// NewDependencyError creates a dependency-related error
func NewDependencyError(resource ResourceReference, message string, cause error) *ReconciliationError {
	return NewReconciliationError(ErrorTypeDependency, resource, message, cause, false)
}

// NewValidationError creates a validation-related error
func NewValidationError(resource ResourceReference, message string, cause error) *ReconciliationError {
	return NewReconciliationError(ErrorTypeValidation, resource, message, cause, false)
}

// NewPodmanAPIError creates a Podman API-related error
func NewPodmanAPIError(resource ResourceReference, message string, cause error, recoverable bool) *ReconciliationError {
	return NewReconciliationError(ErrorTypePodmanAPI, resource, message, cause, recoverable)
}

// NewConfigurationError creates a configuration-related error
func NewConfigurationError(resource ResourceReference, message string, cause error) *ReconciliationError {
	return NewReconciliationError(ErrorTypeConfiguration, resource, message, cause, false)
}

// IsReconciliationError checks if an error is a ReconciliationError
func IsReconciliationError(err error) bool {
	_, ok := err.(*ReconciliationError)
	return ok
}

// AsReconciliationError attempts to cast an error to ReconciliationError
func AsReconciliationError(err error) (*ReconciliationError, bool) {
	if recErr, ok := err.(*ReconciliationError); ok {
		return recErr, true
	}
	return nil, false
}