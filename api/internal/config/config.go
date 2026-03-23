package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port              int
	LogLevel          string
	LogFormat         string
	DatabaseURL       string
	DBMaxOpenConns    int
	DBMaxIdleConns    int
	DBConnMaxLifetime time.Duration
	JWTSecret         string
	JWTExpiryHours    int
	ShutdownTimeout   time.Duration
	GoogleClientID    string
	GoogleClientSecret string
	GoogleRedirectURL  string
	AllowedEmails      []string
	DevMode            bool
	FrontendURL        string
	SecureCookies      bool
	MetricsEnabled     bool
	CORSOrigins        []string
	VAPIDPublicKey     string
	VAPIDPrivateKey    string
	VAPIDSubject       string
	ICalBaseURL          string
	DeployWebhookSecret  string
}

func Load() (*Config, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	cfg := &Config{
		Port:               envInt("PORT", 8080),
		LogLevel:           envString("LOG_LEVEL", "info"),
		LogFormat:          envString("LOG_FORMAT", "console"),
		DatabaseURL:        dbURL,
		DBMaxOpenConns:     envInt("DB_MAX_OPEN_CONNS", 10),
		DBMaxIdleConns:     envInt("DB_MAX_IDLE_CONNS", 5),
		DBConnMaxLifetime:  envDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute),
		JWTSecret:          os.Getenv("JWT_SECRET"),
		JWTExpiryHours:     envInt("JWT_EXPIRY_HOURS", 72),
		ShutdownTimeout:    envDuration("SHUTDOWN_TIMEOUT", 15*time.Second),
		GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		GoogleRedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"),
		AllowedEmails:      envStringSlice("ALLOWED_EMAILS"),
		DevMode:            envBool("DEV_MODE", false),
		FrontendURL:        envString("FRONTEND_URL", "/"),
		SecureCookies:      envBool("SECURE_COOKIES", true),
		MetricsEnabled:     envBool("METRICS_ENABLED", true),
		CORSOrigins:        envStringSlice("CORS_ORIGINS"),
		VAPIDPublicKey:     os.Getenv("VAPID_PUBLIC_KEY"),
		VAPIDPrivateKey:    os.Getenv("VAPID_PRIVATE_KEY"),
		VAPIDSubject:       envString("VAPID_SUBJECT", "admin@localhost"),
		ICalBaseURL:          envString("ICAL_BASE_URL", ""),
		DeployWebhookSecret:  os.Getenv("DEPLOY_WEBHOOK_SECRET"),
	}

	if !cfg.DevMode && cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required when DEV_MODE is not enabled")
	}

	if !cfg.DevMode && (cfg.GoogleClientID == "" || cfg.GoogleClientSecret == "" || cfg.GoogleRedirectURL == "") {
		return nil, fmt.Errorf("GOOGLE_CLIENT_ID, GOOGLE_CLIENT_SECRET, and GOOGLE_REDIRECT_URL are required when DEV_MODE is not enabled")
	}

	return cfg, nil
}

func envString(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		log.Printf("WARNING: env %s=%q is not a valid int, using fallback %d: %v", key, v, fallback, err)
		return fallback
	}
	return n
}

func envBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		log.Printf("WARNING: env %s=%q is not a valid bool, using fallback %v: %v", key, v, fallback, err)
		return fallback
	}
	return b
}

func envStringSlice(key string) []string {
	v := os.Getenv(key)
	if v == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, strings.ToLower(trimmed))
		}
	}
	return result
}

func envDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		log.Printf("WARNING: env %s=%q is not a valid duration, using fallback %v: %v", key, v, fallback, err)
		return fallback
	}
	return d
}
