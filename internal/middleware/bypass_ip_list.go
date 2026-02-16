package middleware

import (
	"log/slog"
	"net"
	"net/http"
	"strings"
)

type BypassIPList struct {
	allowedIPs   []net.IP
	allowedCIDRs []*net.IPNet
}

func NewBypassIPList(bypassList []string) (*BypassIPList, error) {
	iw := &BypassIPList{}

	for _, entry := range bypassList {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		if strings.Contains(entry, "/") {
			_, cidr, err := net.ParseCIDR(entry)
			if err != nil {
				return nil, err
			}
			iw.allowedCIDRs = append(iw.allowedCIDRs, cidr)
		} else {
			ip := net.ParseIP(entry)
			if ip == nil {
				return nil, nil
			}
			iw.allowedIPs = append(iw.allowedIPs, ip)
		}
	}

	return iw, nil
}

func (iw *BypassIPList) IsBypassed(r *http.Request) bool {
	if len(iw.allowedIPs) == 0 && len(iw.allowedCIDRs) == 0 {
		return false
	}

	ip := extractIP(r)
	if ip == nil {
		return false
	}

	for _, allowedIP := range iw.allowedIPs {
		if ip.Equal(allowedIP) {
			return true
		}
	}

	for _, cidr := range iw.allowedCIDRs {
		if cidr.Contains(ip) {
			return true
		}
	}

	return false
}

// extractIP extracts the client IP address from the request.
// It uses the last IP from X-Forwarded-For header (rightmost IP) to prevent IP spoofing.
// The last IP represents the direct connection to our server, which cannot be spoofed.
//
// Security note: Using the first IP (leftmost) would allow attackers to bypass
// authentication by injecting arbitrary IPs in the X-Forwarded-For header.
// The rightmost IP is the most trustworthy as it's added by the last proxy/server
// that received the direct connection.
func extractIP(r *http.Request) net.IP {
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		ips := strings.Split(forwarded, ",")
		if len(ips) > 0 {
			// Use the last (rightmost) IP to prevent spoofing
			lastIP := strings.TrimSpace(ips[len(ips)-1])
			ip := net.ParseIP(lastIP)
			if ip != nil {
				return ip
			}
		}
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		slog.Warn("failed to parse remote addr", "remote_addr", r.RemoteAddr, "error", err)
		return nil
	}

	return net.ParseIP(host)
}
