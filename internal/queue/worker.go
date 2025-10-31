package queue

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/trunov/mediahub/internal/config"
	webp_converter "github.com/trunov/mediahub/internal/webp-converter"
)

type Storage interface {
	Download(ctx context.Context, key string) ([]byte, string, error)
	UploadWithHook(ctx context.Context, key, contentType string, payload []byte, onSuccess func()) error
}

type WebPConverter interface {
	ToWebP(reader io.Reader, ext string) ([]byte, error)
}

type Worker struct {
	rc      redis.UniversalClient
	cfg     config.WebPWorkerConfig
	storage Storage
	conv    WebPConverter
}

func Init(ctx context.Context, rc redis.UniversalClient, cfg config.WebPWorkerConfig, r2Storage Storage) *Producer {
	producer := NewProducer(rc, cfg.Stream, cfg.MaxLen)
	worker := NewWorker(rc, cfg, r2Storage)

	go func() {
		if err := worker.Start(ctx); err != nil {
			log.Printf("[webp-worker] stopped: %v", err)
		}
	}()

	return producer
}

func NewWorker(rc redis.UniversalClient, cfg config.WebPWorkerConfig, storage Storage) *Worker {
	return &Worker{
		rc:      rc,
		cfg:     cfg,
		storage: storage,
		conv:    webp_converter.Converter{},
	}
}

func (w *Worker) EnsureGroup(ctx context.Context) error {
	// Without MkStream, Redis would error out if you try to create a group before any messages exist in the stream.
	err := w.rc.XGroupCreateMkStream(ctx, w.cfg.Stream, w.cfg.Group, "0").Err()
	// Redis returns BUSYGROUP if the group already exists therefore we check for other errors
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		return err
	}
	return nil
}

func (w *Worker) Start(ctx context.Context) error {
	if err := w.EnsureGroup(ctx); err != nil {
		return fmt.Errorf("failed to ensure Redis group: %w", err)
	}

	log.Printf("[webp-worker] starting consumer group=%s stream=%s workers=%d",
		w.cfg.Group, w.cfg.Stream, w.cfg.Workers,
	)

	// Adopt orphaned pending messages
	w.autoClaim(ctx)
	log.Printf("[webp-worker] auto-claim complete, entering loop...")

	errCh := make(chan error, w.cfg.Workers)
	for i := 0; i < w.cfg.Workers; i++ {
		id := i
		go func() {
			log.Printf("[webp-worker] worker #%d started", id)
			err := w.loop(ctx)
			if err != nil {
				log.Printf("[webp-worker] worker #%d stopped with error: %v", id, err)
			} else {
				log.Printf("[webp-worker] worker #%d stopped gracefully", id)
			}
			errCh <- err
		}()
	}

	select {
	case <-ctx.Done():
		log.Printf("[webp-worker] context canceled, stopping all workers")
		return ctx.Err()
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("worker loop exited with error: %w", err)
		}
		return nil
	}
}

// autoClaim scans the Redis Stream's consumer group for "stuck" messages
// that were previously delivered to other consumers but never acknowledged.
// This can happen if a worker crashes or is killed before XACK.
// Using XAUTOCLAIM, we take ownership of those idle messages so they can be retried.
//
// This ensures that incomplete jobs are not lost and will eventually
// be picked up again after a restart or worker failure.
func (w *Worker) autoClaim(ctx context.Context) {
	next := "0-0"

	// Determine how long a message must have been idle before we reclaim it.
	// Default to 30 seconds minimum; increase proportionally to the block timeout
	// (so we don't steal messages still being processed by slow workers).
	minIdle := 30 * time.Second
	if w.cfg.BlockTimeout > 0 {
		t := w.cfg.BlockTimeout * 6
		if t > minIdle {
			minIdle = t
		}
	}

	for {
		// Try to claim up to 100 idle messages from other consumers
		// in the same group that have been pending longer than minIdle.
		msgs, start, err := w.rc.XAutoClaim(ctx, &redis.XAutoClaimArgs{
			Stream:   w.cfg.Stream,
			Group:    w.cfg.Group,
			Consumer: w.cfg.Consumer,
			MinIdle:  minIdle,
			Start:    next,
			Count:    100,
		}).Result()
		if err != nil || len(msgs) == 0 {
			return
		}
		next = start
	}
}

func (w *Worker) loop(ctx context.Context) error {
	for {
		// XREADGROUP is where the actual "delivery" happens.
		// It reads messages from the Redis Stream as part of the specified consumer group.
		//
		// When Redis executes XREADGROUP GROUP <group> <consumer> STREAMS <stream> > :
		//   1. It finds new (undelivered) messages in <stream>.
		//   2. Marks them as *pending* for this consumer (adds to the group's PEL - Pending Entries List).
		//   3. Returns them to this worker for processing.
		//
		// The message stays in the PEL until we explicitly acknowledge it with XACK,
		// which happens at the end of handle() via a deferred call.
		//
		// If the worker crashes before XACK, the message remains pending and
		// will later be reclaimed by autoClaim() on startup.
		streams, err := w.rc.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    w.cfg.Group,
			Consumer: w.cfg.Consumer,
			Streams:  []string{w.cfg.Stream, ">"},
			Count:    1,
			Block:    w.cfg.BlockTimeout,
		}).Result()
		if err != nil && err != redis.Nil {
			if ctx.Err() != nil {
				return nil
			}
			continue
		}
		for _, s := range streams {
			for _, m := range s.Messages {
				// should report error to sentry
				_ = w.handle(ctx, m)
			}
		}
	}
}

func (w *Worker) handle(ctx context.Context, m redis.XMessage) error {
	defer w.rc.XAck(ctx, w.cfg.Stream, w.cfg.Group, m.ID).Err()

	raw, ok := m.Values["payload"].(string)
	if !ok {
		// add sentry error handling
		return nil
	}
	var job ConvertJob
	if err := json.Unmarshal([]byte(raw), &job); err != nil {
		// add sentry error handling
		return nil
	}
	attempt := toInt(m.Values["attempt"])

	if err := w.process(ctx, job); err != nil {
		if attempt+1 >= w.cfg.MaxAttempts {
			// add sentry error handling
			return nil
		}
		// simple exponential backoff requeue
		backoff := w.cfg.BackoffBase << attempt
		time.AfterFunc(backoff, func() {
			_ = w.rc.XAdd(context.Background(), &redis.XAddArgs{
				Stream: w.cfg.Stream,
				MaxLen: w.cfg.MaxLen,
				Values: map[string]any{
					"payload": raw,
					"attempt": attempt + 1,
				},
			}).Err()
		})
		return err
	}
	return nil
}

func (w *Worker) process(ctx context.Context, job ConvertJob) error {
	orig, _, err := w.storage.Download(ctx, job.ObjectKey)
	if err != nil {
		return fmt.Errorf("download %s: %w", job.ObjectKey, err)
	}

	ext := strings.ToLower(job.Ext)
	webpBytes, err := w.conv.ToWebP(bytes.NewReader(orig), ext)
	if err != nil {
		return fmt.Errorf("convert to webp: %w", err)
	}

	target := job.WebPKey
	if target == "" {
		target = job.ObjectKey + ".webp"
	}

	if err := w.storage.UploadWithHook(ctx, target, "image/webp", webpBytes, nil); err != nil {
		return fmt.Errorf("upload webp: %w", err)
	}
	return nil
}

func toInt(v any) int {
	switch t := v.(type) {
	case int:
		return t
	case int64:
		return int(t)
	case string:
		var x int
		fmt.Sscanf(t, "%d", &x)
		return x
	default:
		return 0
	}
}
