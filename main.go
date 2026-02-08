package main

import (
	"context"
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
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	proxyHandler, err := proxy.New(cfg.GoproxyURL)
	if err != nil {
		slog.Error("failed to create proxy", "error", err)
		os.Exit(1)
	}

	ipWhitelist, err := middleware.NewIPWhitelist(cfg.IPWhitelist)
	if err != nil {
		slog.Error("failed to create IP whitelist", "error", err)
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

		if ipWhitelist.IsWhitelisted(r) {
			slog.Info("request allowed by IP whitelist",
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
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		modulePath, err := matcher.ExtractModulePath(r.URL.Path)
		if err != nil {
			slog.Warn("failed to extract module path",
				"remote_ip", remoteIP,
				"path", r.URL.Path,
				"error", err,
			)
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		matches, err := matcher.MatchesRepository(modulePath, claims.Repository)
		if err != nil {
			slog.Warn("failed to match repository",
				"remote_ip", remoteIP,
				"path", r.URL.Path,
				"module_path", modulePath,
				"claim_repository", claims.Repository,
				"error", err,
			)
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		if !matches {
			slog.Warn("repository mismatch",
				"remote_ip", remoteIP,
				"path", r.URL.Path,
				"module_path", modulePath,
				"claim_repository", claims.Repository,
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

	go func() {
		slog.Info("starting server", "port", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
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

