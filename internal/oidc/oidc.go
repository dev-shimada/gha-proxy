package oidc

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"slices"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	githubActionsIssuer = "https://token.actions.githubusercontent.com"
	jwksURL             = "https://token.actions.githubusercontent.com/.well-known/jwks"
)

type Claims struct {
	Repository string `json:"repository"`
	Workflow   string `json:"workflow"`
	RunID      string `json:"run_id"`
	Actor      string `json:"actor"`
	jwt.RegisteredClaims
}

type Verifier struct {
	audience string
	keysCache *jwksCache
}

type jwksCache struct {
	mu         sync.RWMutex
	keys       map[string]*rsa.PublicKey
	lastUpdate time.Time
	cacheTTL   time.Duration
}

type jwks struct {
	Keys []jwk `json:"keys"`
}

type jwk struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
}

func New(audience string) (*Verifier, error) {
	if audience == "" {
		return nil, errors.New("audience is required")
	}

	return &Verifier{
		audience: audience,
		keysCache: &jwksCache{
			keys:     make(map[string]*rsa.PublicKey),
			cacheTTL: 1 * time.Hour,
		},
	}, nil
}

func (v *Verifier) Verify(ctx context.Context, tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, errors.New("missing kid in token header")
		}

		key, err := v.keysCache.getKey(ctx, kid)
		if err != nil {
			return nil, err
		}

		return key, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}

	if claims.Issuer != githubActionsIssuer {
		return nil, fmt.Errorf("invalid issuer: %s", claims.Issuer)
	}

	if !slices.Contains(claims.Audience, v.audience) {
		return nil, fmt.Errorf("invalid audience: expected %s", v.audience)
	}

	if time.Now().After(claims.ExpiresAt.Time) {
		return nil, errors.New("token expired")
	}

	return claims, nil
}

func (c *jwksCache) getKey(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	c.mu.RLock()
	if time.Since(c.lastUpdate) < c.cacheTTL {
		if key, ok := c.keys[kid]; ok {
			c.mu.RUnlock()
			return key, nil
		}
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	if time.Since(c.lastUpdate) < c.cacheTTL {
		if key, ok := c.keys[kid]; ok {
			return key, nil
		}
	}

	if err := c.refresh(ctx); err != nil {
		return nil, err
	}

	key, ok := c.keys[kid]
	if !ok {
		return nil, fmt.Errorf("key not found: %s", kid)
	}

	return key, nil
}

func (c *jwksCache) refresh(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, jwksURL, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch JWKS: status %d", resp.StatusCode)
	}

	var keySet jwks
	if err := json.NewDecoder(resp.Body).Decode(&keySet); err != nil {
		return err
	}

	newKeys := make(map[string]*rsa.PublicKey)
	for _, key := range keySet.Keys {
		if key.Kty != "RSA" {
			continue
		}

		pubKey, err := keyToRSA(key)
		if err != nil {
			continue
		}

		newKeys[key.Kid] = pubKey
	}

	c.keys = newKeys
	c.lastUpdate = time.Now()

	return nil
}

func keyToRSA(key jwk) (*rsa.PublicKey, error) {
	nBytes, err := base64Decode(key.N)
	if err != nil {
		return nil, err
	}

	eBytes, err := base64Decode(key.E)
	if err != nil {
		return nil, err
	}

	var e int
	for _, b := range eBytes {
		e = e<<8 | int(b)
	}

	n := new(big.Int).SetBytes(nBytes)

	return &rsa.PublicKey{
		N: n,
		E: e,
	}, nil
}

func base64Decode(s string) ([]byte, error) {
	switch len(s) % 4 {
	case 2:
		s += "=="
	case 3:
		s += "="
	}
	return base64.URLEncoding.DecodeString(s)
}
