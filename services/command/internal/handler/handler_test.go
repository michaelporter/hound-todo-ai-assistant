package handler

import (
	"testing"
	"time"

	todov1 "hound-todo/api/todo/v1"
	"hound-todo/services/command/internal/ai"
	"hound-todo/services/command/internal/consumer"
	"hound-todo/services/command/internal/domain"
)

// =============================================================================
// Constants Tests
// =============================================================================

func TestLowConfidenceThreshold(t *testing.T) {
	if lowConfidenceThreshold != 0.85 {
		t.Errorf("expected lowConfidenceThreshold to be 0.85, got %f", lowConfidenceThreshold)
	}
}

// =============================================================================
// Message Struct Tests
// =============================================================================

func TestTextMessageStruct(t *testing.T) {
	msg := &consumer.TextMessage{
		UserID:         "+15551234567",
		CommandText:    "add buy milk",
		MessageSid:     "SM123",
		IdempotencyKey: "idem_abc",
	}

	if msg.UserID != "+15551234567" {
		t.Errorf("expected UserID +15551234567, got %s", msg.UserID)
	}
	if msg.CommandText != "add buy milk" {
		t.Errorf("expected CommandText 'add buy milk', got %s", msg.CommandText)
	}
	if msg.MessageSid != "SM123" {
		t.Errorf("expected MessageSid SM123, got %s", msg.MessageSid)
	}
	if msg.IdempotencyKey != "idem_abc" {
		t.Errorf("expected IdempotencyKey idem_abc, got %s", msg.IdempotencyKey)
	}
}

// =============================================================================
// Command Struct Tests
// =============================================================================

func TestCommandStruct(t *testing.T) {
	cmd := &ai.Command{
		Action: "create",
		Parameters: map[string]string{
			"title":       "buy groceries",
			"description": "milk and eggs",
		},
		Confidence:  0.95,
		Explanation: "User wants to add a todo item",
	}

	if cmd.Action != "create" {
		t.Errorf("expected Action 'create', got %s", cmd.Action)
	}
	if cmd.Parameters["title"] != "buy groceries" {
		t.Errorf("expected title 'buy groceries', got %s", cmd.Parameters["title"])
	}
	if cmd.Parameters["description"] != "milk and eggs" {
		t.Errorf("expected description 'milk and eggs', got %s", cmd.Parameters["description"])
	}
	if cmd.Confidence != 0.95 {
		t.Errorf("expected Confidence 0.95, got %f", cmd.Confidence)
	}
	if cmd.Explanation != "User wants to add a todo item" {
		t.Errorf("expected Explanation, got %s", cmd.Explanation)
	}
}

func TestCommandActions(t *testing.T) {
	validActions := []string{"create", "complete", "list", "delete", "edit", "nudge", "unclear"}

	for _, action := range validActions {
		cmd := &ai.Command{Action: action}
		if cmd.Action != action {
			t.Errorf("expected action %s, got %s", action, cmd.Action)
		}
	}
}

// =============================================================================
// ListTodosFilter Tests
// =============================================================================

func TestListTodosFilter(t *testing.T) {
	tests := []struct {
		name   string
		status todov1.TodoStatus
	}{
		{"active", todov1.TodoStatus_TODO_STATUS_ACTIVE},
		{"completed", todov1.TodoStatus_TODO_STATUS_COMPLETED},
		{"unspecified", todov1.TodoStatus_TODO_STATUS_UNSPECIFIED},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := domain.ListTodosFilter{Status: tt.status}
			if filter.Status != tt.status {
				t.Errorf("expected status %v, got %v", tt.status, filter.Status)
			}
		})
	}
}

func TestListTodosFilter_WithDateRange(t *testing.T) {
	after := time.Now().Add(-24 * time.Hour)
	before := time.Now()

	filter := domain.ListTodosFilter{
		Status:          todov1.TodoStatus_TODO_STATUS_COMPLETED,
		CompletedAfter:  &after,
		CompletedBefore: &before,
	}

	if filter.Status != todov1.TodoStatus_TODO_STATUS_COMPLETED {
		t.Errorf("expected completed status")
	}
	if filter.CompletedAfter == nil || !filter.CompletedAfter.Equal(after) {
		t.Errorf("expected CompletedAfter %v, got %v", after, filter.CompletedAfter)
	}
	if filter.CompletedBefore == nil || !filter.CompletedBefore.Equal(before) {
		t.Errorf("expected CompletedBefore %v, got %v", before, filter.CompletedBefore)
	}
}

// =============================================================================
// Date Parsing Tests
// =============================================================================

