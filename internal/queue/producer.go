package queue

import (
	"context"
	"encoding/json"

	"github.com/redis/go-redis/v9"
)

type Producer struct {
	r      redis.UniversalClient
	stream string
	maxLen int64
}

func NewProducer(r redis.UniversalClient, stream string, maxLen int64) *Producer {
	return &Producer{r: r, stream: stream, maxLen: maxLen}
}

// Encodes it as JSON and appends it to a Redis Stream
// Persist the conversion request for background processing
func (p *Producer) EnqueueConvert(ctx context.Context, job ConvertJob) error {
	raw, _ := json.Marshal(job)
	return p.r.XAdd(ctx, &redis.XAddArgs{
		Stream: p.stream,
		MaxLen: p.maxLen,
		Values: map[string]any{
			"payload": string(raw),
			"attempt": 0,
		},
	}).Err()
}
