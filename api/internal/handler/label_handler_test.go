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

type mockLabelCommander struct {
	err error
}

func (m *mockLabelCommander) CreateLabel(_ context.Context, _ domain.CreateLabel) error {
	return m.err
}

func (m *mockLabelCommander) DeleteLabel(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ domain.DeleteLabel) error {
	return m.err
}

func TestLabelHandlerCreate(t *testing.T) {
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
			body:       `{"name":"urgent","colour":"#ff0000"}`,
			hasUser:    true,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "empty name",
			body:       `{"name":"","colour":"#ff0000"}`,
			cmdErr:     domain.ErrEmptyName,
			hasUser:    true,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "no auth",
			body:       `{"name":"urgent"}`,
			hasUser:    false,
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockLabelCommander{err: tc.cmdErr}
			h := NewLabelHandler(mock, nil, zerolog.Nop())

			req := httptest.NewRequest(http.MethodPost, "/api/v1/labels", strings.NewReader(tc.body))
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
