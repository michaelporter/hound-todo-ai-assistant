package server

import (
	"context"
	"encoding/json"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	todov1 "hound-todo/api/todo/v1"
	"hound-todo/services/todo-domain/internal/store"
	"hound-todo/shared/logging"
)

// Server implements the TodoDomain gRPC service
type Server struct {
	todov1.UnimplementedTodoDomainServer // Embed for forward compatibility
	store                                *store.Store
	logger                               *logging.Logger
}

// New creates a new gRPC server
func New(store *store.Store, logger *logging.Logger) *Server {
	return &Server{
		store:  store,
		logger: logger,
	}
}

// CreateTodo creates a new todo item
func (s *Server) CreateTodo(ctx context.Context, req *todov1.CreateTodoRequest) (*todov1.CreateTodoResponse, error) {
	// Validate required fields
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if req.Title == "" {
		return nil, status.Error(codes.InvalidArgument, "title is required")
	}

	// Check idempotency key
	if req.IdempotencyKey != "" {
		cached, err := s.store.CheckIdempotencyKey(ctx, req.IdempotencyKey)
		if err != nil {
			s.logger.Error("Failed to check idempotency key: %v", err)
			return nil, status.Error(codes.Internal, "internal error")
		}
		if cached != nil {
			var resp todov1.CreateTodoResponse
			if err := json.Unmarshal(cached, &resp); err == nil {
				s.logger.Info("Returning cached response for idempotency key %s", req.IdempotencyKey)
				return &resp, nil
			}
		}
	}

	// Create the todo
	todo, err := s.store.CreateTodo(ctx, req.UserId, req.Title, req.Description)
	if err != nil {
		s.logger.Error("Failed to create todo: %v", err)
		return nil, status.Error(codes.Internal, "failed to create todo")
	}

	resp := &todov1.CreateTodoResponse{
		Todo: storeToProto(todo),
	}

	// Store idempotency key
	if req.IdempotencyKey != "" {
		if err := s.store.StoreIdempotencyKey(ctx, req.IdempotencyKey, resp); err != nil {
			s.logger.Error("Failed to store idempotency key: %v", err)
			// Don't fail the request, the todo was created
		}
	}

	s.logger.Info("Created todo %d for user %s: %s", todo.ID, req.UserId, req.Title)
	return resp, nil
}

// CompleteTodo marks a todo as completed
func (s *Server) CompleteTodo(ctx context.Context, req *todov1.CompleteTodoRequest) (*todov1.CompleteTodoResponse, error) {
	if req.TodoId == 0 {
		return nil, status.Error(codes.InvalidArgument, "todo_id is required")
	}
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	// Check idempotency key
	if req.IdempotencyKey != "" {
		cached, err := s.store.CheckIdempotencyKey(ctx, req.IdempotencyKey)
		if err != nil {
			s.logger.Error("Failed to check idempotency key: %v", err)
			return nil, status.Error(codes.Internal, "internal error")
		}
		if cached != nil {
			var resp todov1.CompleteTodoResponse
			if err := json.Unmarshal(cached, &resp); err == nil {
				return &resp, nil
			}
		}
	}

	// Determine completion time
	completedAt := req.CompletedAt.AsTime()
	if req.CompletedAt == nil {
		completedAt = timestamppb.Now().AsTime()
	}

	todo, err := s.store.CompleteTodo(ctx, req.TodoId, req.UserId, completedAt)
	if err == store.ErrNotFound {
		return nil, status.Error(codes.NotFound, "todo not found")
	}
	if err == store.ErrNotOwner {
		return nil, status.Error(codes.PermissionDenied, "you do not own this todo")
	}
	if err != nil {
		s.logger.Error("Failed to complete todo: %v", err)
		return nil, status.Error(codes.Internal, "failed to complete todo")
	}

	resp := &todov1.CompleteTodoResponse{
		Todo: storeToProto(todo),
	}

	if req.IdempotencyKey != "" {
		s.store.StoreIdempotencyKey(ctx, req.IdempotencyKey, resp)
	}

	s.logger.Info("Completed todo %d for user %s", req.TodoId, req.UserId)
	return resp, nil
}

// ListTodos retrieves all todos for a user
func (s *Server) ListTodos(ctx context.Context, req *todov1.ListTodosRequest) (*todov1.ListTodosResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	// Convert proto status to store status string
	statusFilter := ""
	switch req.Status {
	case todov1.TodoStatus_TODO_STATUS_ACTIVE:
		statusFilter = "active"
	case todov1.TodoStatus_TODO_STATUS_COMPLETED:
		statusFilter = "completed"
	case todov1.TodoStatus_TODO_STATUS_DELETED:
		statusFilter = "deleted"
	}

	todos, err := s.store.ListTodos(ctx, req.UserId, statusFilter)
	if err != nil {
		s.logger.Error("Failed to list todos: %v", err)
		return nil, status.Error(codes.Internal, "failed to list todos")
	}

	protoTodos := make([]*todov1.Todo, len(todos))
	for i, todo := range todos {
		protoTodos[i] = storeToProto(todo)
	}

	return &todov1.ListTodosResponse{
		Todos: protoTodos,
	}, nil
}

