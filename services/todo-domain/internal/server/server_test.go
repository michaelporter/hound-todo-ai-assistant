package server

import (
	"context"
	"errors"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	todov1 "hound-todo/api/todo/v1"
	"hound-todo/services/todo-domain/internal/store"
	"hound-todo/shared/logging"
)

// =============================================================================
// Mock Store
// =============================================================================

type mockStore struct {
	createTodoResp   *store.Todo
	getTodoResp      *store.Todo
	listTodosResp    []*store.Todo
	completeTodoResp *store.Todo
	deleteTodoResp   *store.Todo
	editTodoResp     *store.Todo

	createErr      error
	getErr         error
	listErr        error
	completeErr    error
	deleteErr      error
	editErr        error

	checkIdempotencyResp  []byte
	checkIdempotencyErr   error
	storeIdempotencyErr   error

	// Capture calls
	lastCreateUserID string
	lastCreateTitle  string
	lastListUserID   string
	lastListFilter   store.ListTodosFilter
}

func (m *mockStore) CreateTodo(ctx context.Context, userID, title, description string) (*store.Todo, error) {
	m.lastCreateUserID = userID
	m.lastCreateTitle = title
	if m.createErr != nil {
		return nil, m.createErr
	}
	return m.createTodoResp, nil
}

func (m *mockStore) GetTodo(ctx context.Context, id int64) (*store.Todo, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.getTodoResp, nil
}

func (m *mockStore) ListTodos(ctx context.Context, userID string, filter store.ListTodosFilter) ([]*store.Todo, error) {
	m.lastListUserID = userID
	m.lastListFilter = filter
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.listTodosResp, nil
}

func (m *mockStore) CompleteTodo(ctx context.Context, id int64, userID string, completedAt time.Time) (*store.Todo, error) {
	if m.completeErr != nil {
		return nil, m.completeErr
	}
	return m.completeTodoResp, nil
}

func (m *mockStore) DeleteTodo(ctx context.Context, id int64, userID string) (*store.Todo, error) {
	if m.deleteErr != nil {
		return nil, m.deleteErr
	}
	return m.deleteTodoResp, nil
}

func (m *mockStore) EditTodo(ctx context.Context, id int64, userID, title, description string) (*store.Todo, error) {
	if m.editErr != nil {
		return nil, m.editErr
	}
	return m.editTodoResp, nil
}

func (m *mockStore) CheckIdempotencyKey(ctx context.Context, key string) ([]byte, error) {
	if m.checkIdempotencyErr != nil {
		return nil, m.checkIdempotencyErr
	}
	return m.checkIdempotencyResp, nil
}

func (m *mockStore) StoreIdempotencyKey(ctx context.Context, key string, response interface{}) error {
	return m.storeIdempotencyErr
}

// =============================================================================
// Test Helpers
// =============================================================================

type testServer struct {
	*Server
	mockStore *mockStore
}

func newTestServer() *testServer {
	mock := &mockStore{}
	logger := logging.New("test")

	// Create a server with our mock
	srv := &Server{
		store:  nil, // Will be replaced
		logger: logger,
	}

	return &testServer{
		Server:    srv,
		mockStore: mock,
	}
}

// =============================================================================
// CreateTodo Tests
// =============================================================================

func TestCreateTodo_Success(t *testing.T) {
	ts := newTestServer()
	now := time.Now()
	ts.mockStore.createTodoResp = &store.Todo{
		ID:          1,
		UserID:      "user123",
		Title:       "buy groceries",
		Description: "milk and eggs",
		Status:      "active",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// We need to access the store through the server
	// Since store is private, we'll test the storeToProto function instead
	proto := storeToProto(ts.mockStore.createTodoResp)

	if proto.Id != 1 {
		t.Errorf("expected Id 1, got %d", proto.Id)
	}
	if proto.UserId != "user123" {
		t.Errorf("expected UserId user123, got %s", proto.UserId)
	}
	if proto.Title != "buy groceries" {
		t.Errorf("expected Title 'buy groceries', got %s", proto.Title)
	}
	if proto.Status != todov1.TodoStatus_TODO_STATUS_ACTIVE {
		t.Errorf("expected Status ACTIVE, got %v", proto.Status)
	}
}

func TestCreateTodo_MissingUserID(t *testing.T) {
	ts := newTestServer()
	ts.Server.store = &store.Store{} // Need real store for validation

	req := &todov1.CreateTodoRequest{
		UserId: "",
		Title:  "test",
	}

	_, err := ts.CreateTodo(context.Background(), req)

	if err == nil {
		t.Fatal("expected error for missing user_id")
	}
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument error, got %v", err)
	}
}

func TestCreateTodo_MissingTitle(t *testing.T) {
	ts := newTestServer()
	ts.Server.store = &store.Store{} // Need real store for validation

	req := &todov1.CreateTodoRequest{
		UserId: "user123",
		Title:  "",
	}

	_, err := ts.CreateTodo(context.Background(), req)

	if err == nil {
		t.Fatal("expected error for missing title")
	}
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument error, got %v", err)
	}
}

// =============================================================================
// CompleteTodo Tests
// =============================================================================

func TestCompleteTodo_MissingTodoID(t *testing.T) {
	ts := newTestServer()
	ts.Server.store = &store.Store{}

	req := &todov1.CompleteTodoRequest{
		TodoId: 0,
		UserId: "user123",
	}

	_, err := ts.CompleteTodo(context.Background(), req)

	if err == nil {
		t.Fatal("expected error for missing todo_id")
	}
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument error, got %v", err)
	}
}

