package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
)

var (
	ErrNotFound       = errors.New("todo not found")
	ErrAlreadyExists  = errors.New("idempotency key already exists")
	ErrNotOwner       = errors.New("user does not own this todo")
)

// Todo represents a todo item in the database
type Todo struct {
	ID          int64
	UserID      string
	Title       string
	Description string
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	CompletedAt *time.Time
	DeletedAt   *time.Time
}

// Store handles all database operations for todos
type Store struct {
	db *sql.DB
}

// New creates a new Store with the given database connection
func New(db *sql.DB) *Store {
	return &Store{db: db}
}

// Connect establishes a connection to the PostgreSQL database
func Connect(databaseURL string) (*sql.DB, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, err
	}

	// Verify connection
	if err := db.Ping(); err != nil {
		return nil, err
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	return db, nil
}

// CreateTodo inserts a new todo and returns it with the generated ID
func (s *Store) CreateTodo(ctx context.Context, userID, title, description string) (*Todo, error) {
	todo := &Todo{
		UserID:      userID,
		Title:       title,
		Description: description,
		Status:      "active",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	err := s.db.QueryRowContext(ctx, `
		INSERT INTO todos (user_id, title, description, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, todo.UserID, todo.Title, todo.Description, todo.Status, todo.CreatedAt, todo.UpdatedAt).Scan(&todo.ID)

	if err != nil {
		return nil, err
	}

	return todo, nil
}

// GetTodo retrieves a todo by ID
func (s *Store) GetTodo(ctx context.Context, id int64) (*Todo, error) {
	todo := &Todo{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, title, description, status, created_at, updated_at, completed_at, deleted_at
		FROM todos
		WHERE id = $1
	`, id).Scan(
		&todo.ID, &todo.UserID, &todo.Title, &todo.Description, &todo.Status,
		&todo.CreatedAt, &todo.UpdatedAt, &todo.CompletedAt, &todo.DeletedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return todo, nil
}

// ListTodos retrieves all todos for a user, optionally filtered by status
func (s *Store) ListTodos(ctx context.Context, userID string, status string) ([]*Todo, error) {
	var rows *sql.Rows
	var err error

	if status != "" {
		rows, err = s.db.QueryContext(ctx, `
			SELECT id, user_id, title, description, status, created_at, updated_at, completed_at, deleted_at
			FROM todos
			WHERE user_id = $1 AND status = $2
			ORDER BY created_at DESC
		`, userID, status)
	} else {
		// Default: show active and completed, not deleted
		rows, err = s.db.QueryContext(ctx, `
			SELECT id, user_id, title, description, status, created_at, updated_at, completed_at, deleted_at
			FROM todos
			WHERE user_id = $1 AND status != 'deleted'
			ORDER BY created_at DESC
		`, userID)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var todos []*Todo
	for rows.Next() {
		todo := &Todo{}
		err := rows.Scan(
			&todo.ID, &todo.UserID, &todo.Title, &todo.Description, &todo.Status,
			&todo.CreatedAt, &todo.UpdatedAt, &todo.CompletedAt, &todo.DeletedAt,
		)
		if err != nil {
			return nil, err
		}
		todos = append(todos, todo)
	}

	return todos, rows.Err()
}

// CompleteTodo marks a todo as completed
func (s *Store) CompleteTodo(ctx context.Context, id int64, userID string, completedAt time.Time) (*Todo, error) {
	result, err := s.db.ExecContext(ctx, `
		UPDATE todos
		SET status = 'completed', completed_at = $1, updated_at = $2
		WHERE id = $3 AND user_id = $4 AND status = 'active'
	`, completedAt, time.Now(), id, userID)

	if err != nil {
		return nil, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}

	if rowsAffected == 0 {
		// Check if todo exists
		existing, err := s.GetTodo(ctx, id)
		if err == ErrNotFound {
			return nil, ErrNotFound
		}
		if err != nil {
			return nil, err
		}
		if existing.UserID != userID {
			return nil, ErrNotOwner
		}
		// Already completed or deleted - return current state
		return existing, nil
	}

	return s.GetTodo(ctx, id)
}

// DeleteTodo soft-deletes a todo
func (s *Store) DeleteTodo(ctx context.Context, id int64, userID string) (*Todo, error) {
	now := time.Now()
	result, err := s.db.ExecContext(ctx, `
		UPDATE todos
		SET status = 'deleted', deleted_at = $1, updated_at = $2
		WHERE id = $3 AND user_id = $4 AND status != 'deleted'
	`, now, now, id, userID)

	if err != nil {
		return nil, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}

	if rowsAffected == 0 {
		existing, err := s.GetTodo(ctx, id)
		if err == ErrNotFound {
			return nil, ErrNotFound
		}
		if err != nil {
			return nil, err
		}
		if existing.UserID != userID {
			return nil, ErrNotOwner
		}
		return existing, nil
	}

	return s.GetTodo(ctx, id)
}

// EditTodo updates a todo's title and/or description
func (s *Store) EditTodo(ctx context.Context, id int64, userID, title, description string) (*Todo, error) {
	result, err := s.db.ExecContext(ctx, `
		UPDATE todos
		SET title = $1, description = $2, updated_at = $3
		WHERE id = $4 AND user_id = $5 AND status != 'deleted'
	`, title, description, time.Now(), id, userID)

	if err != nil {
		return nil, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}

	if rowsAffected == 0 {
		existing, err := s.GetTodo(ctx, id)
		if err == ErrNotFound {
			return nil, ErrNotFound
		}
		if err != nil {
			return nil, err
		}
		if existing.UserID != userID {
			return nil, ErrNotOwner
		}
		return nil, ErrNotFound // deleted
	}

	return s.GetTodo(ctx, id)
}

// CheckIdempotencyKey checks if an operation was already performed
// Returns the cached response if found, nil if not found
func (s *Store) CheckIdempotencyKey(ctx context.Context, key string) ([]byte, error) {
	var response []byte
	err := s.db.QueryRowContext(ctx, `
		SELECT response FROM idempotency_keys WHERE key = $1
	`, key).Scan(&response)

	if err == sql.ErrNoRows {
		return nil, nil // Not found, operation can proceed
	}
	if err != nil {
		return nil, err
	}

	return response, nil
}

// StoreIdempotencyKey stores the result of an operation for idempotency
func (s *Store) StoreIdempotencyKey(ctx context.Context, key string, response interface{}) error {
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO idempotency_keys (key, response, created_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (key) DO NOTHING
	`, key, jsonResponse, time.Now())

	return err
}
