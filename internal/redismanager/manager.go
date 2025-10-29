package redismanager

import (
	"context"
	"crypto/sha1"
	"encoding/base64"
	"math/rand"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type Manager struct {
	client redis.UniversalClient
	// probably conf for ttl
}

// Create Redis instance
func NewManager(redisClient redis.UniversalClient) *Manager {
	return &Manager{
		client: redisClient,
	}
}

func (m *Manager) Create(ctx context.Context, imageKey string, ttl int) (string, error) {
	hash := GenerateHash()
	dur, err := time.ParseDuration(strconv.Itoa(ttl) + "s")
	if err != nil {

		return "", err
	}

	err = m.client.Set(ctx, "MH:Image:"+hash, imageKey, dur).Err()
	if err != nil {
		return "", err
	}

	return hash, nil
}

func GenerateHash() string {
	src := rand.NewSource(time.Now().UnixNano() * 2)
	r := rand.New(src)

	str := strconv.Itoa(int(time.Now().UnixNano()))
	str += strconv.Itoa(r.Intn(65535))

	in := sha1.Sum([]byte(str))

	out := make([]byte, base64.StdEncoding.EncodedLen(len(in)))
	base64.StdEncoding.Encode(out, in[:])

	return string(out)
}
