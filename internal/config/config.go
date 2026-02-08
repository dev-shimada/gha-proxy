package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port         int
	IPWhitelist  []string
	Audience     string
	GoproxyURL   string
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

	ipWhitelistStr := os.Getenv("IP_WHITELIST")
	var ipWhitelist []string
	if ipWhitelistStr != "" {
		ipWhitelist = strings.Split(ipWhitelistStr, ",")
		for i, ip := range ipWhitelist {
			ipWhitelist[i] = strings.TrimSpace(ip)
		}
	}

	audience := os.Getenv("AUDIENCE")
	if audience == "" {
		return nil, fmt.Errorf("AUDIENCE is required")
	}

	goproxyURL := os.Getenv("GOPROXY_URL")
	if goproxyURL == "" {
		return nil, fmt.Errorf("GOPROXY_URL is required")
	}

	return &Config{
		Port:        port,
		IPWhitelist: ipWhitelist,
		Audience:    audience,
		GoproxyURL:  goproxyURL,
	}, nil
}
