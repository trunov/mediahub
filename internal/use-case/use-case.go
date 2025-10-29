package use_case

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"

	"github.com/trunov/mediahub/internal/entities"
	"github.com/trunov/mediahub/internal/processor"
	"github.com/trunov/mediahub/internal/transport/handler"
)

type Storage interface {
	InsertImage(ctx context.Context, fh *multipart.FileHeader, imageParams handler.UploadImageParams) (entities.Image, error)
}

type RedisStore interface {
	Create(ctx context.Context, imageKey string, ttl int) (string, error)
}

type R2Storage interface {
	Upload(ctx context.Context, key string, ext string, payload []byte) error
}

type useCase struct {
	storage      Storage
	redismanager RedisStore
	r2Storage    R2Storage
}

func New(storage Storage, rm RedisStore, r2Storage R2Storage) *useCase {
	return &useCase{
		storage:      storage,
		redismanager: rm,
		r2Storage:    r2Storage,
	}
}

func (c *useCase) UploadImage(ctx context.Context, file multipart.File, fh *multipart.FileHeader, ext string, fileType string, imageParams handler.UploadImageParams) (entities.Image, error) {
	img := entities.Image{}

	originalData, width, height, err := processImage(file, ext)
	if err != nil {
		return img, fmt.Errorf("error processing image: %v", err)
	}

	fmt.Println(width)
	fmt.Println(height)

	err = c.r2Storage.Upload(ctx, "sample3", fileType, originalData)
	if err != nil {
		return img, err
	}

	return img, nil
}

func processImage(file multipart.File, ext string) ([]byte, int, int, error) {
	imgp := &processor.ImageProcessor{}
	b, err := io.ReadAll(file)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to read image: %w", err)
	}

	if err := loadImage(ext, imgp, bytes.NewReader(b)); err != nil {
		return nil, 0, 0, err
	}

	width, height := imgp.GetBounds()

	return b, width, height, nil
}

func loadImage(ext string, imgp *processor.ImageProcessor, r io.Reader) error {
	switch ext {
	case ".png":
		return imgp.LoadPNG(r)
	case ".jpg", ".jpeg":
		return imgp.LoadJPEG(r)
	case ".webp":
		return imgp.LoadWEBP(r)
	default:
		return fmt.Errorf("unsupported image extension: %s", ext)
	}
}
