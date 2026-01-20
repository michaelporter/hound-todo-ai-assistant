package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	todov1 "hound-todo/api/todo/v1"
	"hound-todo/services/command/internal/ai"
	"hound-todo/services/command/internal/consumer"
	"hound-todo/services/command/internal/domain"
	"hound-todo/services/command/internal/publisher"
	"hound-todo/shared/logging"
)

const (
	// Confidence threshold below which we include AI explanation in reply
	lowConfidenceThreshold = 0.85
)

// Handler processes text commands using AI and executes them via gRPC
type Handler struct {
	ai        *ai.Client
	domain    *domain.Client
	publisher *publisher.Publisher
	logger    *logging.Logger
}

// New creates a new command handler
func New(aiClient *ai.Client, domainClient *domain.Client, pub *publisher.Publisher, logger *logging.Logger) *Handler {
	return &Handler{
		ai:        aiClient,
		domain:    domainClient,
		publisher: pub,
		logger:    logger,
	}
}

// Handle processes a single text message
func (h *Handler) Handle(ctx context.Context, msg *consumer.TextMessage) error {
	// Parse the command using AI
	cmd, err := h.ai.ParseCommand(ctx, msg.CommandText)
	if err != nil {
		h.logger.Error("AI parsing failed: %v", err)
		return fmt.Errorf("AI parsing failed: %w", err)
	}

	h.logger.Info("Parsed command: action=%s confidence=%.2f explanation=%s",
		cmd.Action, cmd.Confidence, cmd.Explanation)

	// Debug: log full parameters
	if paramsJSON, err := json.Marshal(cmd.Parameters); err == nil {
		fmt.Printf("[DEBUG] Parameters: %s\n", string(paramsJSON))
	}

	// Execute the command
	result, err := h.executeCommand(ctx, msg.UserID, msg.IdempotencyKey, cmd)
	if err != nil {
		h.logger.Error("Command execution failed: %v", err)
		return fmt.Errorf("command execution failed: %w", err)
	}

	// If confidence is low, append explanation to help user understand limitations
	if cmd.Confidence < lowConfidenceThreshold && cmd.Explanation != "" {
		result = fmt.Sprintf("%s\n\n(Note: %s)", result, cmd.Explanation)
	}

	h.logger.Info("Command result: %s", result)

	// Publish the result to be sent via SMS
	if err := h.publisher.PublishReply(ctx, msg.UserID, result); err != nil {
		h.logger.Error("Failed to publish reply: %v", err)
		return fmt.Errorf("failed to publish reply: %w", err)
	}

	return nil
}

// executeCommand runs the appropriate action based on the parsed command
func (h *Handler) executeCommand(ctx context.Context, userID, idempotencyKey string, cmd *ai.Command) (string, error) {
	switch cmd.Action {
	case "create":
		return h.handleCreate(ctx, userID, idempotencyKey, cmd)
	case "complete":
		return h.handleComplete(ctx, userID, idempotencyKey, cmd)
	case "list":
		return h.handleList(ctx, userID, cmd)
	case "delete":
		return h.handleDelete(ctx, userID, idempotencyKey, cmd)
	case "edit":
		return h.handleEdit(ctx, userID, idempotencyKey, cmd)
	case "nudge":
		return h.handleNudge(ctx, userID, cmd)
	case "unclear":
		return h.handleUnclear(cmd)
	default:
		return "", fmt.Errorf("unknown action: %s", cmd.Action)
	}
}

func (h *Handler) handleCreate(ctx context.Context, userID, idempotencyKey string, cmd *ai.Command) (string, error) {
	title := cmd.Parameters["title"]
	description := cmd.Parameters["description"]

	if title == "" {
		return "I couldn't figure out what to add. What would you like me to remember?", nil
	}

	todo, err := h.domain.CreateTodo(ctx, userID, title, description, idempotencyKey)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Added #%d: %s", todo.Id, todo.Title), nil
}

func (h *Handler) handleComplete(ctx context.Context, userID, idempotencyKey string, cmd *ai.Command) (string, error) {
	// Try to get todo by ID first
	if idStr := cmd.Parameters["todo_id"]; idStr != "" {
		todoID, err := strconv.ParseInt(idStr, 10, 64)
		if err == nil {
			todo, err := h.domain.CompleteTodo(ctx, todoID, userID, idempotencyKey)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("Completed #%d: %s ✓", todo.Id, todo.Title), nil
		}
	}

	// Try to find by title hint
	if hint := cmd.Parameters["title_hint"]; hint != "" {
		todo, err := h.domain.FindTodoByTitle(ctx, userID, hint)
		if err != nil {
			return "", err
		}
		if todo == nil {
			return fmt.Sprintf("I couldn't find a todo matching '%s'", hint), nil
		}

		completed, err := h.domain.CompleteTodo(ctx, todo.Id, userID, idempotencyKey)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("Completed #%d: %s ✓", completed.Id, completed.Title), nil
	}

	return "Which todo do you want to complete? Give me a number or describe it.", nil
}

