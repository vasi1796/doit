package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name     string
		env      map[string]string
		wantErr  bool
		validate func(t *testing.T, cfg *Config)
	}{
		{
			name:    "missing DATABASE_URL returns error",
			env:     map[string]string{},
			wantErr: true,
		},
		{
			name: "missing JWT_SECRET without dev mode returns error",
			env: map[string]string{
				"DATABASE_URL": "postgres://localhost/test",
			},
			wantErr: true,
		},
		{
			name: "missing Google creds without dev mode returns error",
			env: map[string]string{
				"DATABASE_URL": "postgres://localhost/test",
				"JWT_SECRET":   "test-secret-that-is-at-least-32chars!",
			},
			wantErr: true,
		},
		{
			name: "JWT_SECRET too short without dev mode returns error",
			env: map[string]string{
				"DATABASE_URL": "postgres://localhost/test",
				"JWT_SECRET":   "short",
			},
			wantErr: true,
		},
		{
			name: "invalid REMINDER_HOUR returns error",
			env: map[string]string{
				"DATABASE_URL":  "postgres://localhost/test",
				"DEV_MODE":      "true",
				"REMINDER_HOUR": "25",
			},
			wantErr: true,
		},
		{
			name: "negative REMINDER_HOUR returns error",
			env: map[string]string{
				"DATABASE_URL":  "postgres://localhost/test",
				"DEV_MODE":      "true",
				"REMINDER_HOUR": "-1",
			},
			wantErr: true,
		},
		{
			name: "defaults applied when only DATABASE_URL set with dev mode",
			env: map[string]string{
				"DATABASE_URL": "postgres://localhost/test",
				"DEV_MODE":     "true",
			},
			validate: func(t *testing.T, cfg *Config) {
				if cfg.Port != 8080 {
					t.Errorf("Port = %d, want 8080", cfg.Port)
				}
				if cfg.LogLevel != "info" {
					t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "info")
				}
				if cfg.LogFormat != "console" {
					t.Errorf("LogFormat = %q, want %q", cfg.LogFormat, "console")
				}
				if cfg.DBMaxOpenConns != 10 {
					t.Errorf("DBMaxOpenConns = %d, want 10", cfg.DBMaxOpenConns)
				}
				if cfg.DBMaxIdleConns != 5 {
					t.Errorf("DBMaxIdleConns = %d, want 5", cfg.DBMaxIdleConns)
				}
				if cfg.DBConnMaxLifetime != 5*time.Minute {
					t.Errorf("DBConnMaxLifetime = %v, want 5m", cfg.DBConnMaxLifetime)
				}
				if cfg.JWTExpiryHours != 72 {
					t.Errorf("JWTExpiryHours = %d, want 72", cfg.JWTExpiryHours)
				}
				if cfg.ShutdownTimeout != 15*time.Second {
					t.Errorf("ShutdownTimeout = %v, want 15s", cfg.ShutdownTimeout)
				}
				if cfg.MetricsEnabled != true {
					t.Errorf("MetricsEnabled = %v, want true", cfg.MetricsEnabled)
				}
			},
		},
		{
			name: "custom values override defaults",
			env: map[string]string{
				"DATABASE_URL":        "postgres://localhost/test",
				"PORT":                "9090",
				"LOG_LEVEL":           "debug",
				"LOG_FORMAT":          "json",
				"DB_MAX_OPEN_CONNS":   "20",
				"DB_MAX_IDLE_CONNS":   "10",
				"DB_CONN_MAX_LIFETIME": "10m",
				"JWT_SECRET":          "test-secret-that-is-at-least-32chars!",
				"JWT_EXPIRY_HOURS":    "24",
				"SHUTDOWN_TIMEOUT":    "30s",
				"GOOGLE_CLIENT_ID":    "client-id",
				"GOOGLE_CLIENT_SECRET": "client-secret",
				"GOOGLE_REDIRECT_URL": "http://localhost/callback",
				"METRICS_ENABLED":     "false",
			},
			validate: func(t *testing.T, cfg *Config) {
				if cfg.Port != 9090 {
					t.Errorf("Port = %d, want 9090", cfg.Port)
				}
				if cfg.LogLevel != "debug" {
					t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "debug")
				}
				if cfg.LogFormat != "json" {
					t.Errorf("LogFormat = %q, want %q", cfg.LogFormat, "json")
				}
				if cfg.DBMaxOpenConns != 20 {
					t.Errorf("DBMaxOpenConns = %d, want 20", cfg.DBMaxOpenConns)
				}
				if cfg.DBMaxIdleConns != 10 {
					t.Errorf("DBMaxIdleConns = %d, want 10", cfg.DBMaxIdleConns)
				}
				if cfg.DBConnMaxLifetime != 10*time.Minute {
					t.Errorf("DBConnMaxLifetime = %v, want 10m", cfg.DBConnMaxLifetime)
				}
				if cfg.JWTSecret != "test-secret-that-is-at-least-32chars!" {
					t.Errorf("JWTSecret = %q, want %q", cfg.JWTSecret, "test-secret-that-is-at-least-32chars!")
				}
				if cfg.JWTExpiryHours != 24 {
					t.Errorf("JWTExpiryHours = %d, want 24", cfg.JWTExpiryHours)
				}
				if cfg.ShutdownTimeout != 30*time.Second {
					t.Errorf("ShutdownTimeout = %v, want 30s", cfg.ShutdownTimeout)
				}
				if cfg.GoogleClientID != "client-id" {
					t.Errorf("GoogleClientID = %q, want %q", cfg.GoogleClientID, "client-id")
				}
				if cfg.GoogleClientSecret != "client-secret" {
					t.Errorf("GoogleClientSecret = %q, want %q", cfg.GoogleClientSecret, "client-secret")
				}
				if cfg.GoogleRedirectURL != "http://localhost/callback" {
					t.Errorf("GoogleRedirectURL = %q, want %q", cfg.GoogleRedirectURL, "http://localhost/callback")
				}
				if cfg.MetricsEnabled != false {
					t.Errorf("MetricsEnabled = %v, want false", cfg.MetricsEnabled)
				}
			},
		},
		{
			name: "invalid int falls back to default",
			env: map[string]string{
				"DATABASE_URL": "postgres://localhost/test",
				"DEV_MODE":     "true",
				"PORT":         "not-a-number",
			},
			validate: func(t *testing.T, cfg *Config) {
				if cfg.Port != 8080 {
					t.Errorf("Port = %d, want 8080 (fallback)", cfg.Port)
				}
			},
		},
		{
			name: "invalid duration falls back to default",
			env: map[string]string{
				"DATABASE_URL":     "postgres://localhost/test",
				"DEV_MODE":         "true",
				"SHUTDOWN_TIMEOUT": "bad-duration",
			},
			validate: func(t *testing.T, cfg *Config) {
				if cfg.ShutdownTimeout != 15*time.Second {
					t.Errorf("ShutdownTimeout = %v, want 15s (fallback)", cfg.ShutdownTimeout)
				}
			},
		},
	}

	// Keys that must be cleared between test cases
	envKeys := []string{
		"DATABASE_URL", "PORT", "LOG_LEVEL", "LOG_FORMAT",
		"DB_MAX_OPEN_CONNS", "DB_MAX_IDLE_CONNS", "DB_CONN_MAX_LIFETIME",
		"JWT_SECRET", "JWT_EXPIRY_HOURS", "SHUTDOWN_TIMEOUT",
		"GOOGLE_CLIENT_ID", "GOOGLE_CLIENT_SECRET", "GOOGLE_REDIRECT_URL",
		"METRICS_ENABLED", "DEV_MODE", "REMINDER_HOUR",
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for _, k := range envKeys {
				os.Unsetenv(k)
			}

			for k, v := range tc.env {
				t.Setenv(k, v)
			}

			cfg, err := Load()

			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.validate != nil {
				tc.validate(t, cfg)
			}
		})
	}
}
