package main

import (
	"log"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/trunov/mediahub/internal/app"
	"github.com/trunov/mediahub/internal/config"
)

const file = "config.json"

func initSentry(cfg *config.SentryConfig, version string) error {
	return sentry.Init(sentry.ClientOptions{
		Dsn:         cfg.SentryDSN,
		Environment: cfg.Environment,
		Release:     version,
	})
}

func main() {
	cfg := config.NewConfig()
	err := cfg.Read(file)
	if err != nil {
		log.Fatal(err)
	}

	err = initSentry(&cfg.Sentry, "v1")
	if err != nil {
		log.Fatalf("sentry.Init: %s", err)
	}

	// Flush buffered events before the program terminates.
	defer sentry.Flush(2 * time.Second)

	app, err := app.New(cfg)
	if err != nil {
		log.Fatal(err)
	}

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