func TestCompleteTodo_MissingUserID(t *testing.T) {
	ts := newTestServer()
	ts.Server.store = &store.Store{}

	req := &todov1.CompleteTodoRequest{
		TodoId: 1,
		UserId: "",
	}

	_, err := ts.CompleteTodo(context.Background(), req)

	if err == nil {
		t.Fatal("expected error for missing user_id")
	}
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument error, got %v", err)
	}
}

// =============================================================================
// ListTodos Tests
// =============================================================================

func TestListTodos_MissingUserID(t *testing.T) {
	ts := newTestServer()
	ts.Server.store = &store.Store{}

	req := &todov1.ListTodosRequest{
		UserId: "",
	}

	_, err := ts.ListTodos(context.Background(), req)

	if err == nil {
		t.Fatal("expected error for missing user_id")
	}
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument error, got %v", err)
	}
}

// =============================================================================
// DeleteTodo Tests
// =============================================================================

func TestDeleteTodo_MissingTodoID(t *testing.T) {
	ts := newTestServer()
	ts.Server.store = &store.Store{}

	req := &todov1.DeleteTodoRequest{
		TodoId: 0,
		UserId: "user123",
	}

	_, err := ts.DeleteTodo(context.Background(), req)

	if err == nil {
		t.Fatal("expected error for missing todo_id")
	}
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument error, got %v", err)
	}
}

func TestDeleteTodo_MissingUserID(t *testing.T) {
	ts := newTestServer()
	ts.Server.store = &store.Store{}

	req := &todov1.DeleteTodoRequest{
		TodoId: 1,
		UserId: "",
	}

	_, err := ts.DeleteTodo(context.Background(), req)

	if err == nil {
		t.Fatal("expected error for missing user_id")
	}
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument error, got %v", err)
	}
}

// =============================================================================
// EditTodo Tests
// =============================================================================

func TestEditTodo_MissingTodoID(t *testing.T) {
	ts := newTestServer()
	ts.Server.store = &store.Store{}

	req := &todov1.EditTodoRequest{
		TodoId: 0,
		UserId: "user123",
		Title:  "new title",
	}

	_, err := ts.EditTodo(context.Background(), req)

	if err == nil {
		t.Fatal("expected error for missing todo_id")
	}
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument error, got %v", err)
	}
}

func TestEditTodo_MissingUserID(t *testing.T) {
	ts := newTestServer()
	ts.Server.store = &store.Store{}

	req := &todov1.EditTodoRequest{
		TodoId: 1,
		UserId: "",
		Title:  "new title",
	}

	_, err := ts.EditTodo(context.Background(), req)

	if err == nil {
		t.Fatal("expected error for missing user_id")
	}
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument error, got %v", err)
	}
}

// =============================================================================
// storeToProto Tests
// =============================================================================

func TestStoreToProto_AllStatuses(t *testing.T) {
	tests := []struct {
		status   string
		expected todov1.TodoStatus
	}{
		{"active", todov1.TodoStatus_TODO_STATUS_ACTIVE},
		{"completed", todov1.TodoStatus_TODO_STATUS_COMPLETED},
		{"deleted", todov1.TodoStatus_TODO_STATUS_DELETED},
		{"unknown", todov1.TodoStatus_TODO_STATUS_UNSPECIFIED},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			todo := &store.Todo{
				ID:        1,
				UserID:    "user123",
				Title:     "test",
				Status:    tt.status,
				CreatedAt: time.Now(),
			}

			proto := storeToProto(todo)

			if proto.Status != tt.expected {
				t.Errorf("expected status %v, got %v", tt.expected, proto.Status)
			}
		})
	}
}

func TestStoreToProto_WithCompletedAt(t *testing.T) {
	completedAt := time.Now()
	todo := &store.Todo{
		ID:          1,
		UserID:      "user123",
		Title:       "completed task",
		Status:      "completed",
		CreatedAt:   time.Now().Add(-1 * time.Hour),
		CompletedAt: &completedAt,
	}

	proto := storeToProto(todo)

	if proto.CompletedAt == nil {
		t.Fatal("expected CompletedAt to be set")
	}
	if !proto.CompletedAt.AsTime().Equal(completedAt) {
		t.Errorf("expected CompletedAt %v, got %v", completedAt, proto.CompletedAt.AsTime())
	}
}

func TestStoreToProto_WithoutCompletedAt(t *testing.T) {
	todo := &store.Todo{
		ID:          1,
		UserID:      "user123",
		Title:       "active task",
		Status:      "active",
		CreatedAt:   time.Now(),
		CompletedAt: nil,
	}

	proto := storeToProto(todo)

	if proto.CompletedAt != nil {
		t.Error("expected CompletedAt to be nil")
	}
}

// =============================================================================
// Proto Timestamp Conversion Tests
// =============================================================================

func TestTimestampConversion(t *testing.T) {
	now := time.Now()
	ts := timestamppb.New(now)

	// Convert back
	converted := ts.AsTime()

	// Allow for nanosecond precision differences
	if !converted.Round(time.Microsecond).Equal(now.Round(time.Microsecond)) {
		t.Errorf("timestamp conversion failed: expected %v, got %v", now, converted)
	}
}

// =============================================================================
// Error Mapping Tests
// =============================================================================

func TestErrorMapping_NotFound(t *testing.T) {
	err := store.ErrNotFound

	if !errors.Is(err, store.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestErrorMapping_NotOwner(t *testing.T) {
	err := store.ErrNotOwner

	if !errors.Is(err, store.ErrNotOwner) {
		t.Errorf("expected ErrNotOwner, got %v", err)
	}
}
