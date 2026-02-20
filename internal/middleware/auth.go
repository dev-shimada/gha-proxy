package middleware

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/dev-shimada/gha-proxy/internal/oidc"
)

type Auth struct {
	verifier *oidc.Verifier
}

func NewAuth(audience string) (*Auth, error) {
	verifier, err := oidc.New(audience)
	if err != nil {
		return nil, err
	}

	return &Auth{
		verifier: verifier,
	}, nil
}

func (a *Auth) VerifyToken(ctx context.Context, r *http.Request) (*oidc.Claims, error) {
	fmt.Printf("Header: %v\n", r.Header)
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return nil, errors.New("missing Authorization header")
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 {
		return nil, errors.New("invalid Authorization header format")
	}

	var token string
	authScheme := strings.ToLower(parts[0])

	switch authScheme {
	case "bearer":
		// Standard Bearer token: Authorization: Bearer <token>
		token = parts[1]
		if token == "" {
			return nil, errors.New("empty bearer token")
		}

	case "basic":
		// Basic Auth for GOPROXY URL-embedded credentials: https://bearer:token@host
		// Decode base64(username:password)
		decoded, err := base64.StdEncoding.DecodeString(parts[1])
		if err != nil {
			return nil, fmt.Errorf("invalid Basic auth encoding: %w", err)
		}

		credentials := strings.SplitN(string(decoded), ":", 2)
		if len(credentials) != 2 {
			return nil, errors.New("invalid Basic auth format")
		}

		username := credentials[0]
		password := credentials[1]

		// Only accept Basic Auth if username is "bearer" (case-insensitive)
		if strings.ToLower(username) != "bearer" {
			return nil, fmt.Errorf("Basic auth username must be 'bearer', got '%s'", username)
		}

		// Use password as Bearer token
		token = password
		if token == "" {
			return nil, errors.New("empty token in Basic auth")
		}

		slog.Debug("extracted Bearer token from Basic auth",
			"username", username,
			"token_length", len(token),
		)

	default:
		return nil, fmt.Errorf("unsupported Authorization scheme: %s", authScheme)
	}

	result, err := a.verifier.Verify(ctx, token)
	if err != nil {
		slog.Warn("token verification failed", "error", err)
		return nil, err
	}

	return result, nil
}
