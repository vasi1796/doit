package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

func TestEscapeICalText(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "plain text unchanged",
			input: "Buy milk",
			want:  "Buy milk",
		},
		{
			name:  "backslash escaped",
			input: `path\to\file`,
			want:  `path\\to\\file`,
		},
		{
			name:  "semicolons escaped",
			input: "item1;item2;item3",
			want:  `item1\;item2\;item3`,
		},
		{
			name:  "commas escaped",
			input: "apples, bananas, oranges",
			want:  `apples\, bananas\, oranges`,
		},
		{
			name:  "newlines escaped",
			input: "line1\nline2\nline3",
			want:  `line1\nline2\nline3`,
		},
		{
			name:  "carriage returns stripped",
			input: "line1\r\nline2",
			want:  `line1\nline2`,
		},
		{
			name:  "all special characters combined",
			input: "a\\b;c,d\ne\r\n",
			want:  `a\\b\;c\,d\ne\n`,
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := escapeICalText(tc.input)
			if got != tc.want {
				t.Errorf("escapeICalText(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestFormatICalDate(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "standard date",
			input: "2026-03-23",
			want:  "20260323",
		},
		{
			name:  "beginning of year",
			input: "2026-01-01",
			want:  "20260101",
		},
		{
			name:  "end of year",
			input: "2026-12-31",
			want:  "20261231",
		},
		{
			name:  "already formatted (no dashes)",
			input: "20260323",
			want:  "20260323",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := formatICalDate(tc.input)
			if got != tc.want {
				t.Errorf("formatICalDate(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestParseDateAndTime(t *testing.T) {
	tests := []struct {
		name    string
		date    string
		time_   string
		wantErr bool
		wantUTC string // expected UTC format "20060102T150405Z"
	}{
		{
			name:    "valid date and time",
			date:    "2026-03-23",
			time_:   "14:30",
			wantErr: false,
			wantUTC: "20260323T143000Z",
		},
		{
			name:    "midnight",
			date:    "2026-01-01",
			time_:   "00:00",
			wantErr: false,
			wantUTC: "20260101T000000Z",
		},
		{
			name:    "end of day",
			date:    "2026-12-31",
			time_:   "23:59",
			wantErr: false,
			wantUTC: "20261231T235900Z",
		},
		{
			name:    "invalid date format",
			date:    "03-23-2026",
			time_:   "14:30",
			wantErr: true,
		},
		{
			name:    "invalid time format",
			date:    "2026-03-23",
			time_:   "2:30 PM",
			wantErr: true,
		},
		{
			name:    "empty time string",
			date:    "2026-03-23",
			time_:   "",
			wantErr: true,
		},
		{
			name:    "empty date string",
			date:    "",
			time_:   "14:30",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseDateAndTime(tc.date, tc.time_)
			if tc.wantErr {
				if err == nil {
					t.Errorf("parseDateAndTime(%q, %q) expected error, got nil", tc.date, tc.time_)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseDateAndTime(%q, %q) unexpected error: %v", tc.date, tc.time_, err)
			}
			formatted := got.UTC().Format("20060102T150405Z")
			if formatted != tc.wantUTC {
				t.Errorf("parseDateAndTime(%q, %q) = %q, want %q", tc.date, tc.time_, formatted, tc.wantUTC)
			}
		})
	}
}

func TestMapRecurrenceToRRule(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "daily lowercase",
			input: "daily",
			want:  "FREQ=DAILY",
		},
		{
			name:  "weekly lowercase",
			input: "weekly",
			want:  "FREQ=WEEKLY",
		},
		{
			name:  "monthly lowercase",
			input: "monthly",
			want:  "FREQ=MONTHLY",
		},
		{
			name:  "yearly lowercase",
			input: "yearly",
			want:  "FREQ=YEARLY",
		},
		{
			name:  "daily mixed case",
			input: "Daily",
			want:  "FREQ=DAILY",
		},
		{
			name:  "weekly uppercase",
			input: "WEEKLY",
			want:  "FREQ=WEEKLY",
		},
		{
			name:  "unknown rule returns empty",
			input: "biweekly",
			want:  "",
		},
		{
			name:  "empty string returns empty",
			input: "",
			want:  "",
		},
		{
			name:  "arbitrary string returns empty",
			input: "every 3 days",
			want:  "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := mapRecurrenceToRRule(tc.input)
			if got != tc.want {
				t.Errorf("mapRecurrenceToRRule(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestResolveBaseURL(t *testing.T) {
	tests := []struct {
		name       string
		baseURL    string
		reqHost    string
		wantResult string
	}{
		{
			name:       "configured baseURL used as-is",
			baseURL:    "https://doit.example.com",
			reqHost:    "ignored.host",
			wantResult: "https://doit.example.com",
		},
		{
			name:       "configured baseURL trailing slash stripped",
			baseURL:    "https://doit.example.com/",
			reqHost:    "ignored.host",
			wantResult: "https://doit.example.com",
		},
		{
			name:       "configured baseURL multiple trailing slashes stripped",
			baseURL:    "https://doit.example.com///",
			reqHost:    "ignored.host",
			wantResult: "https://doit.example.com",
		},
		{
			name:       "empty baseURL falls back to request host with https",
			baseURL:    "",
			reqHost:    "myapp.example.com",
			wantResult: "https://myapp.example.com",
		},
		{
			name:       "empty baseURL with port in host",
			baseURL:    "",
			reqHost:    "localhost:8080",
			wantResult: "https://localhost:8080",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := &ICalHandler{
				baseURL: tc.baseURL,
				logger:  zerolog.Nop(),
			}
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Host = tc.reqHost

			got := h.resolveBaseURL(req)
			if got != tc.wantResult {
				t.Errorf("resolveBaseURL() = %q, want %q", got, tc.wantResult)
			}
		})
	}
}

func TestServeCalendarMissingToken(t *testing.T) {
	// When chi URL param "token" is empty, ServeCalendar should return 400.
	h := NewICalHandler(nil, zerolog.Nop(), "")

	req := httptest.NewRequest(http.MethodGet, "/ical//calendar.ics", nil)
	// Set up chi context with empty token param
	req = withChiParam(req, "token", "")

	rr := httptest.NewRecorder()
	h.ServeCalendar(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestICalDateTimeFormatting(t *testing.T) {
	// Verify that parseDateAndTime produces timestamps that format correctly
	// for the iCal DTSTART field.
	tests := []struct {
		name     string
		date     string
		timeStr  string
		wantFmt  string
	}{
		{
			name:    "morning time",
			date:    "2026-06-15",
			timeStr: "09:00",
			wantFmt: "20260615T090000Z",
		},
		{
			name:    "afternoon time",
			date:    "2026-12-25",
			timeStr: "17:45",
			wantFmt: "20261225T174500Z",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			parsed, err := parseDateAndTime(tc.date, tc.timeStr)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			got := parsed.UTC().Format("20060102T150405Z")
			if got != tc.wantFmt {
				t.Errorf("formatted = %q, want %q", got, tc.wantFmt)
			}
		})
	}
}

func TestICalTimestampFormat(t *testing.T) {
	// Verify that LAST-MODIFIED format matches the iCal spec.
	ts := time.Date(2026, 3, 23, 14, 30, 45, 0, time.UTC)
	got := ts.UTC().Format("20060102T150405Z")
	want := "20260323T143045Z"
	if got != want {
		t.Errorf("timestamp format = %q, want %q", got, want)
	}
}
