package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

func TestPushHandlerGetVAPIDKey(t *testing.T) {
	tests := []struct {
		name       string
		publicKey  string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "returns key when configured",
			publicKey:  "BTestKey123",
			wantStatus: http.StatusOK,
			wantBody:   `"vapid_public_key":"BTestKey123"`,
		},
		{
			name:       "returns 503 when not configured",
			publicKey:  "",
			wantStatus: http.StatusServiceUnavailable,
			wantBody:   "not configured",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := NewPushHandler(nil, tc.publicKey, "", "", zerolog.Nop())

			req := httptest.NewRequest(http.MethodGet, "/api/v1/push/vapid-key", nil)
			rr := httptest.NewRecorder()
			h.GetVAPIDKey(rr, req)

			if rr.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", rr.Code, tc.wantStatus)
			}
			if !strings.Contains(rr.Body.String(), tc.wantBody) {
				t.Errorf("body = %q, want to contain %q", rr.Body.String(), tc.wantBody)
			}
		})
	}
}

func TestPushHandlerSubscribe(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		hasUser    bool
		wantStatus int
	}{
		{
			name:       "no auth",
			body:       `{"endpoint":"https://push.example.com","keys":{"p256dh":"abc","auth":"def"}}`,
			hasUser:    false,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "missing endpoint",
			body:       `{"endpoint":"","keys":{"p256dh":"abc","auth":"def"}}`,
			hasUser:    true,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing p256dh",
			body:       `{"endpoint":"https://push.example.com","keys":{"p256dh":"","auth":"def"}}`,
			hasUser:    true,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing auth key",
			body:       `{"endpoint":"https://push.example.com","keys":{"p256dh":"abc","auth":""}}`,
			hasUser:    true,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := NewPushHandler(nil, "pubkey", "privkey", "mailto:test@test.com", zerolog.Nop())

			req := httptest.NewRequest(http.MethodPost, "/api/v1/push/subscribe", strings.NewReader(tc.body))
			if tc.hasUser {
				req = withUserContext(req, uuid.New())
			}

			rr := httptest.NewRecorder()
			h.Subscribe(rr, req)

			if rr.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", rr.Code, tc.wantStatus)
			}
		})
	}
}

func TestPushHandlerUnsubscribe(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		hasUser    bool
		wantStatus int
	}{
		{
			name:       "no auth",
			body:       `{"endpoint":"https://push.example.com"}`,
			hasUser:    false,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "missing endpoint",
			body:       `{"endpoint":""}`,
			hasUser:    true,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := NewPushHandler(nil, "pubkey", "privkey", "mailto:test@test.com", zerolog.Nop())

			req := httptest.NewRequest(http.MethodDelete, "/api/v1/push/subscribe", strings.NewReader(tc.body))
			if tc.hasUser {
				req = withUserContext(req, uuid.New())
			}

			rr := httptest.NewRecorder()
			h.Unsubscribe(rr, req)

			if rr.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", rr.Code, tc.wantStatus)
			}
		})
	}
}

func TestPushHandlerTestEndpoint(t *testing.T) {
	tests := []struct {
		name       string
		hasUser    bool
		hasVAPID   bool
		wantStatus int
	}{
		{
			name:       "no auth",
			hasUser:    false,
			hasVAPID:   true,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "VAPID not configured",
			hasUser:    true,
			hasVAPID:   false,
			wantStatus: http.StatusServiceUnavailable,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pubKey, privKey := "", ""
			if tc.hasVAPID {
				pubKey = "pubkey"
				privKey = "privkey"
			}
			h := NewPushHandler(nil, pubKey, privKey, "mailto:test@test.com", zerolog.Nop())

			req := httptest.NewRequest(http.MethodPost, "/api/v1/push/test", nil)
			if tc.hasUser {
				req = withUserContext(req, uuid.New())
			}

			rr := httptest.NewRecorder()
			h.Test(rr, req)

			if rr.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", rr.Code, tc.wantStatus)
			}
		})
	}
}
