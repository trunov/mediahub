package cache

import (
	"context"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type Cache struct {
	Redis     redis.UniversalClient
	Namespace string
}

// Get value from Redis
func (c *Cache) Get(ctx context.Context, key string) (interface{}, error) {
	cmd := c.Redis.Get(ctx, c.Namespace+":"+key)
	return cmd.Val(), cmd.Err()
}

// Store data to Redis
func (c *Cache) Store(ctx context.Context, key string, ttl int, value interface{}) error {
	dur, err := time.ParseDuration(strconv.Itoa(ttl) + "s")
	if err != nil {
		return err
	}

	cmd := c.Redis.Set(ctx, c.Namespace+":"+key, value, dur)
	return cmd.Err()
}

func (c *Cache) Flush(ctx context.Context) error {
	keys := c.Redis.Keys(ctx, c.Namespace+":*")
	//using pipeline to delete keys efficiently
	pl := c.Redis.Pipeline()

	for _, key := range keys.Val() {
		pl.Del(ctx, key)
	}

	_, err := pl.Exec(ctx)
	return err
}

// Delete key from Redis
func (c *Cache) Remove(ctx context.Context, key string) error {
	cmd := c.Redis.Del(ctx, c.Namespace+":"+key)
	return cmd.Err()
}

// Create Redis connection
func NewCache(namespace string, redisCl redis.UniversalClient) *Cache {
	return &Cache{
		Namespace: namespace,
		Redis:     redisCl,
	}
}