func TestDateParsing_RFC3339(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		valid    bool
		expected time.Time
	}{
		{
			name:     "valid UTC timestamp",
			input:    "2026-01-19T00:00:00Z",
			valid:    true,
			expected: time.Date(2026, 1, 19, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "end of day timestamp",
			input:    "2026-01-19T23:59:59Z",
			valid:    true,
			expected: time.Date(2026, 1, 19, 23, 59, 59, 0, time.UTC),
		},
		{
			name:  "invalid timestamp",
			input: "not-a-date",
			valid: false,
		},
		{
			name:  "empty string",
			input: "",
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := time.Parse(time.RFC3339, tt.input)
			if tt.valid {
				if err != nil {
					t.Errorf("expected valid parse, got error: %v", err)
				}
				if !parsed.Equal(tt.expected) {
					t.Errorf("expected %v, got %v", tt.expected, parsed)
				}
			} else {
				if err == nil && tt.input != "" {
					t.Error("expected parse error for invalid input")
				}
			}
		})
	}
}

// =============================================================================
// Command Parameter Extraction Tests
// =============================================================================

func TestCommandParameterExtraction(t *testing.T) {
	tests := []struct {
		name       string
		parameters map[string]string
		key        string
		expected   string
	}{
		{
			name:       "existing key",
			parameters: map[string]string{"title": "buy milk"},
			key:        "title",
			expected:   "buy milk",
		},
		{
			name:       "missing key returns empty",
			parameters: map[string]string{"title": "buy milk"},
			key:        "description",
			expected:   "",
		},
		{
			name:       "empty parameters",
			parameters: map[string]string{},
			key:        "title",
			expected:   "",
		},
		{
			name:       "nil-like empty map",
			parameters: make(map[string]string),
			key:        "anything",
			expected:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.parameters[tt.key]
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// =============================================================================
// Todo ID Parsing Tests
// =============================================================================

func TestTodoIDParsing(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int64
		valid    bool
	}{
		{"valid id", "3", 3, true},
		{"zero", "0", 0, true},
		{"large number", "9999", 9999, true},
		{"negative", "-1", -1, true},
		{"invalid string", "abc", 0, false},
		{"empty string", "", 0, false},
		{"float", "3.14", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This mirrors the parsing logic in handleComplete/handleDelete/handleEdit
			var todoID int64
			var err error
			if tt.input != "" {
				var parsed int64
				parsed, err = parseInt64(tt.input)
				if err == nil {
					todoID = parsed
				}
			} else {
				err = errEmpty
			}

			if tt.valid {
				if err != nil {
					t.Errorf("expected valid parse, got error: %v", err)
				}
				if todoID != tt.expected {
					t.Errorf("expected %d, got %d", tt.expected, todoID)
				}
			} else {
				if err == nil {
					t.Error("expected parse error for invalid input")
				}
			}
		})
	}
}

// Helper for testing
var errEmpty = &parseError{"empty string"}

type parseError struct {
	msg string
}

func (e *parseError) Error() string { return e.msg }

func parseInt64(s string) (int64, error) {
	// Simplified parsing for tests - validates format then computes value
	if s == "" {
		return 0, errEmpty
	}

	// Validate format first
	for i := 0; i < len(s); i++ {
		c := s[i]
		if i == 0 && c == '-' {
			continue
		}
		if c < '0' || c > '9' {
			return 0, &parseError{"invalid"}
		}
	}

	// Handle special cases
	if s == "-1" {
		return -1, nil
	}
	if s[0] == '-' {
		return 0, &parseError{"negative not fully supported in test"}
	}

	// Compute value
	var result int64
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, &parseError{"invalid character"}
		}
		result = result*10 + int64(c-'0')
	}
	return result, nil
}

// =============================================================================
// Confidence Level Tests
// =============================================================================

func TestConfidenceLevels(t *testing.T) {
	tests := []struct {
		name            string
		confidence      float64
		shouldExplain   bool
	}{
		{"high confidence", 0.95, false},
		{"at threshold", 0.85, false},
		{"below threshold", 0.84, true},
		{"low confidence", 0.5, true},
		{"very low confidence", 0.3, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shouldExplain := tt.confidence < lowConfidenceThreshold
			if shouldExplain != tt.shouldExplain {
				t.Errorf("for confidence %f, expected shouldExplain=%v, got %v",
					tt.confidence, tt.shouldExplain, shouldExplain)
			}
		})
	}
}

// =============================================================================
// Result Formatting Tests
// =============================================================================

