// Package config loads JiboConnector's runtime settings from environment
// variables, following the JIBOCONNECTOR_ prefix convention (mirroring
// jibo-api's own OPENJIBO_ prefix) so both services' env files read
// consistently side by side in docker-compose.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	// JiboAPIBaseURL is the base URL of the jibo-api server this worker polls,
	// e.g. http://api:8080 inside the openjibo docker-compose network.
	JiboAPIBaseURL string

	// JiboAPIKey authenticates this worker to jibo-api's /api/connector/*
	// endpoints (OpenJibo:Connector:ApiKey on the jibo-api side). Required —
	// those endpoints reject every request without a matching bearer token.
	JiboAPIKey string

	// PollInterval controls how often the worker checks jibo-api for newly
	// captured, person-tagged photos.
	PollInterval time.Duration

	// HealthAddr is the address the worker's health-check HTTP server binds to.
	HealthAddr string

	// AWSRegion is the region SES sends from, e.g. us-east-1.
	AWSRegion string

	// SESFromAddress is the SES-verified sender address for outgoing photo
	// notification emails. If empty, email delivery is skipped.
	SESFromAddress string

	// SMSEnabled turns on SMS delivery via Amazon SNS. Unlike SES there's no
	// per-service required value (like a verified sender) to gate on, so
	// this is an explicit opt-in rather than inferred from AWS credentials
	// being present — being able to send email shouldn't silently imply
	// being willing to spend money on SMS.
	SMSEnabled bool
}

func Load() (Config, error) {
	cfg := Config{
		JiboAPIBaseURL: getEnv("JIBOCONNECTOR_JIBO_API_BASE_URL", "http://api:8080"),
		JiboAPIKey:     getEnv("JIBOCONNECTOR_JIBO_API_KEY", ""),
		HealthAddr:     getEnv("JIBOCONNECTOR_HEALTH_ADDR", ":8090"),
		AWSRegion:      getEnv("JIBOCONNECTOR_AWS_REGION", "us-east-1"),
		SESFromAddress: getEnv("JIBOCONNECTOR_SES_FROM_ADDRESS", ""),
	}

	pollSeconds, err := strconv.Atoi(getEnv("JIBOCONNECTOR_POLL_INTERVAL_SECONDS", "30"))
	if err != nil {
		return Config{}, fmt.Errorf("parsing JIBOCONNECTOR_POLL_INTERVAL_SECONDS: %w", err)
	}
	cfg.PollInterval = time.Duration(pollSeconds) * time.Second

	smsEnabled, err := strconv.ParseBool(getEnv("JIBOCONNECTOR_SMS_ENABLED", "false"))
	if err != nil {
		return Config{}, fmt.Errorf("parsing JIBOCONNECTOR_SMS_ENABLED: %w", err)
	}
	cfg.SMSEnabled = smsEnabled

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return fallback
}
