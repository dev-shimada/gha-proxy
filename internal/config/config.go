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

	return &Config{
		Port:                port,
		BypassIPList:        bypassIPList,
		Audience:            audience,
		BackendURL:          backendURL,
		AllowedRepositories: allowedRepositories,
	}, nil
}
