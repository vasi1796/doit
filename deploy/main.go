// Standalone deploy webhook sidecar.
// Verifies GitHub HMAC-SHA256, deploys on pushes to main.
// Runs outside the app containers with Docker socket access.
package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
)

var (
	secret  string
	repoDir string
	mu      sync.Mutex // prevent concurrent deploys
)

func main() {
	secret = os.Getenv("DEPLOY_WEBHOOK_SECRET")
	if secret == "" {
		log.Fatal("DEPLOY_WEBHOOK_SECRET is required")
	}

	repoDir = os.Getenv("REPO_DIR")
	if repoDir == "" {
		repoDir = "/repo"
	}

	port := os.Getenv("WEBHOOK_PORT")
	if port == "" {
		port = "9000"
	}

	http.HandleFunc("/deploy/webhook", handleWebhook)
	http.HandleFunc("/deploy/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	log.Printf("deploy webhook listening on :%s (repo: %s)", port, repoDir)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func handleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}

	// Verify HMAC-SHA256
	sigHeader := r.Header.Get("X-Hub-Signature-256")
	if !verifySignature(body, sigHeader) {
		log.Printf("invalid webhook signature from %s", r.RemoteAddr)
		http.Error(w, "invalid signature", http.StatusForbidden)
		return
	}

	// Only deploy on pushes to main
	var payload struct {
		Ref string `json:"ref"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	if payload.Ref != "refs/heads/main" {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"skipped","reason":"not main branch"}`)
		return
	}

	// Prevent concurrent deploys
	if !mu.TryLock() {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		fmt.Fprintf(w, `{"status":"skipped","reason":"deploy already in progress"}`)
		return
	}

	log.Println("main branch push detected, starting deploy")
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"status":"deploying"}`)

	// Run deploy async — unlock when done
	go func() {
		defer mu.Unlock()
		runDeploy()
	}()
}

func runDeploy() {
	composeFile := repoDir + "/docker-compose.yml"

	// git pull
	pullCmd := exec.Command("git", "-C", repoDir, "pull", "--ff-only")
	pullOut, err := pullCmd.CombinedOutput()
	if err != nil {
		log.Printf("git pull failed: %s\n%s", err, pullOut)
		return
	}
	log.Printf("git pull: %s", pullOut)

	// Force rebuild and re-run the web-build one-shot container so new
	// assets are copied into the shared volume that Caddy serves from.
	rmCmd := exec.Command("docker", "compose", "-f", composeFile, "rm", "-fsv", "web-build")
	rmCmd.Dir = repoDir
	rmOut, _ := rmCmd.CombinedOutput()
	log.Printf("web-build rm: %s", rmOut)

	// docker compose up -d --build
	composeCmd := exec.Command("docker", "compose", "-f", composeFile, "up", "-d", "--build")
	composeCmd.Dir = repoDir
	composeOut, err := composeCmd.CombinedOutput()
	if err != nil {
		log.Printf("docker compose failed: %s\n%s", err, composeOut)
		return
	}
	log.Printf("deploy completed: %s", composeOut)
}

func verifySignature(body []byte, sigHeader string) bool {
	if sigHeader == "" {
		return false
	}
	prefix := "sha256="
	if !strings.HasPrefix(sigHeader, prefix) {
		return false
	}
	sig, err := hex.DecodeString(sigHeader[len(prefix):])
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hmac.Equal(sig, mac.Sum(nil))
}
