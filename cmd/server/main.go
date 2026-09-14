package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	deliveryhttp "github.com/akordium-id/mergaite-core/internal/core/delivery/http"
	v1 "github.com/akordium-id/mergaite-core/internal/core/delivery/http/v1"
	"github.com/akordium-id/mergaite-core/internal/core/repository/postgres"
	"github.com/akordium-id/mergaite-core/internal/core/usecase/tenant"
	"github.com/akordium-id/mergaite-core/pkg/config"
	"github.com/akordium-id/mergaite-core/pkg/database"
)

func main() {
	// Initialize structured logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load configuration", slog.Any("error", err))
		os.Exit(1)
	}

	slog.Info("starting application",
		slog.String("app", cfg.AppName),
		slog.String("env", cfg.AppEnv),
		slog.String("port", cfg.AppPort),
	)

	// Database Connection Pool
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dbPool, err := database.NewPostgresPool(ctx, cfg)
	if err != nil {
		slog.Warn("database connection failed (will run in degraded mode if offline)", slog.Any("error", err))
	} else {
		defer dbPool.Close()
	}

	// Layer Wiring (Clean Architecture)
	var tenantRepo = postgres.NewTenantRepository(dbPool)
	tenantUsecase := tenant.NewUsecase(tenantRepo)
	tenantHandler := v1.NewTenantHandler(tenantUsecase)

	handlers := deliveryhttp.Handlers{
		TenantHandler: tenantHandler,
	}

	router := deliveryhttp.NewRouter(dbPool, handlers)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.AppPort),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Server shutdown channel
	shutdownErrChan := make(chan error, 1)

	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		sig := <-quit

		slog.Info("received shutdown signal", slog.String("signal", sig.String()))

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()

		shutdownErrChan <- server.Shutdown(shutdownCtx)
	}()

	slog.Info("server listening", slog.String("addr", server.Addr))
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("server fatal error", slog.Any("error", err))
		os.Exit(1)
	}

	if err := <-shutdownErrChan; err != nil {
		slog.Error("error during server shutdown", slog.Any("error", err))
	}

	slog.Info("server exited cleanly")
}
