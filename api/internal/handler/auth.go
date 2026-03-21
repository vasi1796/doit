package handler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/vasi1796/doit/internal/auth"
)

// OAuthExchanger is the interface the handler needs for Google OAuth.
// Defined here (consumer-side) per project conventions.
type OAuthExchanger interface {
	AuthURL(state string) string
	Exchange(ctx context.Context, code string) (*auth.GoogleUser, error)
}

const upsertUserByGoogleIDSQL = `INSERT INTO users (id, google_id, email, name, avatar_url, allowed, created_at)
VALUES ($1, $2, $3, $4, $5, true, NOW())
ON CONFLICT (google_id) DO UPDATE SET
	email = EXCLUDED.email, name = EXCLUDED.name, avatar_url = EXCLUDED.avatar_url
RETURNING id`

const upsertUserByEmailSQL = `UPDATE users SET google_id = $1, name = $2, avatar_url = $3
WHERE email = $4
RETURNING id`

// AuthHandler holds dependencies for authentication endpoints.
type AuthHandler struct {
	google        OAuthExchanger
	tokens        *auth.TokenService
	pool          *pgxpool.Pool
	allowedEmails map[string]bool
	logger        zerolog.Logger
	frontendURL   string
	devMode       bool
	secureCookies bool
}

func NewAuthHandler(
	google OAuthExchanger,
	tokens *auth.TokenService,
	pool *pgxpool.Pool,
	allowedEmails []string,
	logger zerolog.Logger,
	frontendURL string,
	devMode bool,
	secureCookies bool,
) *AuthHandler {
	emailSet := make(map[string]bool, len(allowedEmails))
	for _, e := range allowedEmails {
		emailSet[strings.ToLower(strings.TrimSpace(e))] = true
	}
	return &AuthHandler{
		google:        google,
		tokens:        tokens,
		pool:          pool,
		allowedEmails: emailSet,
		logger:        logger,
		frontendURL:   frontendURL,
		devMode:       devMode,
		secureCookies: secureCookies,
	}
}

// GoogleLogin redirects the user to Google's consent screen.
func (h *AuthHandler) GoogleLogin(w http.ResponseWriter, r *http.Request) {
	stateBytes := make([]byte, 16)
	if _, err := rand.Read(stateBytes); err != nil {
		h.logger.Error().Err(err).Msg("generating oauth state")
		writeError(w, h.logger, http.StatusInternalServerError, "internal error")
		return
	}
	state := hex.EncodeToString(stateBytes)

	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		MaxAge:   300, // 5 minutes
		HttpOnly: true,
		Secure:   h.secureCookies,
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, h.google.AuthURL(state), http.StatusFound)
}

// GoogleCallback handles the OAuth callback from Google.
func (h *AuthHandler) GoogleCallback(w http.ResponseWriter, r *http.Request) {
	// Validate state
	stateCookie, err := r.Cookie("oauth_state")

	// Clear state cookie immediately after reading, regardless of outcome
	http.SetCookie(w, &http.Cookie{
		Name:   "oauth_state",
		Path:   "/",
		MaxAge: -1,
	})

	if err != nil || stateCookie.Value != r.URL.Query().Get("state") {
		h.logger.Debug().Msg("oauth state mismatch")
		writeError(w, h.logger, http.StatusForbidden, "invalid oauth state")
		return
	}

	// Check for OAuth error
	if errParam := r.URL.Query().Get("error"); errParam != "" {
		h.logger.Warn().Str("error", errParam).Msg("oauth error from google")
		writeError(w, h.logger, http.StatusForbidden, "authentication failed")
		return
	}

	// Exchange code for user info
	code := r.URL.Query().Get("code")
	googleUser, err := h.google.Exchange(r.Context(), code)
	if err != nil {
		h.logger.Error().Err(err).Msg("exchanging oauth code")
		writeError(w, h.logger, http.StatusInternalServerError, "authentication failed")
		return
	}

	// Check allowlist
	if !h.isAllowed(googleUser.Email) {
		h.logger.Warn().Str("email", googleUser.Email).Msg("email not in allowlist")
		writeError(w, h.logger, http.StatusForbidden, "email not allowed")
		return
	}

	// Upsert user
	userID, err := h.upsertUser(r.Context(), googleUser)
	if err != nil {
		h.logger.Error().Err(err).Msg("upserting user")
		writeError(w, h.logger, http.StatusInternalServerError, "internal error")
		return
	}

	// Issue JWT
	if !h.issueTokenCookie(w, userID, googleUser.Email) {
		return
	}

	h.logger.Info().Str("email", googleUser.Email).Msg("user logged in")
	http.Redirect(w, r, h.frontendURL, http.StatusFound)
}

