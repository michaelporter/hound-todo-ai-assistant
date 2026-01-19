package domain

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	todov1 "hound-todo/api/todo/v1"
)

// Client wraps the gRPC client for todo-domain-svc
type Client struct {
	conn   *grpc.ClientConn
	client todov1.TodoDomainClient
}

// NewClient creates a new gRPC client connection
func NewClient(addr string) (*Client, error) {
	// Connect to the gRPC server
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to todo-domain: %w", err)
	}

	return &Client{
		conn:   conn,
		client: todov1.NewTodoDomainClient(conn),
	}, nil
}

// Close closes the gRPC connection
func (c *Client) Close() error {
	return c.conn.Close()
}

// CreateTodo creates a new todo
func (c *Client) CreateTodo(ctx context.Context, userID, title, description, idempotencyKey string) (*todov1.Todo, error) {
	resp, err := c.client.CreateTodo(ctx, &todov1.CreateTodoRequest{
		UserId:         userID,
		Title:          title,
		Description:    description,
		IdempotencyKey: idempotencyKey,
	})
	if err != nil {
		return nil, err
	}
	return resp.Todo, nil
}

// CompleteTodo marks a todo as completed
func (c *Client) CompleteTodo(ctx context.Context, todoID int64, userID, idempotencyKey string) (*todov1.Todo, error) {
	resp, err := c.client.CompleteTodo(ctx, &todov1.CompleteTodoRequest{
		TodoId:         todoID,
		UserId:         userID,
		IdempotencyKey: idempotencyKey,
	})
	if err != nil {
		return nil, err
	}
	return resp.Todo, nil
}

// ListTodos retrieves todos for a user
func (c *Client) ListTodos(ctx context.Context, userID string, status todov1.TodoStatus) ([]*todov1.Todo, error) {
	resp, err := c.client.ListTodos(ctx, &todov1.ListTodosRequest{
		UserId: userID,
		Status: status,
	})
	if err != nil {
		return nil, err
	}
	return resp.Todos, nil
}

// DeleteTodo soft-deletes a todo
func (c *Client) DeleteTodo(ctx context.Context, todoID int64, userID, idempotencyKey string) (*todov1.Todo, error) {
	resp, err := c.client.DeleteTodo(ctx, &todov1.DeleteTodoRequest{
		TodoId:         todoID,
		UserId:         userID,
		IdempotencyKey: idempotencyKey,
	})
	if err != nil {
		return nil, err
	}
	return resp.Todo, nil
}

// EditTodo updates a todo
func (c *Client) EditTodo(ctx context.Context, todoID int64, userID, title, description, idempotencyKey string) (*todov1.Todo, error) {
	resp, err := c.client.EditTodo(ctx, &todov1.EditTodoRequest{
		TodoId:         todoID,
		UserId:         userID,
		Title:          title,
		Description:    description,
		IdempotencyKey: idempotencyKey,
	})
	if err != nil {
		return nil, err
	}
	return resp.Todo, nil
}

// FindTodoByTitle searches for a todo by partial title match
// Returns the first matching active todo, or nil if not found
func (c *Client) FindTodoByTitle(ctx context.Context, userID, titleHint string) (*todov1.Todo, error) {
	todos, err := c.ListTodos(ctx, userID, todov1.TodoStatus_TODO_STATUS_ACTIVE)
	if err != nil {
		return nil, err
	}

	// Simple substring match - could be improved with fuzzy matching
	for _, todo := range todos {
		if containsIgnoreCase(todo.Title, titleHint) {
			return todo, nil
		}
	}

	return nil, nil // Not found
}

// containsIgnoreCase checks if s contains substr (case-insensitive)
func containsIgnoreCase(s, substr string) bool {
	sLower := toLower(s)
	substrLower := toLower(substr)
	return contains(sLower, substrLower)
}

func toLower(s string) string {
	b := make([]byte, len(s))
	for i := range s {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
