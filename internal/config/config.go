package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port                 int
	BypassIPList         []string
	Audience             string
	BackendURL           string
	AllowedRepositories  []string
	TLSEnabled           bool
	TLSCertFile          string
	TLSKeyFile           string
}

func Load() (*Config, error) {
	portStr := os.Getenv("PORT")
	if portStr == "" {
		portStr = "8080"
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, fmt.Errorf("invalid PORT: %w", err)
	}

	bypassIPListStr := os.Getenv("BYPASS_IP_LIST")
	var bypassIPList []string
	if bypassIPListStr != "" {
		bypassIPList = strings.Split(bypassIPListStr, ",")
		for i, ip := range bypassIPList {
			bypassIPList[i] = strings.TrimSpace(ip)
		}
	}

	audience := os.Getenv("AUDIENCE")
	if audience == "" {
		return nil, fmt.Errorf("AUDIENCE is required")
	}

	backendURL := os.Getenv("BACKEND_URL")
	if backendURL == "" {
		return nil, fmt.Errorf("BACKEND_URL is required")
	}

	allowedReposStr := os.Getenv("ALLOWED_REPOSITORIES")
	var allowedRepositories []string
	if allowedReposStr != "" {
		allowedRepositories = strings.Split(allowedReposStr, ",")
		for i, repo := range allowedRepositories {
			allowedRepositories[i] = strings.TrimSpace(repo)
		}
	}

	tlsEnabled := false
	if tlsEnabledStr := os.Getenv("TLS_ENABLED"); tlsEnabledStr == "true" {
		tlsEnabled = true
	}

	tlsCertFile := os.Getenv("TLS_CERT_FILE")
	tlsKeyFile := os.Getenv("TLS_KEY_FILE")

	if tlsEnabled && (tlsCertFile == "" || tlsKeyFile == "") {
		return nil, fmt.Errorf("TLS_CERT_FILE and TLS_KEY_FILE are required when TLS_ENABLED is true")
	}

	return &Config{
		Port:                port,
		BypassIPList:        bypassIPList,
		Audience:            audience,
		BackendURL:          backendURL,
		AllowedRepositories: allowedRepositories,
		TLSEnabled:          tlsEnabled,
		TLSCertFile:         tlsCertFile,
		TLSKeyFile:          tlsKeyFile,
	}, nil
}
