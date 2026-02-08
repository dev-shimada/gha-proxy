package middleware

import (
	"log/slog"
	"net"
	"net/http"
	"strings"
)

type IPWhitelist struct {
	allowedIPs   []net.IP
	allowedCIDRs []*net.IPNet
}

func NewIPWhitelist(whitelist []string) (*IPWhitelist, error) {
	iw := &IPWhitelist{}

	for _, entry := range whitelist {
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

func (iw *IPWhitelist) IsWhitelisted(r *http.Request) bool {
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

func extractIP(r *http.Request) net.IP {
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		ips := strings.Split(forwarded, ",")
		if len(ips) > 0 {
			ip := net.ParseIP(strings.TrimSpace(ips[0]))
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
