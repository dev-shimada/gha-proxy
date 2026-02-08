package middleware

import (
	"net/http/httptest"
	"testing"
)

func TestNewIPWhitelist(t *testing.T) {
	tests := []struct {
		name      string
		whitelist []string
		wantErr   bool
	}{
		{
			name:      "valid IPs",
			whitelist: []string{"127.0.0.1", "192.168.1.1"},
			wantErr:   false,
		},
		{
			name:      "valid CIDRs",
			whitelist: []string{"192.168.1.0/24", "10.0.0.0/8"},
			wantErr:   false,
		},
		{
			name:      "mixed IPs and CIDRs",
			whitelist: []string{"127.0.0.1", "192.168.1.0/24"},
			wantErr:   false,
		},
		{
			name:      "IPv6",
			whitelist: []string{"::1", "fe80::/10"},
			wantErr:   false,
		},
		{
			name:      "empty whitelist",
			whitelist: []string{},
			wantErr:   false,
		},
		{
			name:      "whitespace trimming",
			whitelist: []string{" 127.0.0.1 ", " 192.168.1.0/24 "},
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewIPWhitelist(tt.whitelist)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewIPWhitelist() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIPWhitelist_IsWhitelisted(t *testing.T) {
	iw, err := NewIPWhitelist([]string{
		"127.0.0.1",
		"192.168.1.0/24",
		"::1",
	})
	if err != nil {
		t.Fatalf("Failed to create IPWhitelist: %v", err)
	}

	tests := []struct {
		name       string
		remoteAddr string
		xForwardedFor string
		want       bool
	}{
		{
			name:       "localhost IPv4",
			remoteAddr: "127.0.0.1:12345",
			want:       true,
		},
		{
			name:       "localhost IPv6",
			remoteAddr: "[::1]:12345",
			want:       true,
		},
		{
			name:       "IP in CIDR range",
			remoteAddr: "192.168.1.100:12345",
			want:       true,
		},
		{
			name:       "IP not in whitelist",
			remoteAddr: "203.0.113.1:12345",
			want:       false,
		},
		{
			name:          "X-Forwarded-For takes precedence",
			remoteAddr:    "203.0.113.1:12345",
			xForwardedFor: "127.0.0.1",
			want:          true,
		},
		{
			name:          "X-Forwarded-For with multiple IPs",
			remoteAddr:    "203.0.113.1:12345",
			xForwardedFor: "192.168.1.50, 203.0.113.2",
			want:          true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "http://example.com", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.xForwardedFor != "" {
				req.Header.Set("X-Forwarded-For", tt.xForwardedFor)
			}

			got := iw.IsWhitelisted(req)
			if got != tt.want {
				t.Errorf("IsWhitelisted() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIPWhitelist_EmptyWhitelist(t *testing.T) {
	iw, err := NewIPWhitelist([]string{})
	if err != nil {
		t.Fatalf("Failed to create IPWhitelist: %v", err)
	}

	req := httptest.NewRequest("GET", "http://example.com", nil)
	req.RemoteAddr = "127.0.0.1:12345"

	if iw.IsWhitelisted(req) {
		t.Error("Empty whitelist should not allow any IPs")
	}
}
