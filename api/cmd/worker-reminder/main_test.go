package main

import (
	"testing"
	"time"
)

func TestPluralS(t *testing.T) {
	tests := []struct {
		name string
		n    int
		want string
	}{
		{name: "zero", n: 0, want: "s"},
		{name: "one", n: 1, want: ""},
		{name: "two", n: 2, want: "s"},
		{name: "many", n: 42, want: "s"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := pluralS(tc.n)
			if got != tc.want {
				t.Errorf("pluralS(%d) = %q, want %q", tc.n, got, tc.want)
			}
		})
	}
}

func TestEnvString(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		envVal   string
		fallback string
		want     string
	}{
		{name: "uses fallback when unset", key: "TEST_ENVSTR_UNSET", envVal: "", fallback: "default", want: "default"},
		{name: "uses env when set", key: "TEST_ENVSTR_SET", envVal: "custom", fallback: "default", want: "custom"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.envVal != "" {
				t.Setenv(tc.key, tc.envVal)
			}
			got := envString(tc.key, tc.fallback)
			if got != tc.want {
				t.Errorf("envString(%q) = %q, want %q", tc.key, got, tc.want)
			}
		})
	}
}

func TestEnvInt(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		envVal   string
		fallback int
		want     int
	}{
		{name: "uses fallback when unset", key: "TEST_ENVINT_UNSET", envVal: "", fallback: 8, want: 8},
		{name: "parses valid int", key: "TEST_ENVINT_VALID", envVal: "10", fallback: 8, want: 10},
		{name: "uses fallback on invalid", key: "TEST_ENVINT_BAD", envVal: "abc", fallback: 8, want: 8},
		{name: "parses zero", key: "TEST_ENVINT_ZERO", envVal: "0", fallback: 8, want: 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.envVal != "" {
				t.Setenv(tc.key, tc.envVal)
			}
			got := envInt(tc.key, tc.fallback)
			if got != tc.want {
				t.Errorf("envInt(%q) = %d, want %d", tc.key, got, tc.want)
			}
		})
	}
}

func TestEnvDuration(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		envVal   string
		fallback time.Duration
		want     time.Duration
	}{
		{name: "uses fallback when unset", key: "TEST_ENVDUR_UNSET", envVal: "", fallback: 10 * time.Minute, want: 10 * time.Minute},
		{name: "parses valid duration", key: "TEST_ENVDUR_VALID", envVal: "5m", fallback: 10 * time.Minute, want: 5 * time.Minute},
		{name: "parses seconds", key: "TEST_ENVDUR_SEC", envVal: "30s", fallback: 10 * time.Minute, want: 30 * time.Second},
		{name: "uses fallback on invalid", key: "TEST_ENVDUR_BAD", envVal: "nope", fallback: 10 * time.Minute, want: 10 * time.Minute},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.envVal != "" {
				t.Setenv(tc.key, tc.envVal)
			}
			got := envDuration(tc.key, tc.fallback)
			if got != tc.want {
				t.Errorf("envDuration(%q) = %v, want %v", tc.key, got, tc.want)
			}
		})
	}
}
