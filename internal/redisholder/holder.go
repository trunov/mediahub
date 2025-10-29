package redisholder

import (
	"sync/atomic"

	"github.com/redis/go-redis/v9"
)

type Holder struct {
	v atomic.Value // stores redis.UniversalClient
}

func NewHolder(initial redis.UniversalClient) *Holder {
	h := &Holder{}
	h.v.Store(initial)
	return h
}

func (h *Holder) Get() redis.UniversalClient {
	c, _ := h.v.Load().(redis.UniversalClient)
	return c
}

func (h *Holder) swap(newc redis.UniversalClient) (old redis.UniversalClient) {
	old, _ = h.v.Load().(redis.UniversalClient)
	h.v.Store(newc)
	return old
}

func (h *Holder) Close() error {
	if c := h.Get(); c != nil {
		return c.Close()
	}
	return nil
}
