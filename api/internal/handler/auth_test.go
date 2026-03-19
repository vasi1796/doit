package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rs/zerolog"

	"github.com/vasi1796/doit/internal/auth"
)

type mockOAuthExchanger struct {
	authURL  string
	user     *auth.GoogleUser
	exchErr  error
}

func (m *mockOAuthExchanger) AuthURL(state string) string {
	return m.authURL + "?state=" + state
}

func (m *mockOAuthExchanger) Exchange(_ context.Context, _ string) (*auth.GoogleUser, error) {
	return m.user, m.exchErr
}

func TestGoogleLoginRedirects(t *testing.T) {
	h := NewAuthHandler(
		&mockOAuthExchanger{authURL: "https://accounts.google.com/o/oauth2/auth"},
		auth.NewTokenService("secret", 72),
		nil, // pool not needed for login redirect
		nil,
		zerolog.Nop(),
		"/",
		false,
		false,
	)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/auth/google/login", nil)
	h.GoogleLogin(rr, req)

	if rr.Code != http.StatusFound {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusFound)
	}

	location := rr.Header().Get("Location")
	if !strings.HasPrefix(location, "https://accounts.google.com") {
		t.Errorf("redirect location = %q, want google URL", location)
	}

	// Check state cookie was set
	cookies := rr.Result().Cookies()
	found := false
	for _, c := range cookies {
		if c.Name == "oauth_state" {
			found = true
			if c.Value == "" {
				t.Error("oauth_state cookie is empty")
			}
		}
	}
	if !found {
		t.Error("oauth_state cookie not set")
	}
}

func TestGoogleCallbackStateMismatch(t *testing.T) {
	h := NewAuthHandler(
		&mockOAuthExchanger{},
		auth.NewTokenService("secret", 72),
		nil,
		nil,
		zerolog.Nop(),
		"/",
		false,
		false,
	)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/auth/google/callback?state=abc&code=xyz", nil)
	req.AddCookie(&http.Cookie{Name: "oauth_state", Value: "different"})
	h.GoogleCallback(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusForbidden)
	}
}

func TestGoogleCallbackEmailNotAllowed(t *testing.T) {
	h := NewAuthHandler(
		&mockOAuthExchanger{
			user: &auth.GoogleUser{ID: "123", Email: "notallowed@example.com", Name: "Bad User"},
		},
		auth.NewTokenService("secret", 72),
		nil,
		[]string{"allowed@example.com"},
		zerolog.Nop(),
		"/",
		false,
		false,
	)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/auth/google/callback?state=abc&code=xyz", nil)
	req.AddCookie(&http.Cookie{Name: "oauth_state", Value: "abc"})
	h.GoogleCallback(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusForbidden)
	}
}

func TestDevLoginDisabled(t *testing.T) {
	h := NewAuthHandler(
		&mockOAuthExchanger{},
		auth.NewTokenService("secret", 72),
		nil,
		nil,
		zerolog.Nop(),
		"/",
		false, // devMode off
		false,
	)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth/dev", nil)
	h.DevLogin(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestLogout(t *testing.T) {
	h := NewAuthHandler(
		&mockOAuthExchanger{},
		auth.NewTokenService("secret", 72),
		nil,
		nil,
		zerolog.Nop(),
		"/",
		false,
		false,
	)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	h.Logout(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	// Check cookie was cleared
	cookies := rr.Result().Cookies()
	found := false
	for _, c := range cookies {
		if c.Name == "doit_token" && c.MaxAge < 0 {
			found = true
		}
	}
	if !found {
		t.Error("doit_token cookie not cleared")
	}
}
