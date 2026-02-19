package config

import (
	"os"
	"testing"
)

func TestLoad_DefaultAllowedRepositories(t *testing.T) {
	// Save original env vars
	originalPort := os.Getenv("PORT")
	originalAudience := os.Getenv("AUDIENCE")
	originalGoproxyURL := os.Getenv("BACKEND_URL")
	originalAllowedRepos := os.Getenv("ALLOWED_REPOSITORIES")

	// Restore env vars after test
	defer func() {
		os.Setenv("PORT", originalPort)
		os.Setenv("AUDIENCE", originalAudience)
		os.Setenv("BACKEND_URL", originalGoproxyURL)
		os.Setenv("ALLOWED_REPOSITORIES", originalAllowedRepos)
	}()

	// Set required env vars
	os.Setenv("AUDIENCE", "https://test.example.com")
	os.Setenv("BACKEND_URL", "https://proxy.golang.org")
	os.Unsetenv("ALLOWED_REPOSITORIES")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(cfg.AllowedRepositories) != 0 {
		t.Errorf("AllowedRepositories should be empty by default, got %v", cfg.AllowedRepositories)
	}
}

func TestLoad_AllowedRepositoriesWildcard(t *testing.T) {
	// Save original env vars
	originalPort := os.Getenv("PORT")
	originalAudience := os.Getenv("AUDIENCE")
	originalGoproxyURL := os.Getenv("BACKEND_URL")
	originalAllowedRepos := os.Getenv("ALLOWED_REPOSITORIES")

	// Restore env vars after test
	defer func() {
		os.Setenv("PORT", originalPort)
		os.Setenv("AUDIENCE", originalAudience)
		os.Setenv("BACKEND_URL", originalGoproxyURL)
		os.Setenv("ALLOWED_REPOSITORIES", originalAllowedRepos)
	}()

	// Set required env vars
	os.Setenv("AUDIENCE", "https://test.example.com")
	os.Setenv("BACKEND_URL", "https://proxy.golang.org")
	os.Setenv("ALLOWED_REPOSITORIES", "*")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(cfg.AllowedRepositories) != 1 || cfg.AllowedRepositories[0] != "*" {
		t.Errorf("AllowedRepositories should be [\"*\"], got %v", cfg.AllowedRepositories)
	}
}

func TestLoad_AllowedRepositoriesMultiple(t *testing.T) {
	// Save original env vars
	originalPort := os.Getenv("PORT")
	originalAudience := os.Getenv("AUDIENCE")
	originalGoproxyURL := os.Getenv("BACKEND_URL")
	originalAllowedRepos := os.Getenv("ALLOWED_REPOSITORIES")

	// Restore env vars after test
	defer func() {
		os.Setenv("PORT", originalPort)
		os.Setenv("AUDIENCE", originalAudience)
		os.Setenv("BACKEND_URL", originalGoproxyURL)
		os.Setenv("ALLOWED_REPOSITORIES", originalAllowedRepos)
	}()

	// Set required env vars
	os.Setenv("AUDIENCE", "https://test.example.com")
	os.Setenv("BACKEND_URL", "https://proxy.golang.org")
	os.Setenv("ALLOWED_REPOSITORIES", "owner1/repo1, owner2/* , owner3/repo3")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	expected := []string{"owner1/repo1", "owner2/*", "owner3/repo3"}
	if len(cfg.AllowedRepositories) != len(expected) {
		t.Fatalf("AllowedRepositories length mismatch: got %d, want %d", len(cfg.AllowedRepositories), len(expected))
	}

	for i, want := range expected {
		if cfg.AllowedRepositories[i] != want {
			t.Errorf("AllowedRepositories[%d] = %v, want %v", i, cfg.AllowedRepositories[i], want)
		}
	}
}