func (h *Handler) handleList(ctx context.Context, userID string, cmd *ai.Command) (string, error) {
	filterParam := cmd.Parameters["filter"]

	// Build the filter
	filter := domain.ListTodosFilter{}

	switch filterParam {
	case "completed":
		filter.Status = todov1.TodoStatus_TODO_STATUS_COMPLETED
	case "all":
		filter.Status = todov1.TodoStatus_TODO_STATUS_UNSPECIFIED
	default:
		filter.Status = todov1.TodoStatus_TODO_STATUS_ACTIVE
	}

	// Parse date filters if provided
	if completedAfter := cmd.Parameters["completed_after"]; completedAfter != "" {
		if t, err := time.Parse(time.RFC3339, completedAfter); err == nil {
			filter.CompletedAfter = &t
		}
	}
	if completedBefore := cmd.Parameters["completed_before"]; completedBefore != "" {
		if t, err := time.Parse(time.RFC3339, completedBefore); err == nil {
			filter.CompletedBefore = &t
		}
	}

	todos, err := h.domain.ListTodos(ctx, userID, filter)
	if err != nil {
		return "", err
	}

	if len(todos) == 0 {
		if filter.CompletedAfter != nil || filter.CompletedBefore != nil {
			return "No completed todos found in that time range.", nil
		}
		return "Your list is empty! Text me something to remember.", nil
	}

	result := "Your todos:\n"
	for _, todo := range todos {
		statusMark := ""
		if todo.Status == todov1.TodoStatus_TODO_STATUS_COMPLETED {
			statusMark = " ✓"
		}
		result += fmt.Sprintf("#%d: %s%s\n", todo.Id, todo.Title, statusMark)
	}

	return result, nil
}

func (h *Handler) handleDelete(ctx context.Context, userID, idempotencyKey string, cmd *ai.Command) (string, error) {
	// Try to get todo by ID first
	if idStr := cmd.Parameters["todo_id"]; idStr != "" {
		todoID, err := strconv.ParseInt(idStr, 10, 64)
		if err == nil {
			todo, err := h.domain.DeleteTodo(ctx, todoID, userID, idempotencyKey)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("Deleted #%d: %s", todo.Id, todo.Title), nil
		}
	}

	// Try to find by title hint
	if hint := cmd.Parameters["title_hint"]; hint != "" {
		todo, err := h.domain.FindTodoByTitle(ctx, userID, hint)
		if err != nil {
			return "", err
		}
		if todo == nil {
			return fmt.Sprintf("I couldn't find a todo matching '%s'", hint), nil
		}

		deleted, err := h.domain.DeleteTodo(ctx, todo.Id, userID, idempotencyKey)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("Deleted #%d: %s", deleted.Id, deleted.Title), nil
	}

	return "Which todo do you want to delete? Give me a number or describe it.", nil
}

func (h *Handler) handleEdit(ctx context.Context, userID, idempotencyKey string, cmd *ai.Command) (string, error) {
	newTitle := cmd.Parameters["new_title"]
	newDescription := cmd.Parameters["new_description"]

	if newTitle == "" && newDescription == "" {
		return "What do you want to change it to?", nil
	}

	// Try to get todo by ID first
	if idStr := cmd.Parameters["todo_id"]; idStr != "" {
		todoID, err := strconv.ParseInt(idStr, 10, 64)
		if err == nil {
			todo, err := h.domain.EditTodo(ctx, todoID, userID, newTitle, newDescription, idempotencyKey)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("Updated #%d: %s", todo.Id, todo.Title), nil
		}
	}

	// Try to find by title hint
	if hint := cmd.Parameters["title_hint"]; hint != "" {
		todo, err := h.domain.FindTodoByTitle(ctx, userID, hint)
		if err != nil {
			return "", err
		}
		if todo == nil {
			return fmt.Sprintf("I couldn't find a todo matching '%s'", hint), nil
		}

		// If no new title specified, keep the old one
		if newTitle == "" {
			newTitle = todo.Title
		}
		if newDescription == "" {
			newDescription = todo.Description
		}

		edited, err := h.domain.EditTodo(ctx, todo.Id, userID, newTitle, newDescription, idempotencyKey)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("Updated #%d: %s", edited.Id, edited.Title), nil
	}

	return "Which todo do you want to edit? Give me a number or describe it.", nil
}

func (h *Handler) handleNudge(ctx context.Context, userID string, cmd *ai.Command) (string, error) {
	suggestedAction := cmd.Parameters["suggested_action"]
	taskContext := cmd.Parameters["task_context"]

	if suggestedAction == "" {
		return "I'm not sure how to help with that. What are you trying to start?", nil
	}

	// The LLM already generated the nudge - just return it
	response := suggestedAction

	// Add some encouragement
	if taskContext != "" {
		response = fmt.Sprintf("For %s: %s", taskContext, suggestedAction)
	}

	return response, nil
}

func (h *Handler) handleUnclear(cmd *ai.Command) (string, error) {
	reason := cmd.Parameters["reason"]
	if reason != "" {
		return fmt.Sprintf("I didn't quite understand that. %s", reason), nil
	}
	return "I didn't understand. Try 'add [task]', 'done with [task]', or 'show my list'", nil
}
