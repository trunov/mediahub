package storage

import (
	"context"
	"fmt"
	"mime/multipart"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/trunov/mediahub/internal/entities"
	"github.com/trunov/mediahub/internal/transport/handler"
)

type dbStorage struct {
	dbpool *pgxpool.Pool
}

func New(ctx context.Context, databaseDSN string) (*dbStorage, error) {
	pool, err := pgxpool.New(ctx, databaseDSN)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	return &dbStorage{dbpool: pool}, nil
}

func (s *dbStorage) Ping(ctx context.Context) error {
	err := s.dbpool.Ping(ctx)

	if err != nil {
		return err
	}
	return nil
}

func (s *dbStorage) InsertImage(ctx context.Context, fh *multipart.FileHeader, imageParams handler.UploadImageParams) (entities.Image, error) {
	return entities.Image{}, nil
}
