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

	if cfg.JiboAPIKey == "" {
		logger.Warn("JIBOCONNECTOR_JIBO_API_KEY is not set — every jibo-api call will be rejected with 401")
	}
	client := jiboapi.NewClient(cfg.JiboAPIBaseURL, cfg.JiboAPIKey)

	var channels []notify.Notifier

	if cfg.SESFromAddress != "" {
		sesNotifier, err := notify.NewSESNotifier(ctx, cfg.AWSRegion, cfg.SESFromAddress)
		if err != nil {
			logger.Error("failed to set up SES notifier", "error", err)
			os.Exit(1)
		}
		channels = append(channels, sesNotifier)
		logger.Info("email notifications enabled via SES", "region", cfg.AWSRegion, "fromAddress", cfg.SESFromAddress)
	} else {
		logger.Warn("JIBOCONNECTOR_SES_FROM_ADDRESS is not set — email notifications disabled")
	}

	if cfg.SMSEnabled {
		snsNotifier, err := notify.NewSNSNotifier(ctx, cfg.AWSRegion)
		if err != nil {
			logger.Error("failed to set up SNS notifier", "error", err)
			os.Exit(1)
		}
		channels = append(channels, snsNotifier)
		logger.Info("SMS notifications enabled via SNS", "region", cfg.AWSRegion)
	} else {
		logger.Warn("JIBOCONNECTOR_SMS_ENABLED is not set — SMS notifications disabled")
	}

	if len(channels) == 0 {
		channels = append(channels, notify.NoopNotifier{Logger: logger})
		logger.Warn("no notification channels configured — using no-op notifier (logs only, sends nothing)")
	}
	notifier := notify.NewComposite(channels...)

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

	// Start from "now" rather than the beginning of time — on a fresh
	// deploy this avoids notifying every contact about the entire existing
	// photo backlog in one burst. This cursor lives only in memory: a
	// restart re-checks from "now" again, so a photo captured in the
	// seconds around a restart could in principle be missed. Durable
	// tracking across restarts is a known open gap (see README).
	lastSeenUnixMs := time.Now().UnixMilli()

	for {
		select {
		case <-ctx.Done():
			logger.Info("poll loop stopping")
			return
		case <-ticker.C:
			lastSeenUnixMs = pollOnce(ctx, logger, client, notifier, lastSeenUnixMs)
		}
	}
}

func pollOnce(
	ctx context.Context,
	logger *slog.Logger,
	client *jiboapi.Client,
	notifier notify.Notifier,
	sinceUnixMs int64,
) int64 {
	media, err := client.ListRecentPersonTaggedMedia(ctx, sinceUnixMs)
	if err != nil {
		logger.Error("failed to list recent person-tagged media", "error", err)
		return sinceUnixMs
	}

	nextCursor := sinceUnixMs
	for _, item := range media {
		if item.CreatedUnix > nextCursor {
			nextCursor = item.CreatedUnix
		}

		contacts, err := client.PhotoContactsForPerson(ctx, item.PersonID)
		if err != nil {
			logger.Error("failed to look up photo contacts",
				"personId", item.PersonID, "mediaPath", item.Path, "error", err)
			continue
		}

		if len(contacts) == 0 {
			logger.Info("no photo notification contacts configured for person",
				"personId", item.PersonID, "mediaPath", item.Path)
			continue
		}

		for _, contact := range contacts {
			if err := notifier.Deliver(ctx, item, contact); err != nil {
				logger.Error("failed to deliver photo notification",
					"mediaPath", item.Path, "contactId", contact.ID, "error", err)
				continue
			}
			logger.Info("delivered photo notification",
				"mediaPath", item.Path, "personId", item.PersonID, "contactId", contact.ID)
		}
	}

	return nextCursor
}
