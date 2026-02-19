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

func IsAllowedRepository(repositoryOrModulePath string, allowedPatterns []string) (bool, error) {
	var repository string

	// If it starts with github.com/, extract the repository
	if strings.HasPrefix(repositoryOrModulePath, "github.com/") {
		var err error
		repository, err = ExtractRepository(repositoryOrModulePath)
		if err != nil {
			return false, err
		}
	} else {
		// Assume it's already in "owner/repo" format
		repository = repositoryOrModulePath
	}

	repository = strings.ToLower(repository)

	for _, pattern := range allowedPatterns {
		pattern = strings.ToLower(strings.TrimSpace(pattern))

		if pattern == "*" {
			return true, nil
		}

		if owner, ok := strings.CutSuffix(pattern, "/*"); ok {
			repoOwner := strings.Split(repository, "/")[0]
			if owner == repoOwner {
				return true, nil
			}
		} else {
			if pattern == repository {
				return true, nil
			}
		}
	}

	return false, nil
}
