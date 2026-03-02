package middleware

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// extractToken extracts the token from the Authorization header
// This mirrors the logic in Auth.VerifyToken without the OIDC verification
func extractToken(authHeader string) (string, error) {
	if authHeader == "" {
		return "", errMissingAuth
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 {
		return "", errInvalidFormat
	}

	authScheme := strings.ToLower(parts[0])

	switch authScheme {
	case "bearer":
		token := parts[1]
		if token == "" {
			return "", errEmptyBearer
		}
		return token, nil

	case "basic":
		decoded, err := base64.StdEncoding.DecodeString(parts[1])
		if err != nil {
			return "", errInvalidBasicEncoding
		}

		credentials := strings.SplitN(string(decoded), ":", 2)
		if len(credentials) != 2 {
			return "", errInvalidBasicFormat
		}

		username := credentials[0]
		password := credentials[1]

		if strings.ToLower(username) != "bearer" {
			return "", errInvalidBasicUsername
		}

		if password == "" {
			return "", errEmptyBasicToken
		}

		return password, nil

	default:
		return "", errUnsupportedScheme
	}
}

// Sentinel errors for testing
var (
	errMissingAuth          = stringError("missing Authorization header")
	errInvalidFormat        = stringError("invalid Authorization header format")
	errEmptyBearer          = stringError("empty bearer token")
	errInvalidBasicEncoding = stringError("invalid Basic auth encoding")
	errInvalidBasicFormat   = stringError("invalid Basic auth format")
	errInvalidBasicUsername = stringError("Basic auth username must be 'bearer'")
	errEmptyBasicToken      = stringError("empty token in Basic auth")
	errUnsupportedScheme    = stringError("unsupported Authorization scheme")
)

type stringError string

func (e stringError) Error() string { return string(e) }

func TestExtractToken_BearerAuth(t *testing.T) {
	tests := []struct {
		name      string
		header    string
		wantToken string
		wantErr   error
	}{
		{
			name:      "valid bearer token",
			header:    "Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.test",
			wantToken: "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.test",
			wantErr:   nil,
		},
		{
			name:      "bearer case insensitive",
			header:    "bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.test",
			wantToken: "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.test",
			wantErr:   nil,
		},
		{
			name:      "BEARER uppercase",
			header:    "BEARER eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.test",
			wantToken: "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.test",
			wantErr:   nil,
		},
		{
			name:      "empty bearer token",
			header:    "Bearer ",
			wantToken: "",
			wantErr:   errEmptyBearer,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := extractToken(tt.header)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("extractToken() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if err != tt.wantErr {
					t.Errorf("extractToken() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("extractToken() unexpected error = %v", err)
				return
			}

			if token != tt.wantToken {
				t.Errorf("extractToken() token = %v, want %v", token, tt.wantToken)
			}
		})
	}
}

func TestExtractToken_BasicAuth(t *testing.T) {
	tests := []struct {
		name      string
		header    string
		wantToken string
		wantErr   error
	}{
		{
			name:      "valid basic auth with bearer username",
			header:    "Basic " + base64.StdEncoding.EncodeToString([]byte("bearer:my-jwt-token")),
			wantToken: "my-jwt-token",
			wantErr:   nil,
		},
		{
			name:      "basic auth case insensitive username",
			header:    "Basic " + base64.StdEncoding.EncodeToString([]byte("BEARER:my-jwt-token")),
			wantToken: "my-jwt-token",
			wantErr:   nil,
		},
		{
			name:      "basic auth scheme case insensitive",
			header:    "basic " + base64.StdEncoding.EncodeToString([]byte("bearer:my-jwt-token")),
			wantToken: "my-jwt-token",
			wantErr:   nil,
		},
		{
			name:      "basic auth with complex jwt token",
			header:    "Basic " + base64.StdEncoding.EncodeToString([]byte("bearer:eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.sig")),
			wantToken: "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.sig",
			wantErr:   nil,
		},
		{
			name:    "invalid base64 encoding",
			header:  "Basic not-valid-base64!!!",
			wantErr: errInvalidBasicEncoding,
		},
		{
			name:    "username not bearer",
			header:  "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:my-jwt-token")),
			wantErr: errInvalidBasicUsername,
		},
		{
			name:    "empty password",
			header:  "Basic " + base64.StdEncoding.EncodeToString([]byte("bearer:")),
			wantErr: errEmptyBasicToken,
		},
		{
			name:    "missing colon separator",
			header:  "Basic " + base64.StdEncoding.EncodeToString([]byte("bearertoken")),
			wantErr: errInvalidBasicFormat,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := extractToken(tt.header)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("extractToken() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if err != tt.wantErr {
					t.Errorf("extractToken() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("extractToken() unexpected error = %v", err)
				return
			}

			if token != tt.wantToken {
				t.Errorf("extractToken() token = %v, want %v", token, tt.wantToken)
			}
		})
	}
}

