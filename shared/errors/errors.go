package errors

import "fmt"

// Common error types for the todo application

// NotFoundError represents a resource that doesn't exist
type NotFoundError struct {
	Resource string
	ID       string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s not found: %s", e.Resource, e.ID)
}

// ConflictError represents a conflict with existing state
type ConflictError struct {
	Message string
}

func (e *ConflictError) Error() string {
	return e.Message
}

// ValidationError represents invalid input
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error on field '%s': %s", e.Field, e.Message)
}
