package main

import "testing"

func TestGetHash(t *testing.T) {
	for _, tc := range []struct{ link, want string }{
		{"", "00000000"},
		{"https://example.com", "96248650"},
		{"https://example.com/69897/4885590609", "2be9aa73"},
		{"https://example.com/105844/11202952336", "2be9aa73"},
	} {
		t.Run(tc.link, func(t *testing.T) {
			if got := getHash(tc.link); got != tc.want {
				t.Fatalf("getHash(%q) = %q, want %q", tc.link, got, tc.want)
			}
		})
	}
}