func TestExtractToken_InvalidAuth(t *testing.T) {
	tests := []struct {
		name    string
		header  string
		wantErr error
	}{
		{
			name:    "missing authorization header",
			header:  "",
			wantErr: errMissingAuth,
		},
		{
			name:    "invalid format - no space",
			header:  "Bearertoken",
			wantErr: errInvalidFormat,
		},
		{
			name:    "invalid format - only scheme",
			header:  "Bearer",
			wantErr: errInvalidFormat,
		},
		{
			name:    "unsupported scheme - digest",
			header:  "Digest username=test",
			wantErr: errUnsupportedScheme,
		},
		{
			name:    "unsupported scheme - negotiate",
			header:  "Negotiate abc123",
			wantErr: errUnsupportedScheme,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := extractToken(tt.header)

			if err == nil {
				t.Errorf("extractToken() error = nil, wantErr %v", tt.wantErr)
				return
			}

			if err != tt.wantErr {
				t.Errorf("extractToken() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestExtractToken_WithHTTPRequest(t *testing.T) {
	tests := []struct {
		name      string
		setupReq  func(*http.Request)
		wantToken string
		wantErr   error
	}{
		{
			name: "bearer token from request",
			setupReq: func(r *http.Request) {
				r.Header.Set("Authorization", "Bearer test-token-123")
			},
			wantToken: "test-token-123",
			wantErr:   nil,
		},
		{
			name: "basic auth from request",
			setupReq: func(r *http.Request) {
				r.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("bearer:test-token-456")))
			},
			wantToken: "test-token-456",
			wantErr:   nil,
		},
		{
			name: "no authorization header",
			setupReq: func(r *http.Request) {
				// Don't set Authorization header
			},
			wantErr: errMissingAuth,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "http://example.com", nil)
			tt.setupReq(req)

			token, err := extractToken(req.Header.Get("Authorization"))

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("extractToken() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if err != tt.wantErr {
					t.Errorf("extractToken() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("extractToken() unexpected error = %v", err)
				return
			}

			if token != tt.wantToken {
				t.Errorf("extractToken() token = %v, want %v", token, tt.wantToken)
			}
		})
	}
}

// TestAuth_VerifyToken_MissingHeader tests that VerifyToken returns an error
// when the Authorization header is missing. This requires a real Auth instance
// but will fail early before OIDC verification.
func TestAuth_VerifyToken_MissingHeader(t *testing.T) {
	// Create Auth with an invalid audience - verification won't reach OIDC
	auth, err := NewAuth("test-audience")
	if err != nil {
		t.Skipf("Cannot create Auth (expected in unit test environment): %v", err)
	}

	req := httptest.NewRequest("GET", "http://example.com", nil)
	// Don't set Authorization header

	_, err = auth.VerifyToken(context.Background(), req)
	if err == nil {
		t.Error("VerifyToken() should return error for missing Authorization header")
	}

	if !strings.Contains(err.Error(), "missing Authorization header") {
		t.Errorf("VerifyToken() error = %v, want error containing 'missing Authorization header'", err)
	}
}
