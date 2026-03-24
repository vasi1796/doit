package main

import (
	"testing"
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
