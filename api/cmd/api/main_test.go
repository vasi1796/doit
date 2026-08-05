package main

import "testing"

func TestRedactPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "ical feed token is redacted",
			path:     "/ical/a1b2c3d4e5f6/calendar.ics",
			expected: "/ical/[redacted]/calendar.ics",
		},
		{
			name:     "ical path without trailing segment is redacted",
			path:     "/ical/a1b2c3d4e5f6",
			expected: "/ical/[redacted]",
		},
		{
			name:     "api routes pass through",
			path:     "/api/v1/tasks",
			expected: "/api/v1/tasks",
		},
		{
			name:     "authenticated ical token management route passes through",
			path:     "/api/v1/ical/token",
			expected: "/api/v1/ical/token",
		},
		{
			name:     "health check passes through",
			path:     "/healthz",
			expected: "/healthz",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := redactPath(tc.path); got != tc.expected {
				t.Errorf("redactPath(%q) = %q, want %q", tc.path, got, tc.expected)
			}
		})
	}
}
