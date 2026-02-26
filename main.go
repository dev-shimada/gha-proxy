package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dev-shimada/gha-proxy/internal/config"
	"github.com/dev-shimada/gha-proxy/internal/matcher"
	"github.com/dev-shimada/gha-proxy/internal/middleware"
	"github.com/dev-shimada/gha-proxy/internal/proxy"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	logLevel := slog.LevelInfo
	if cfg.Debug {
		logLevel = slog.LevelDebug
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))
	slog.SetDefault(logger)

	if cfg.Debug {
		slog.Debug("debug mode enabled")
	}

	proxyHandler, err := proxy.New(cfg.BackendURL, cfg.Debug)
	if err != nil {
		slog.Error("failed to create proxy", "error", err)
		os.Exit(1)
	}

	bypassIPList, err := middleware.NewBypassIPList(cfg.BypassIPList)
	if err != nil {
		slog.Error("failed to create bypass IP list", "error", err)
		os.Exit(1)
	}

	auth, err := middleware.NewAuth(cfg.Audience)
	if err != nil {
		slog.Error("failed to create auth middleware", "error", err)
		os.Exit(1)
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		remoteIP := r.RemoteAddr
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			remoteIP = forwarded
		}

		if cfg.Debug {
			headers := make(map[string]string)
			for k, v := range r.Header {
				if len(v) > 0 {
					headers[k] = v[0]
				}
			}
			slog.Debug("incoming request",
				"method", r.Method,
				"path", r.URL.Path,
				"remote_ip", remoteIP,
				"headers", headers,
			)
		}

		if bypassIPList.IsBypassed(r) {
			slog.Info("request allowed by bypass IP list",
				"remote_ip", remoteIP,
				"path", r.URL.Path,
			)
			proxyHandler.ServeHTTP(w, r)
			return
		}

		claims, err := auth.VerifyToken(ctx, r)
		if err != nil {
			slog.Warn("authentication failed",
				"remote_ip", remoteIP,
				"path", r.URL.Path,
				"error", err,
			)
			w.Header().Set("WWW-Authenticate", `Bearer realm="gha-proxy"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		allowed, err := matcher.IsAllowedRepository(claims.Repository, cfg.AllowedRepositories)
		if err != nil {
			slog.Warn("failed to check allowed repository",
				"remote_ip", remoteIP,
				"path", r.URL.Path,
				"claim_repository", claims.Repository,
				"error", err,
			)
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		if !allowed {
			slog.Warn("repository not allowed",
				"remote_ip", remoteIP,
				"path", r.URL.Path,
				"claim_repository", claims.Repository,
				"allowed_patterns", cfg.AllowedRepositories,
			)
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		slog.Info("request authenticated and authorized",
			"remote_ip", remoteIP,
			"path", r.URL.Path,
			"repository", claims.Repository,
		)
		proxyHandler.ServeHTTP(w, r)
	})

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	if cfg.TLSEnabled {
		server.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	}

	go func() {
		var err error
		if cfg.TLSEnabled {
			slog.Info("starting HTTPS server", "port", cfg.Port)
			err = server.ListenAndServeTLS(cfg.TLSCertFile, cfg.TLSKeyFile)
		} else {
			slog.Info("starting HTTP server", "port", cfg.Port)
			err = server.ListenAndServe()
		}
		if err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		slog.Error("server shutdown error", "error", err)
		os.Exit(1)
	}

	slog.Info("server stopped")
}

