package app

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/trunov/mediahub/cmd/migrate"
	"github.com/trunov/mediahub/internal/cache"
	"github.com/trunov/mediahub/internal/config"
	"github.com/trunov/mediahub/internal/r2"
	"github.com/trunov/mediahub/internal/redisholder"
	"github.com/trunov/mediahub/internal/redismanager"
	"github.com/trunov/mediahub/internal/repository/storage"
	"github.com/trunov/mediahub/internal/transport/handler"
	"github.com/trunov/mediahub/internal/transport/router"
	use_case "github.com/trunov/mediahub/internal/use-case"
)

type App struct {
	HttpServer *http.Server
}

func New(cfg *config.Config) (*App, error) {
	ctx := context.Background()

	err := migrate.Migrate(cfg.Database.DSN, migrate.Migrations)
	if err != nil {
		return nil, err
	}

	repo, err := storage.New(ctx, cfg.Database.DSN)
	if err != nil {
		return nil, err
	}

	holder, err := redisholder.Build(ctx, cfg)
	if err != nil {
		log.Fatal(err)
	}

	rc := holder.Get()
	rm := redismanager.NewManager(rc)

	redisCache := cache.NewCache("mediahub:images", rc)

	r2Storage := r2.NewStorage(&cfg.R2, redisCache)

	uc := use_case.New(repo, rm, r2Storage)

	h := handler.New(uc, cfg)
	r := router.NewRouter(h)

	s := &http.Server{
		Handler: r,
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
	}

	return &App{
		HttpServer: s,
	}, nil
}

func (a *App) Run() error {
	log.Printf("starting server")
	return a.HttpServer.ListenAndServe()
}
