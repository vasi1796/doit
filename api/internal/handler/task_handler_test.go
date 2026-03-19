package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/vasi1796/doit/internal/auth"
	"github.com/vasi1796/doit/internal/domain"
)

type mockTaskCommander struct {
	err     error
	lastCmd any
}

func (m *mockTaskCommander) CreateTask(_ context.Context, cmd domain.CreateTask) error {
	m.lastCmd = cmd
	return m.err
}
func (m *mockTaskCommander) CompleteTask(_ context.Context, _ uuid.UUID, _ uuid.UUID, cmd domain.CompleteTask) error {
	m.lastCmd = cmd
	return m.err
}
func (m *mockTaskCommander) UncompleteTask(_ context.Context, _ uuid.UUID, _ uuid.UUID, cmd domain.UncompleteTask) error {
	m.lastCmd = cmd
	return m.err
}
func (m *mockTaskCommander) DeleteTask(_ context.Context, _ uuid.UUID, _ uuid.UUID, cmd domain.DeleteTask) error {
	m.lastCmd = cmd
	return m.err
}
func (m *mockTaskCommander) MoveTask(_ context.Context, _ uuid.UUID, _ uuid.UUID, cmd domain.MoveTask) error {
	m.lastCmd = cmd
	return m.err
}
func (m *mockTaskCommander) UpdateTaskDescription(_ context.Context, _ uuid.UUID, _ uuid.UUID, cmd domain.UpdateTaskDescription) error {
	m.lastCmd = cmd
	return m.err
}
func (m *mockTaskCommander) AddLabel(_ context.Context, _ uuid.UUID, _ uuid.UUID, cmd domain.AddLabel) error {
	m.lastCmd = cmd
	return m.err
}
func (m *mockTaskCommander) RemoveLabel(_ context.Context, _ uuid.UUID, _ uuid.UUID, cmd domain.RemoveLabel) error {
	m.lastCmd = cmd
	return m.err
}
func (m *mockTaskCommander) CreateSubtask(_ context.Context, _ uuid.UUID, _ uuid.UUID, cmd domain.CreateSubtask) error {
	m.lastCmd = cmd
	return m.err
}
func (m *mockTaskCommander) CompleteSubtask(_ context.Context, _ uuid.UUID, _ uuid.UUID, cmd domain.CompleteSubtask) error {
	m.lastCmd = cmd
	return m.err
}

func withUserContext(r *http.Request, userID uuid.UUID) *http.Request {
	ctx := auth.WithUserID(r.Context(), userID)
	return r.WithContext(ctx)
}

func withChiParam(r *http.Request, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func TestTaskHandlerCreate(t *testing.T) {
	userID := uuid.New()
	listID := uuid.New()

	tests := []struct {
		name       string
		body       string
		cmdErr     error
		hasUser    bool
		wantStatus int
	}{
		{
			name:       "success",
			body:       `{"title":"Buy milk","priority":1,"list_id":"` + listID.String() + `","position":"a"}`,
			hasUser:    true,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "validation error",
			body:       `{"title":"","priority":1,"list_id":"` + listID.String() + `","position":"a"}`,
			cmdErr:     domain.ErrEmptyTitle,
			hasUser:    true,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid JSON",
			body:       `{bad`,
			hasUser:    true,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "no auth",
			body:       `{"title":"Buy milk"}`,
			hasUser:    false,
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockTaskCommander{err: tc.cmdErr}
			h := NewTaskHandler(mock, nil, zerolog.Nop())

			req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", strings.NewReader(tc.body))
			if tc.hasUser {
				req = withUserContext(req, userID)
			}

			rr := httptest.NewRecorder()
			h.Create(rr, req)

			if rr.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", rr.Code, tc.wantStatus)
			}
		})
	}
}

func TestTaskHandlerComplete(t *testing.T) {
	userID := uuid.New()
	taskID := uuid.New()

	tests := []struct {
		name       string
		cmdErr     error
		wantStatus int
	}{
		{
			name:       "success",
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "already completed",
			cmdErr:     domain.ErrTaskAlreadyCompleted,
			wantStatus: http.StatusConflict,
		},
		{
			name:       "not found",
			cmdErr:     domain.ErrTaskNotFound,
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockTaskCommander{err: tc.cmdErr}
			h := NewTaskHandler(mock, nil, zerolog.Nop())

			req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/"+taskID.String()+"/complete", nil)
			req = withUserContext(req, userID)
			req = withChiParam(req, "id", taskID.String())

			rr := httptest.NewRecorder()
			h.Complete(rr, req)

			if rr.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", rr.Code, tc.wantStatus)
			}
		})
	}
}

func TestTaskHandlerDelete(t *testing.T) {
	taskID := uuid.New()
	userID := uuid.New()

	tests := []struct {
		name       string
		cmdErr     error
		wantStatus int
	}{
		{
			name:       "success",
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "already deleted",
			cmdErr:     domain.ErrTaskAlreadyDeleted,
			wantStatus: http.StatusConflict,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockTaskCommander{err: tc.cmdErr}
			h := NewTaskHandler(mock, nil, zerolog.Nop())

			req := httptest.NewRequest(http.MethodDelete, "/api/v1/tasks/"+taskID.String(), nil)
			req = withUserContext(req, userID)
			req = withChiParam(req, "id", taskID.String())

			rr := httptest.NewRecorder()
			h.Delete(rr, req)

			if rr.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", rr.Code, tc.wantStatus)
			}
		})
	}
}

func TestTaskHandlerInvalidUUID(t *testing.T) {
	mock := &mockTaskCommander{}
	h := NewTaskHandler(mock, nil, zerolog.Nop())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/not-a-uuid/complete", nil)
	req = withUserContext(req, uuid.New())
	req = withChiParam(req, "id", "not-a-uuid")

	rr := httptest.NewRecorder()
	h.Complete(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestTaskHandlerCreateSubtask(t *testing.T) {
	taskID := uuid.New()

	tests := []struct {
		name       string
		body       string
		cmdErr     error
		wantStatus int
	}{
		{
			name:       "success",
			body:       `{"title":"Sub item","position":"a"}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "empty title",
			body:       `{"title":"","position":"a"}`,
			cmdErr:     domain.ErrEmptyTitle,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockTaskCommander{err: tc.cmdErr}
			h := NewTaskHandler(mock, nil, zerolog.Nop())

			req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/"+taskID.String()+"/subtasks", strings.NewReader(tc.body))
			req = withUserContext(req, uuid.New())
			req = withChiParam(req, "id", taskID.String())

			rr := httptest.NewRecorder()
			h.CreateSubtask(rr, req)

			if rr.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", rr.Code, tc.wantStatus)
			}
		})
	}
}

