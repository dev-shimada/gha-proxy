package middleware

import (
	"net/http/httptest"
	"testing"
)

func TestNewBypassIPList(t *testing.T) {
	tests := []struct {
		name      string
		bypassList []string
		wantErr   bool
	}{
		{
			name:      "valid IPs",
			bypassList: []string{"127.0.0.1", "192.168.1.1"},
			wantErr:   false,
		},
		{
			name:      "valid CIDRs",
			bypassList: []string{"192.168.1.0/24", "10.0.0.0/8"},
			wantErr:   false,
		},
		{
			name:      "mixed IPs and CIDRs",
			bypassList: []string{"127.0.0.1", "192.168.1.0/24"},
			wantErr:   false,
		},
		{
			name:      "IPv6",
			bypassList: []string{"::1", "fe80::/10"},
			wantErr:   false,
		},
		{
			name:      "empty bypassList",
			bypassList: []string{},
			wantErr:   false,
		},
		{
			name:      "whitespace trimming",
			bypassList: []string{" 127.0.0.1 ", " 192.168.1.0/24 "},
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewBypassIPList(tt.bypassList)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewBypassIPList() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBypassIPList_IsBypassed(t *testing.T) {
	iw, err := NewBypassIPList([]string{
		"127.0.0.1",
		"192.168.1.0/24",
		"::1",
	})
	if err != nil {
		t.Fatalf("Failed to create BypassIPList: %v", err)
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
			name:       "IP not in bypassList",
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
			name:          "X-Forwarded-For with multiple IPs - uses last IP",
			remoteAddr:    "203.0.113.1:12345",
			xForwardedFor: "203.0.113.2, 192.168.1.50",
			want:          true,
		},
		{
			name:          "X-Forwarded-For with multiple IPs - last IP not in bypassList",
			remoteAddr:    "203.0.113.1:12345",
			xForwardedFor: "192.168.1.50, 203.0.113.2",
			want:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "http://example.com", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.xForwardedFor != "" {
				req.Header.Set("X-Forwarded-For", tt.xForwardedFor)
			}

			got := iw.IsBypassed(req)
			if got != tt.want {
				t.Errorf("IsBypassed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBypassIPList_EmptyBypassList(t *testing.T) {
	iw, err := NewBypassIPList([]string{})
	if err != nil {
		t.Fatalf("Failed to create BypassIPList: %v", err)
	}

	req := httptest.NewRequest("GET", "http://example.com", nil)
	req.RemoteAddr = "127.0.0.1:12345"

	if iw.IsBypassed(req) {
		t.Error("Empty bypassList should not allow any IPs")
	}
}
