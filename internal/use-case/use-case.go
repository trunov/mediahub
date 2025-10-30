package use_case

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"strings"

	"github.com/trunov/mediahub/internal/entities"
	"github.com/trunov/mediahub/internal/processor"
	"github.com/trunov/mediahub/internal/queue"
	"github.com/trunov/mediahub/internal/transport/handler"
)

type Storage interface {
	InsertImage(ctx context.Context, fh *multipart.FileHeader, imageParams handler.UploadImageParams) (entities.Image, error)
}

type RedisStore interface {
	Create(ctx context.Context, imageKey string, ttl int) (string, error)
}

type R2Storage interface {
	UploadWithHook(ctx context.Context, key string, ext string, payload []byte, onSuccess func()) error
}

type useCase struct {
	storage      Storage
	redismanager RedisStore
	r2Storage    R2Storage
	wqueue       *queue.Producer
}

func New(storage Storage, rm RedisStore, r2Storage R2Storage, wqueue *queue.Producer) *useCase {
	return &useCase{
		storage:      storage,
		redismanager: rm,
		r2Storage:    r2Storage,
		wqueue:       wqueue,
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

	key := "pro_test"

	err = c.r2Storage.UploadWithHook(ctx, key, fileType, originalData, func() {
		c.wqueue.EnqueueConvert(ctx, queue.ConvertJob{
			ObjectKey:   key,
			ContentType: fileType,
			Ext:         strings.ToLower(ext),
			// WebPKey:   optional override; default is objectKey + ".webp"
		})
	})
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
