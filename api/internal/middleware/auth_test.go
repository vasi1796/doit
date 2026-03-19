package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/vasi1796/doit/internal/auth"
)

type mockTokenValidator struct {
	claims *auth.Claims
	err    error
}

func (m *mockTokenValidator) Validate(_ string) (*auth.Claims, error) {
	return m.claims, m.err
}

func TestJWTAuth(t *testing.T) {
	logger := zerolog.Nop()
	userID := uuid.New()

	tests := []struct {
		name       string
		cookie     *http.Cookie
		validator  *mockTokenValidator
		wantStatus int
		wantUserID bool
	}{
		{
			name:   "valid cookie",
			cookie: &http.Cookie{Name: "doit_token", Value: "valid-token"},
			validator: &mockTokenValidator{
				claims: &auth.Claims{UserID: userID, Email: "test@example.com"},
			},
			wantStatus: http.StatusOK,
			wantUserID: true,
		},
		{
			name:       "missing cookie",
			cookie:     nil,
			validator:  &mockTokenValidator{},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:   "invalid token",
			cookie: &http.Cookie{Name: "doit_token", Value: "bad-token"},
			validator: &mockTokenValidator{
				err: auth.ErrInvalidToken,
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:   "expired token",
			cookie: &http.Cookie{Name: "doit_token", Value: "expired-token"},
			validator: &mockTokenValidator{
				err: auth.ErrExpiredToken,
			},
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var gotUserID uuid.UUID
			var gotUser bool

			// Handler that checks for user ID in context
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotUserID, gotUser = auth.UserIDFromContext(r.Context())
				w.WriteHeader(http.StatusOK)
			})

			mw := JWTAuth(tc.validator, logger)
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks", nil)
			if tc.cookie != nil {
				req.AddCookie(tc.cookie)
			}

			mw(handler).ServeHTTP(rr, req)

			if rr.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", rr.Code, tc.wantStatus)
			}
			if tc.wantUserID {
				if !gotUser {
					t.Error("expected user ID in context, got none")
				}
				if gotUserID != userID {
					t.Errorf("user ID = %v, want %v", gotUserID, userID)
				}
			}
		})
	}
}
