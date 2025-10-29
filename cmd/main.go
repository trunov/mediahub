package main

import (
	"log"

	"github.com/trunov/mediahub/internal/app"
	"github.com/trunov/mediahub/internal/config"
)

const file = "config.json"

func main() {
	cfg := config.NewConfig()
	err := cfg.Read(file)
	if err != nil {
		log.Fatal(err)
	}

	app, err := app.New(cfg)
	if err != nil {
		log.Fatal(err)
	}

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
