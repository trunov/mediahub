package redisholder

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/trunov/mediahub/internal/config"
)

func Build(ctx context.Context, cfg *config.Config) (*Holder, error) {
	var cl redis.UniversalClient
	cl, err := newClusterClient(&cfg.Redis)
	if err != nil {
		clusterErr := err
		cl, err = newClient(&cfg.Redis)
		if err != nil {
			return nil, fmt.Errorf("create redis client: %w", err)
		}
		log.Printf("redis: cluster client failed (%v); using single-node client", clusterErr)
	}

	h := NewHolder(cl)

	go healthLoop(ctx, h, cfg)

	return h, nil
}

func healthLoop(ctx context.Context, h *Holder, cfg *config.Config) {
	log.Printf("redis: health loop started (interval=%v)", cfg.Redis.HealthCheckInterval*time.Second)

	ping := func() {
		pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		err := h.Get().Ping(pingCtx).Err()
		cancel()

		if err == nil {
			log.Printf("redis: all good with redis")
			return
		}
		log.Printf("redis: ping failed (%v); attempting reconnect…", err)

		var newCl redis.UniversalClient
		var newErr error
		// Rebuild client (cluster first, then fallback)
		newCl, newErr = newClusterClient(&cfg.Redis)
		if newErr != nil {
			newCl, newErr = newClient(&cfg.Redis)
		}
		if newErr != nil {
			log.Printf("redis: reconnect failed: %v", newErr)
			return
		}

		old := h.swap(newCl)
		if old != nil {
			_ = old.Close()
		}
		log.Printf("redis: reconnected successfully")
	}

	ping()

	t := time.NewTicker(cfg.Redis.HealthCheckInterval * time.Second)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			_ = h.Close()
			log.Printf("redis: health loop stopped (%v)", ctx.Err())
			return
		case <-t.C:
			ping()
		}
	}
}

func newClusterClient(cfg *config.RedisConfig) (*redis.ClusterClient, error) {
	if len(cfg.Nodes) < 1 {
		return nil, errors.New("no nodes defined")
	}

	nodeAddrs := make([]string, 0)

	for _, node := range cfg.Nodes {
		nodeAddrs = append(nodeAddrs, node.Addr())
	}

	cl := redis.NewClusterClient(&redis.ClusterOptions{
		RouteByLatency: true,
		Password:       cfg.Password,
		Addrs:          nodeAddrs,
		DialTimeout:    cfg.DialTimeout * time.Second,
		ReadTimeout:    cfg.ReadTimeout * time.Second,
		WriteTimeout:   cfg.WriteTimeout * time.Second,
		PoolSize:       20,
		PoolTimeout:    time.Duration(30) * time.Second,
		MaxRetries:     30,
	})

	err := cl.Ping(context.Background()).Err()
	if err != nil {
		return nil, fmt.Errorf("error pinging redis cluster: %w", err)
	}

	return cl, nil
}

func newClient(cfg *config.RedisConfig) (*redis.Client, error) {
	var stickyErr = errors.New("no nodes defined")

	for _, node := range cfg.Nodes {
		cl := redis.NewClient(&redis.Options{
			Addr:         node.Addr(),
			Password:     cfg.Password,
			DB:           cfg.DatabaseID,
			DialTimeout:  cfg.DialTimeout * time.Second,
			ReadTimeout:  cfg.ReadTimeout * time.Second,
			WriteTimeout: cfg.WriteTimeout * time.Second,
		})

		err := cl.Ping(context.Background()).Err()
		if err != nil {
			stickyErr = fmt.Errorf("error pinging redis server: %w", err)
			continue
		}

		return cl, nil
	}

	return nil, stickyErr
}
