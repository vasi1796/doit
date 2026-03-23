package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"os/exec"
	"strings"

	"github.com/rs/zerolog"
)

// DeployHandler handles GitHub webhook-triggered deployments.
type DeployHandler struct {
	secret string
	logger zerolog.Logger
}

func NewDeployHandler(secret string, logger zerolog.Logger) *DeployHandler {
	return &DeployHandler{secret: secret, logger: logger}
}

// Webhook handles POST /deploy/webhook — verifies GitHub HMAC signature,
// checks the push is to main, and triggers a background deploy.
func (h *DeployHandler) Webhook(w http.ResponseWriter, r *http.Request) {
	if h.secret == "" {
		http.Error(w, "webhook not configured", http.StatusServiceUnavailable)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1MB limit
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}

	// Verify HMAC-SHA256 signature
	sigHeader := r.Header.Get("X-Hub-Signature-256")
	if !h.verifySignature(body, sigHeader) {
		h.logger.Warn().Str("remote", r.RemoteAddr).Msg("deploy: invalid webhook signature")
		http.Error(w, "invalid signature", http.StatusForbidden)
		return
	}

	// Parse push event — only deploy on pushes to main
	var payload struct {
		Ref string `json:"ref"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	if payload.Ref != "refs/heads/main" {
		w.WriteHeader(http.StatusOK)
		writeJSON(w, h.logger, http.StatusOK, map[string]string{"status": "skipped", "reason": "not main branch"})
		return
	}

	// Trigger deploy in background — return immediately
	h.logger.Info().Msg("deploy: webhook received, starting deploy")
	go h.runDeploy()

	writeJSON(w, h.logger, http.StatusOK, map[string]string{"status": "deploying"})
}

func (h *DeployHandler) runDeploy() {
	cmd := exec.Command("sh", "-c", "git pull && docker compose up -d --build")
	output, err := cmd.CombinedOutput()
	if err != nil {
		h.logger.Error().Err(err).Str("output", string(output)).Msg("deploy: failed")
		return
	}
	h.logger.Info().Str("output", string(output)).Msg("deploy: completed successfully")
}

func (h *DeployHandler) verifySignature(body []byte, sigHeader string) bool {
	if sigHeader == "" {
		return false
	}

	// Expected format: "sha256=<hex>"
	prefix := "sha256="
	if !strings.HasPrefix(sigHeader, prefix) {
		return false
	}
	sig, err := hex.DecodeString(sigHeader[len(prefix):])
	if err != nil {
		return false
	}

	mac := hmac.New(sha256.New, []byte(h.secret))
	mac.Write(body)
	expected := mac.Sum(nil)

	return hmac.Equal(sig, expected)
}
