package matcher

import (
	"fmt"
	"path"
	"strings"
)

func ExtractModulePath(requestPath string) (string, error) {
	requestPath = strings.TrimPrefix(requestPath, "/")

	parts := strings.Split(requestPath, "/@")
	if len(parts) == 0 {
		return "", fmt.Errorf("invalid module path: %s", requestPath)
	}

	modulePath := parts[0]
	modulePath = path.Clean(modulePath)

	return modulePath, nil
}

func ExtractRepository(modulePath string) (string, error) {
	parts := strings.SplitN(modulePath, "/", 4)
	if len(parts) < 3 {
		return "", fmt.Errorf("invalid module path format: %s", modulePath)
	}

	if parts[0] != "github.com" {
		return "", fmt.Errorf("unsupported module host: %s", parts[0])
	}

	owner := parts[1]
	repo := parts[2]

	return fmt.Sprintf("%s/%s", owner, repo), nil
}

func MatchesRepository(modulePath, claimRepository string) (bool, error) {
	moduleRepo, err := ExtractRepository(modulePath)
	if err != nil {
		return false, err
	}

	moduleRepo = strings.ToLower(moduleRepo)
	claimRepository = strings.ToLower(claimRepository)

	return moduleRepo == claimRepository, nil
}
