package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/folsik/doit-it/internal/api"
	"github.com/folsik/doit-it/internal/auth"
	"github.com/folsik/doit-it/internal/config"
	"github.com/folsik/doit-it/internal/store"
)

func Run() error {
	cfg := config.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if strings.EqualFold(cfg.Env, "production") {
		if strings.TrimSpace(cfg.JWTPrivateKey) == "" {
			return errors.New("JWT_PRIVATE_KEY is required in production; provide a base64url-encoded Ed25519 private key")
		}
		if strings.Contains(cfg.DatabaseURL, "doit_dev_password") {
			return errors.New("development database credentials are not allowed in production")
		}
		databaseURL, err := url.Parse(cfg.DatabaseURL)
		if err != nil || databaseURL.Scheme != "postgres" && databaseURL.Scheme != "postgresql" {
			return errors.New("production DATABASE_URL must be a valid PostgreSQL URL")
		}
		sslModes := databaseURL.Query()["sslmode"]
		if len(sslModes) != 1 || sslModes[0] != "verify-full" {
			return errors.New("production DATABASE_URL must set exactly one sslmode=verify-full parameter")
		}
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := store.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	migrationCtx, cancelMigration := context.WithTimeout(ctx, 30*time.Second)
	defer cancelMigration()
	if err := db.Migrate(migrationCtx); err != nil {
		return fmt.Errorf("run database migrations: %w", err)
	}

	var tokens *auth.Manager
	if strings.TrimSpace(cfg.JWTPrivateKey) != "" {
		tokens, err = auth.NewManagerFromPrivateKey(cfg.JWTIssuer, cfg.AccessTokenTTL, cfg.JWTPrivateKey)
	} else {
		logger.Warn("JWT_PRIVATE_KEY is unset; generated signing key will change on restart (development only)")
		tokens, err = auth.NewManager(cfg.JWTIssuer, cfg.AccessTokenTTL)
	}
	if err != nil {
		return fmt.Errorf("initialize token manager: %w", err)
	}

	handler := api.New(db.DB, tokens, logger).Handler()
	srv := &http.Server{
		Addr: cfg.HTTPAddr, Handler: handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout: 15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout: 60 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	listener, err := net.Listen("tcp", cfg.HTTPAddr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", cfg.HTTPAddr, err)
	}
	serverErr := make(chan error, 1)
	go func() {
		logger.Info("HTTP server starting", "addr", cfg.HTTPAddr, "env", cfg.Env)
		if err := srv.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
	case <-ctx.Done():
	case err := <-serverErr:
		stop()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
		return fmt.Errorf("HTTP server failed: %w", err)
	}

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful HTTP shutdown: %w", err)
	}
	return nil
}
