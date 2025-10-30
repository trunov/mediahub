package r2

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/trunov/mediahub/internal/cache"
	conf "github.com/trunov/mediahub/internal/config"
)

var ErrQueueFull = errors.New("upload queue is full")

type uploadReq struct {
	ctx      context.Context
	key      string
	fileType string
	payload  []byte

	onSuccess func()
}

type S3 struct {
	AccountID          string
	Bucket             string
	Region             string // usually "auto" for R2
	AwsAccessKeyId     string
	AwsSecretAccessKey string

	Workers        int
	QueueSize      int
	MaxRetries     int
	RetryBaseDelay time.Duration

	queue chan uploadReq
	wg    sync.WaitGroup

	S3Client *s3.Client
	Uploader *manager.Uploader

	Cache *cache.Cache
}

func NewStorage(cfg *conf.R2Config, redisCache *cache.Cache) *S3 {
	r2c := &S3{
		AccountID:          cfg.AccountID,
		Bucket:             cfg.BucketName,
		Region:             "auto",
		AwsAccessKeyId:     cfg.AccessKeyID,
		AwsSecretAccessKey: cfg.SecretKey,
		Workers:            8,
		QueueSize:          1000,
		MaxRetries:         3,
		RetryBaseDelay:     300 * time.Millisecond,
		Cache:              redisCache,
	}
	if err := r2c.Run(); err != nil {
		log.Fatal(err)
	}

	return r2c
}
func (s *S3) Run() error {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			s.AwsAccessKeyId, s.AwsSecretAccessKey, "",
		)),
		config.WithRegion(s.Region),
	)
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	s.S3Client = s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(fmt.Sprintf("https://%s.r2.cloudflarestorage.com", s.AccountID))
		o.UsePathStyle = true
	})
	s.Uploader = manager.NewUploader(s.S3Client)

	s.queue = make(chan uploadReq, s.QueueSize)
	for i := 0; i < s.Workers; i++ {
		s.wg.Add(1)
		go s.worker()
	}

	log.Println("✅ R2 client + worker pool initialized.")
	return nil
}

// Close waits for all queued tasks to be processed.
func (s *S3) Close() {
	close(s.queue)
	s.wg.Wait()
}

// Upload tries to put an upload on the queue without blocking.
// If the queue is full, it returns ErrQueueFull immediately.
func (s *S3) UploadWithHook(ctx context.Context, key string, fileType string, payload []byte, onSuccess func()) error {
	req := uploadReq{ctx: ctx, key: key, fileType: fileType, payload: payload, onSuccess: onSuccess}
	select {
	case s.queue <- req:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return ErrQueueFull
	}
}

func (s *S3) worker() {
	defer s.wg.Done()
	for req := range s.queue {
		var err error
		attempt := 0

		for {
			attempt++
			_, err = s.Uploader.Upload(req.ctx, &s3.PutObjectInput{
				Bucket:      aws.String(s.Bucket),
				Key:         aws.String(req.key),
				Body:        bytes.NewReader(req.payload),
				ContentType: aws.String(req.fileType),
			})
			if err == nil {
				if req.onSuccess != nil {
					req.onSuccess() // cheap enough so synchronous
				}
				break
			}

			// retry?
			if attempt > s.MaxRetries {
				break
			}

			// backoff with jitter
			backoff := s.backoffDelay(attempt)
			timer := time.NewTimer(backoff)
			select {
			case <-timer.C:
			case <-req.ctx.Done():
				timer.Stop()
			}
			if req.ctx != nil && req.ctx.Err() != nil {
				break
			}
		}

	}
}

func (s *S3) backoffDelay(attempt int) time.Duration {
	delay := s.RetryBaseDelay << (attempt - 1)
	jitter := time.Duration(int64(delay) / 10)
	return delay - (jitter / 2) + time.Duration(int64(jitter)*time.Now().UnixNano()%2)
}

func (s *S3) Download(ctx context.Context, key string) ([]byte, string, error) {
	out, err := s.S3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, "", fmt.Errorf("failed to download %q: %w", key, err)
	}
	defer out.Body.Close()

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(out.Body); err != nil {
		return nil, "", fmt.Errorf("failed to read body for %q: %w", key, err)
	}

	return buf.Bytes(), *out.ContentType, nil
}
