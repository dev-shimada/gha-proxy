package middleware

import (
	"context"
	"errors"
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
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return nil, errors.New("missing Authorization header")
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return nil, errors.New("invalid Authorization header format")
	}

	token := parts[1]
	if token == "" {
		return nil, errors.New("empty token")
	}

	result, err := a.verifier.Verify(ctx, token)
	if err != nil {
		slog.Warn("token verification failed", "error", err)
		return nil, err
	}

	return result, nil
}
