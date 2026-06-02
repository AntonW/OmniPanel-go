package main

import "testing"

func TestServerAddrToHTTP(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"wss://example.com:443", "https://example.com:443"},
		{"ws://example.com:80", "http://example.com:80"},
		{"example.com:3000", "http://example.com:3000"},
		{"https://example.com", "https://example.com"},
	}

	for _, tc := range tests {
		if got := serverAddrToHTTP(tc.in); got != tc.want {
			t.Fatalf("serverAddrToHTTP(%q)=%q, want %q", tc.in, got, tc.want)
		}
	}
}