// DevLogin creates a test user and issues a JWT without Google OAuth.
// Only available when DEV_MODE=true.
func (h *AuthHandler) DevLogin(w http.ResponseWriter, r *http.Request) {
	if !h.devMode {
		http.NotFound(w, r)
		return
	}

	email := "dev@test.com"
	name := "Dev User"

	// Allow overriding via JSON body
	var body struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if r.Body != nil && r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, h.logger, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if body.Email != "" {
			email = body.Email
		}
		if body.Name != "" {
			name = body.Name
		}
	}

	// Check allowlist (if no allowlist is configured, all emails are allowed)
	if !h.isAllowed(email) {
		h.logger.Warn().Str("email", email).Msg("dev login email not in allowlist")
		writeError(w, h.logger, http.StatusForbidden, "email not allowed")
		return
	}

	googleUser := &auth.GoogleUser{
		ID:    "dev-" + email,
		Email: email,
		Name:  name,
	}

	userID, err := h.upsertUser(r.Context(), googleUser)
	if err != nil {
		h.logger.Error().Err(err).Msg("upserting dev user")
		writeError(w, h.logger, http.StatusInternalServerError, "internal error")
		return
	}

	if !h.issueTokenCookie(w, userID, email) {
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{
		"user_id": userID.String(),
		"email":   email,
	}); err != nil {
		h.logger.Error().Err(err).Msg("encoding dev login response")
	}
}

// Logout clears the auth cookie.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "doit_token",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.secureCookies,
		SameSite: http.SameSiteLaxMode,
	})

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "logged out"}); err != nil {
		h.logger.Error().Err(err).Msg("encoding logout response")
	}
}

func (h *AuthHandler) isAllowed(email string) bool {
	if len(h.allowedEmails) == 0 {
		return true // no allowlist = allow all
	}
	return h.allowedEmails[strings.ToLower(email)]
}

func (h *AuthHandler) upsertUser(ctx context.Context, user *auth.GoogleUser) (uuid.UUID, error) {
	var userID uuid.UUID
	err := h.pool.QueryRow(ctx, upsertUserByGoogleIDSQL,
		uuid.New(), user.ID, user.Email, user.Name, user.AvatarURL,
	).Scan(&userID)
	if err == nil {
		return userID, nil
	}

	// If the email already exists under a different google_id (e.g., from dev login),
	// update the existing row to use the real google_id.
	err = h.pool.QueryRow(ctx, upsertUserByEmailSQL,
		user.ID, user.Name, user.AvatarURL, user.Email,
	).Scan(&userID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("upserting user: %w", err)
	}
	return userID, nil
}

func (h *AuthHandler) issueTokenCookie(w http.ResponseWriter, userID uuid.UUID, email string) bool {
	tokenStr, expiry, err := h.tokens.Issue(userID, email)
	if err != nil {
		h.logger.Error().Err(err).Msg("issuing JWT")
		writeError(w, h.logger, http.StatusInternalServerError, "internal error")
		return false
	}

	maxAge := int(time.Until(expiry).Seconds())
	http.SetCookie(w, &http.Cookie{
		Name:     "doit_token",
		Value:    tokenStr,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   h.secureCookies,
		SameSite: http.SameSiteLaxMode,
	})
	return true
}