// DeleteTodo soft-deletes a todo
func (s *Server) DeleteTodo(ctx context.Context, req *todov1.DeleteTodoRequest) (*todov1.DeleteTodoResponse, error) {
	if req.TodoId == 0 {
		return nil, status.Error(codes.InvalidArgument, "todo_id is required")
	}
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	// Check idempotency key
	if req.IdempotencyKey != "" {
		cached, err := s.store.CheckIdempotencyKey(ctx, req.IdempotencyKey)
		if err != nil {
			s.logger.Error("Failed to check idempotency key: %v", err)
			return nil, status.Error(codes.Internal, "internal error")
		}
		if cached != nil {
			var resp todov1.DeleteTodoResponse
			if err := json.Unmarshal(cached, &resp); err == nil {
				return &resp, nil
			}
		}
	}

	todo, err := s.store.DeleteTodo(ctx, req.TodoId, req.UserId)
	if err == store.ErrNotFound {
		return nil, status.Error(codes.NotFound, "todo not found")
	}
	if err == store.ErrNotOwner {
		return nil, status.Error(codes.PermissionDenied, "you do not own this todo")
	}
	if err != nil {
		s.logger.Error("Failed to delete todo: %v", err)
		return nil, status.Error(codes.Internal, "failed to delete todo")
	}

	resp := &todov1.DeleteTodoResponse{
		Todo: storeToProto(todo),
	}

	if req.IdempotencyKey != "" {
		s.store.StoreIdempotencyKey(ctx, req.IdempotencyKey, resp)
	}

	s.logger.Info("Deleted todo %d for user %s", req.TodoId, req.UserId)
	return resp, nil
}

// EditTodo updates a todo's title or description
func (s *Server) EditTodo(ctx context.Context, req *todov1.EditTodoRequest) (*todov1.EditTodoResponse, error) {
	if req.TodoId == 0 {
		return nil, status.Error(codes.InvalidArgument, "todo_id is required")
	}
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	// Check idempotency key
	if req.IdempotencyKey != "" {
		cached, err := s.store.CheckIdempotencyKey(ctx, req.IdempotencyKey)
		if err != nil {
			s.logger.Error("Failed to check idempotency key: %v", err)
			return nil, status.Error(codes.Internal, "internal error")
		}
		if cached != nil {
			var resp todov1.EditTodoResponse
			if err := json.Unmarshal(cached, &resp); err == nil {
				return &resp, nil
			}
		}
	}

	todo, err := s.store.EditTodo(ctx, req.TodoId, req.UserId, req.Title, req.Description)
	if err == store.ErrNotFound {
		return nil, status.Error(codes.NotFound, "todo not found")
	}
	if err == store.ErrNotOwner {
		return nil, status.Error(codes.PermissionDenied, "you do not own this todo")
	}
	if err != nil {
		s.logger.Error("Failed to edit todo: %v", err)
		return nil, status.Error(codes.Internal, "failed to edit todo")
	}

	resp := &todov1.EditTodoResponse{
		Todo: storeToProto(todo),
	}

	if req.IdempotencyKey != "" {
		s.store.StoreIdempotencyKey(ctx, req.IdempotencyKey, resp)
	}

	s.logger.Info("Edited todo %d for user %s", req.TodoId, req.UserId)
	return resp, nil
}

// storeToProto converts a store.Todo to a protobuf Todo
func storeToProto(t *store.Todo) *todov1.Todo {
	proto := &todov1.Todo{
		Id:          t.ID,
		UserId:      t.UserID,
		Title:       t.Title,
		Description: t.Description,
		CreatedAt:   timestamppb.New(t.CreatedAt),
	}

	// Map status string to proto enum
	switch t.Status {
	case "active":
		proto.Status = todov1.TodoStatus_TODO_STATUS_ACTIVE
	case "completed":
		proto.Status = todov1.TodoStatus_TODO_STATUS_COMPLETED
	case "deleted":
		proto.Status = todov1.TodoStatus_TODO_STATUS_DELETED
	default:
		proto.Status = todov1.TodoStatus_TODO_STATUS_UNSPECIFIED
	}

	if t.CompletedAt != nil {
		proto.CompletedAt = timestamppb.New(*t.CompletedAt)
	}

	return proto
}
