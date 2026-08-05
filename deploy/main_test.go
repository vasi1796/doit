package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func sign(key, body string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(body))
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func TestHandleWebhook(t *testing.T) {
	testCases := []struct {
		name     string
		secret   string
		method   string
		body     string
		sig      string
		expected int
	}{
		{
			name:     "blank secret disables webhook instead of crashing",
			secret:   "",
			method:   http.MethodPost,
			body:     `{"ref":"refs/heads/main"}`,
			sig:      sign("", `{"ref":"refs/heads/main"}`),
			expected: http.StatusServiceUnavailable,
		},
		{
			name:     "invalid signature rejected",
			secret:   "test-key",
			method:   http.MethodPost,
			body:     `{"ref":"refs/heads/main"}`,
			sig:      sign("wrong-key", `{"ref":"refs/heads/main"}`),
			expected: http.StatusForbidden,
		},
		{
			name:     "missing signature rejected",
			secret:   "test-key",
			method:   http.MethodPost,
			body:     `{"ref":"refs/heads/main"}`,
			sig:      "",
			expected: http.StatusForbidden,
		},
		{
			name:     "valid signature on non-main ref skipped",
			secret:   "test-key",
			method:   http.MethodPost,
			body:     `{"ref":"refs/heads/feature"}`,
			sig:      sign("test-key", `{"ref":"refs/heads/feature"}`),
			expected: http.StatusOK,
		},
		{
			name:     "non-POST rejected",
			secret:   "test-key",
			method:   http.MethodGet,
			body:     "",
			sig:      "",
			expected: http.StatusMethodNotAllowed,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			secret = tc.secret
			req := httptest.NewRequest(tc.method, "/deploy/webhook", strings.NewReader(tc.body))
			if tc.sig != "" {
				req.Header.Set("X-Hub-Signature-256", tc.sig)
			}
			rec := httptest.NewRecorder()

			handleWebhook(rec, req)

			if rec.Code != tc.expected {
				t.Errorf("status = %d, expected %d (body: %s)", rec.Code, tc.expected, rec.Body.String())
			}
		})
	}
}