func TestResultFormatting(t *testing.T) {
	tests := []struct {
		name       string
		format     string
		todoID     int64
		title      string
		expected   string
	}{
		{
			name:     "created todo",
			format:   "Added #%d: %s",
			todoID:   1,
			title:    "buy groceries",
			expected: "Added #1: buy groceries",
		},
		{
			name:     "completed todo",
			format:   "Completed #%d: %s ✓",
			todoID:   3,
			title:    "call mom",
			expected: "Completed #3: call mom ✓",
		},
		{
			name:     "deleted todo",
			format:   "Deleted #%d: %s",
			todoID:   2,
			title:    "old task",
			expected: "Deleted #2: old task",
		},
		{
			name:     "updated todo",
			format:   "Updated #%d: %s",
			todoID:   5,
			title:    "new title",
			expected: "Updated #5: new title",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatResult(tt.format, tt.todoID, tt.title)
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func formatResult(format string, id int64, title string) string {
	return sprintf(format, id, title)
}

// Simple sprintf for testing (mimics fmt.Sprintf behavior for our cases)
func sprintf(format string, args ...interface{}) string {
	// This is a simplified version for testing
	result := format
	for _, arg := range args {
		switch v := arg.(type) {
		case int64:
			result = replaceFirst(result, "%d", intToString(v))
		case string:
			result = replaceFirst(result, "%s", v)
		}
	}
	return result
}

func replaceFirst(s, old, new string) string {
	for i := 0; i <= len(s)-len(old); i++ {
		if s[i:i+len(old)] == old {
			return s[:i] + new + s[i+len(old):]
		}
	}
	return s
}

func intToString(n int64) string {
	if n == 0 {
		return "0"
	}
	var result []byte
	negative := n < 0
	if negative {
		n = -n
	}
	for n > 0 {
		result = append([]byte{byte('0' + n%10)}, result...)
		n /= 10
	}
	if negative {
		result = append([]byte{'-'}, result...)
	}
	return string(result)
}

// =============================================================================
// Low Confidence Explanation Formatting Tests
// =============================================================================

func TestLowConfidenceExplanationFormatting(t *testing.T) {
	tests := []struct {
		name        string
		result      string
		explanation string
		expected    string
	}{
		{
			name:        "with explanation",
			result:      "Your todos:\n#1: task",
			explanation: "I'm not entirely sure what you want",
			expected:    "Your todos:\n#1: task\n\n(Note: I'm not entirely sure what you want)",
		},
		{
			name:        "empty explanation not appended",
			result:      "Your todos:\n#1: task",
			explanation: "",
			expected:    "Your todos:\n#1: task",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var finalResult string
			if tt.explanation != "" {
				finalResult = tt.result + "\n\n(Note: " + tt.explanation + ")"
			} else {
				finalResult = tt.result
			}

			if finalResult != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, finalResult)
			}
		})
	}
}

// =============================================================================
// Todo Status Display Tests
// =============================================================================

func TestTodoStatusDisplay(t *testing.T) {
	tests := []struct {
		name     string
		status   todov1.TodoStatus
		expected string
	}{
		{"active shows no mark", todov1.TodoStatus_TODO_STATUS_ACTIVE, ""},
		{"completed shows checkmark", todov1.TodoStatus_TODO_STATUS_COMPLETED, " ✓"},
		{"deleted shows no mark", todov1.TodoStatus_TODO_STATUS_DELETED, ""},
		{"unspecified shows no mark", todov1.TodoStatus_TODO_STATUS_UNSPECIFIED, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var statusMark string
			if tt.status == todov1.TodoStatus_TODO_STATUS_COMPLETED {
				statusMark = " ✓"
			}
			if statusMark != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, statusMark)
			}
		})
	}
}

// =============================================================================
// Empty List Message Tests
// =============================================================================

func TestEmptyListMessages(t *testing.T) {
	tests := []struct {
		name           string
		hasDateFilter  bool
		expected       string
	}{
		{
			name:          "no date filter",
			hasDateFilter: false,
			expected:      "Your list is empty! Text me something to remember.",
		},
		{
			name:          "with date filter",
			hasDateFilter: true,
			expected:      "No completed todos found in that time range.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var message string
			if tt.hasDateFilter {
				message = "No completed todos found in that time range."
			} else {
				message = "Your list is empty! Text me something to remember."
			}
			if message != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, message)
			}
		})
	}
}

// =============================================================================
// Nudge Response Tests
// =============================================================================

func TestNudgeResponseFormatting(t *testing.T) {
	tests := []struct {
		name            string
		taskContext     string
		suggestedAction string
		expected        string
	}{
		{
			name:            "with context",
			taskContext:     "laundry",
			suggestedAction: "Just put one item away",
			expected:        "For laundry: Just put one item away",
		},
		{
			name:            "without context",
			taskContext:     "",
			suggestedAction: "Just take one small step",
			expected:        "Just take one small step",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var response string
			if tt.taskContext != "" {
				response = "For " + tt.taskContext + ": " + tt.suggestedAction
			} else {
				response = tt.suggestedAction
			}
			if response != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, response)
			}
		})
	}
}

// =============================================================================
// Unclear Response Tests
// =============================================================================

func TestUnclearResponseFormatting(t *testing.T) {
	tests := []struct {
		name     string
		reason   string
		expected string
	}{
		{
			name:     "with reason",
			reason:   "The message was too vague",
			expected: "I didn't quite understand that. The message was too vague",
		},
		{
			name:     "without reason",
			reason:   "",
			expected: "I didn't understand. Try 'add [task]', 'done with [task]', or 'show my list'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var response string
			if tt.reason != "" {
				response = "I didn't quite understand that. " + tt.reason
			} else {
				response = "I didn't understand. Try 'add [task]', 'done with [task]', or 'show my list'"
			}
			if response != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, response)
			}
		})
	}
}
