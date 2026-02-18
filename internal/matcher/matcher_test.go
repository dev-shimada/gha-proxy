package matcher

import (
	"testing"
)

func TestExtractModulePath(t *testing.T) {
	tests := []struct {
		name        string
		requestPath string
		want        string
		wantErr     bool
	}{
		{
			name:        "github module with version",
			requestPath: "/github.com/myorg/myrepo/@v/v1.0.0.info",
			want:        "github.com/myorg/myrepo",
			wantErr:     false,
		},
		{
			name:        "github module with list",
			requestPath: "/github.com/myorg/myrepo/@v/list",
			want:        "github.com/myorg/myrepo",
			wantErr:     false,
		},
		{
			name:        "github module with subpath",
			requestPath: "/github.com/myorg/myrepo/pkg/foo/@v/v1.0.0.mod",
			want:        "github.com/myorg/myrepo/pkg/foo",
			wantErr:     false,
		},
		{
			name:        "golang.org module",
			requestPath: "/golang.org/x/text/@v/list",
			want:        "golang.org/x/text",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractModulePath(tt.requestPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractModulePath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ExtractModulePath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractRepository(t *testing.T) {
	tests := []struct {
		name       string
		modulePath string
		want       string
		wantErr    bool
	}{
		{
			name:       "basic github repo",
			modulePath: "github.com/myorg/myrepo",
			want:       "myorg/myrepo",
			wantErr:    false,
		},
		{
			name:       "github repo with subpath",
			modulePath: "github.com/myorg/myrepo/pkg/foo",
			want:       "myorg/myrepo",
			wantErr:    false,
		},
		{
			name:       "non-github host",
			modulePath: "gitlab.com/myorg/myrepo",
			want:       "",
			wantErr:    true,
		},
		{
			name:       "invalid format",
			modulePath: "github.com/myorg",
			want:       "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractRepository(tt.modulePath)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractRepository() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ExtractRepository() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMatchesRepository(t *testing.T) {
	tests := []struct {
		name            string
		modulePath      string
		claimRepository string
		want            bool
		wantErr         bool
	}{
		{
			name:            "exact match",
			modulePath:      "github.com/myorg/myrepo",
			claimRepository: "myorg/myrepo",
			want:            true,
			wantErr:         false,
		},
		{
			name:            "case insensitive match",
			modulePath:      "github.com/MyOrg/MyRepo",
			claimRepository: "myorg/myrepo",
			want:            true,
			wantErr:         false,
		},
		{
			name:            "subpath still matches repo",
			modulePath:      "github.com/myorg/myrepo/pkg/foo",
			claimRepository: "myorg/myrepo",
			want:            true,
			wantErr:         false,
		},
		{
			name:            "different repo",
			modulePath:      "github.com/myorg/myrepo",
			claimRepository: "myorg/otherrepo",
			want:            false,
			wantErr:         false,
		},
		{
			name:            "different org",
			modulePath:      "github.com/myorg/myrepo",
			claimRepository: "otherorg/myrepo",
			want:            false,
			wantErr:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MatchesRepository(tt.modulePath, tt.claimRepository)
			if (err != nil) != tt.wantErr {
				t.Errorf("MatchesRepository() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("MatchesRepository() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsAllowedRepository(t *testing.T) {
	tests := []struct {
		name            string
		modulePath      string
		allowedPatterns []string
		want            bool
		wantErr         bool
	}{
		{
			name:            "wildcard allows all",
			modulePath:      "github.com/myorg/myrepo",
			allowedPatterns: []string{"*"},
			want:            true,
			wantErr:         false,
		},
		{
			name:            "exact match",
			modulePath:      "github.com/dev-shimada/package1",
			allowedPatterns: []string{"dev-shimada/package1"},
			want:            true,
			wantErr:         false,
		},
		{
			name:            "multiple exact matches",
			modulePath:      "github.com/dev-shimada/package2",
			allowedPatterns: []string{"dev-shimada/package1", "dev-shimada/package2"},
			want:            true,
			wantErr:         false,
		},
		{
			name:            "owner wildcard match",
			modulePath:      "github.com/dev-shimada/any-package",
			allowedPatterns: []string{"dev-shimada/*"},
			want:            true,
			wantErr:         false,
		},
		{
			name:            "owner wildcard with subpath",
			modulePath:      "github.com/dev-shimada/package/subpath",
			allowedPatterns: []string{"dev-shimada/*"},
			want:            true,
			wantErr:         false,
		},
		{
			name:            "mixed patterns",
			modulePath:      "github.com/otherowner/shared-lib",
			allowedPatterns: []string{"dev-shimada/*", "otherowner/shared-lib"},
			want:            true,
			wantErr:         false,
		},
		{
			name:            "no match",
			modulePath:      "github.com/other/package",
			allowedPatterns: []string{"dev-shimada/*"},
			want:            false,
			wantErr:         false,
		},
		{
			name:            "case insensitive",
			modulePath:      "github.com/Dev-Shimada/Package1",
			allowedPatterns: []string{"dev-shimada/package1"},
			want:            true,
			wantErr:         false,
		},
		{
			name:            "whitespace trimming",
			modulePath:      "github.com/dev-shimada/package1",
			allowedPatterns: []string{" dev-shimada/package1 "},
			want:            true,
			wantErr:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := IsAllowedRepository(tt.modulePath, tt.allowedPatterns)
			if (err != nil) != tt.wantErr {
				t.Errorf("IsAllowedRepository() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("IsAllowedRepository() = %v, want %v", got, tt.want)
			}
		})
	}
}
