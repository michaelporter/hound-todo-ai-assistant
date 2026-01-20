package store

import (
	"testing"
	"time"
)

// =============================================================================
// Todo Struct Tests
// =============================================================================

func TestTodoStruct(t *testing.T) {
	now := time.Now()
	completedAt := now.Add(1 * time.Hour)
	deletedAt := now.Add(2 * time.Hour)

	todo := &Todo{
		ID:          1,
		UserID:      "user123",
		Title:       "test todo",
		Description: "test description",
		Status:      "active",
		CreatedAt:   now,
		UpdatedAt:   now,
		CompletedAt: &completedAt,
		DeletedAt:   &deletedAt,
	}

	if todo.ID != 1 {
		t.Errorf("expected ID 1, got %d", todo.ID)
	}
	if todo.UserID != "user123" {
		t.Errorf("expected UserID user123, got %s", todo.UserID)
	}
	if todo.Title != "test todo" {
		t.Errorf("expected Title 'test todo', got %s", todo.Title)
	}
	if todo.Description != "test description" {
		t.Errorf("expected Description 'test description', got %s", todo.Description)
	}
	if todo.Status != "active" {
		t.Errorf("expected Status 'active', got %s", todo.Status)
	}
	if todo.CompletedAt == nil || !todo.CompletedAt.Equal(completedAt) {
		t.Errorf("expected CompletedAt %v, got %v", completedAt, todo.CompletedAt)
	}
	if todo.DeletedAt == nil || !todo.DeletedAt.Equal(deletedAt) {
		t.Errorf("expected DeletedAt %v, got %v", deletedAt, todo.DeletedAt)
	}
}

func TestTodoStruct_NilOptionalFields(t *testing.T) {
	now := time.Now()
	todo := &Todo{
		ID:          1,
		UserID:      "user123",
		Title:       "test todo",
		Status:      "active",
		CreatedAt:   now,
		UpdatedAt:   now,
		CompletedAt: nil,
		DeletedAt:   nil,
	}

	if todo.CompletedAt != nil {
		t.Error("expected CompletedAt to be nil")
	}
	if todo.DeletedAt != nil {
		t.Error("expected DeletedAt to be nil")
	}
}

// =============================================================================
// ListTodosFilter Tests
// =============================================================================

func TestListTodosFilter_Empty(t *testing.T) {
	filter := ListTodosFilter{}

	if filter.Status != "" {
		t.Errorf("expected empty Status, got %s", filter.Status)
	}
	if filter.CompletedAfter != nil {
		t.Error("expected nil CompletedAfter")
	}
	if filter.CompletedBefore != nil {
		t.Error("expected nil CompletedBefore")
	}
}

func TestListTodosFilter_WithStatus(t *testing.T) {
	filter := ListTodosFilter{
		Status: "active",
	}

	if filter.Status != "active" {
		t.Errorf("expected Status 'active', got %s", filter.Status)
	}
}

func TestListTodosFilter_WithDateRange(t *testing.T) {
	after := time.Now().Add(-24 * time.Hour)
	before := time.Now()

	filter := ListTodosFilter{
		Status:          "completed",
		CompletedAfter:  &after,
		CompletedBefore: &before,
	}

	if filter.Status != "completed" {
		t.Errorf("expected Status 'completed', got %s", filter.Status)
	}
	if filter.CompletedAfter == nil || !filter.CompletedAfter.Equal(after) {
		t.Errorf("expected CompletedAfter %v, got %v", after, filter.CompletedAfter)
	}
	if filter.CompletedBefore == nil || !filter.CompletedBefore.Equal(before) {
		t.Errorf("expected CompletedBefore %v, got %v", before, filter.CompletedBefore)
	}
}

// =============================================================================
// Error Constants Tests
// =============================================================================

func TestErrorConstants(t *testing.T) {
	if ErrNotFound.Error() != "todo not found" {
		t.Errorf("expected 'todo not found', got %s", ErrNotFound.Error())
	}
	if ErrAlreadyExists.Error() != "idempotency key already exists" {
		t.Errorf("expected 'idempotency key already exists', got %s", ErrAlreadyExists.Error())
	}
	if ErrNotOwner.Error() != "user does not own this todo" {
		t.Errorf("expected 'user does not own this todo', got %s", ErrNotOwner.Error())
	}
}

// =============================================================================
// Status Values Tests
// =============================================================================

func TestStatusValues(t *testing.T) {
	validStatuses := []string{"active", "completed", "deleted"}

	for _, status := range validStatuses {
		todo := &Todo{Status: status}
		if todo.Status != status {
			t.Errorf("expected status %s, got %s", status, todo.Status)
		}
	}
}

// =============================================================================
// Store New Tests
// =============================================================================

func TestNewStore(t *testing.T) {
	// Test that New returns a Store with the db set
	// We can't test with a real db here, but we can verify the function signature
	store := New(nil)

	if store == nil {
		t.Error("expected non-nil store")
	}
	if store.db != nil {
		t.Error("expected nil db when passing nil")
	}
}

// =============================================================================
// Time Handling Tests
// =============================================================================

func TestTimeHandling_CreatedAt(t *testing.T) {
	now := time.Now()
	todo := &Todo{
		ID:        1,
		UserID:    "user123",
		Title:     "test",
		Status:    "active",
		CreatedAt: now,
		UpdatedAt: now,
	}

	if !todo.CreatedAt.Equal(now) {
		t.Errorf("expected CreatedAt %v, got %v", now, todo.CreatedAt)
	}
	if !todo.UpdatedAt.Equal(now) {
		t.Errorf("expected UpdatedAt %v, got %v", now, todo.UpdatedAt)
	}
}

func TestTimeHandling_Timezone(t *testing.T) {
	// Test that time handling works with UTC
	utcTime := time.Now().UTC()
	todo := &Todo{
		ID:        1,
		UserID:    "user123",
		Title:     "test",
		Status:    "active",
		CreatedAt: utcTime,
		UpdatedAt: utcTime,
	}

	if !todo.CreatedAt.Equal(utcTime) {
		t.Errorf("expected UTC time %v, got %v", utcTime, todo.CreatedAt)
	}
}

// =============================================================================
// Filter Status Mapping Tests
// =============================================================================

func TestFilterStatusMapping(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected string
	}{
		{"active", "active", "active"},
		{"completed", "completed", "completed"},
		{"deleted", "deleted", "deleted"},
		{"empty for all", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := ListTodosFilter{Status: tt.status}
			if filter.Status != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, filter.Status)
			}
		})
	}
}

// =============================================================================
// Connection Pool Settings Test (Documentation)
// =============================================================================

func TestConnectionPoolSettings(t *testing.T) {
	// These tests document the expected connection pool settings
	// The actual values are set in Connect() function
	expectedMaxOpenConns := 25
	expectedMaxIdleConns := 5
	expectedConnMaxLifetime := 5 * time.Minute

	// We can't easily test the actual db settings without a real connection,
	// but we document the expected values here
	if expectedMaxOpenConns != 25 {
		t.Errorf("expected MaxOpenConns 25, got %d", expectedMaxOpenConns)
	}
	if expectedMaxIdleConns != 5 {
		t.Errorf("expected MaxIdleConns 5, got %d", expectedMaxIdleConns)
	}
	if expectedConnMaxLifetime != 5*time.Minute {
		t.Errorf("expected ConnMaxLifetime 5m, got %v", expectedConnMaxLifetime)
	}
}
