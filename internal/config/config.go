package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port         int
	BypassIPList []string
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

	goproxyURL := os.Getenv("GOPROXY_URL")
	if goproxyURL == "" {
		return nil, fmt.Errorf("GOPROXY_URL is required")
	}

	return &Config{
		Port:         port,
		BypassIPList: bypassIPList,
		Audience:     audience,
		GoproxyURL:   goproxyURL,
	}, nil
}
