package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/vasi1796/doit/internal/domain"
)

type mockListCommander struct {
	err error
}

func (m *mockListCommander) CreateList(_ context.Context, _ domain.CreateList) error {
	return m.err
}

func TestListHandlerCreate(t *testing.T) {
	userID := uuid.New()

	tests := []struct {
		name       string
		body       string
		cmdErr     error
		hasUser    bool
		wantStatus int
	}{
		{
			name:       "success",
			body:       `{"name":"Work","colour":"#ff0000","position":"a"}`,
			hasUser:    true,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "empty name",
			body:       `{"name":"","colour":"#ff0000","position":"a"}`,
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
			body:       `{"name":"Work"}`,
			hasUser:    false,
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockListCommander{err: tc.cmdErr}
			h := NewListHandler(mock, nil, zerolog.Nop())

			req := httptest.NewRequest(http.MethodPost, "/api/v1/lists", strings.NewReader(tc.body))
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
