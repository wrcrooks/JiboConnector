// Command worker is JiboConnector's entrypoint: it periodically checks
// jibo-api for newly captured, person-tagged photos and delivers them to
// that person's notification contacts.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/wrcrooks/JiboConnector/internal/config"
	"github.com/wrcrooks/JiboConnector/internal/jiboapi"
	"github.com/wrcrooks/JiboConnector/internal/notify"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	client := jiboapi.NewClient(cfg.JiboAPIBaseURL)
	notifier := notify.NoopNotifier{Logger: logger}

	healthServer := &http.Server{
		Addr:    cfg.HealthAddr,
		Handler: healthMux(),
	}

	go func() {
		logger.Info("health server listening", "addr", cfg.HealthAddr)
		if err := healthServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("health server failed", "error", err)
		}
	}()

	logger.Info("starting poll loop",
		"jiboApiBaseUrl", cfg.JiboAPIBaseURL,
		"pollInterval", cfg.PollInterval)
	runPollLoop(ctx, logger, cfg.PollInterval, client, notifier)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := healthServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("health server shutdown failed", "error", err)
	}
}

func healthMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	return mux
}

func runPollLoop(
	ctx context.Context,
	logger *slog.Logger,
	interval time.Duration,
	client *jiboapi.Client,
	notifier notify.Notifier,
) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("poll loop stopping")
			return
		case <-ticker.C:
			pollOnce(ctx, logger, client, notifier)
		}
	}
}

func pollOnce(ctx context.Context, logger *slog.Logger, client *jiboapi.Client, notifier notify.Notifier) {
	// jiboapi.Client's list/lookup methods aren't implemented yet — see
	// internal/jiboapi/client.go for what's still open.
	logger.Info("poll tick (not yet wired to jibo-api)")
	_ = ctx
	_ = client
	_ = notifier
}
