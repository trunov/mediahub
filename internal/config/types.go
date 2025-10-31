package config

import (
	"fmt"
	"time"
)

type Config struct {
	Server   ServerConfig     `json:"server"`
	Upload   UploadConfig     `json:"upload"`
	Database Database         `json:"database"`
	Redis    RedisConfig      `json:"redis"`
	R2       R2Config         `json:"r2"`
	WebP     WebPWorkerConfig `json:"webp_worker"`
	Sentry   SentryConfig     `json:"sentry"`
}

type ServerConfig struct {
	Port         int           `json:"port"`
	ReadTimeout  time.Duration `json:"read_timeout"`
	WriteTimeout time.Duration `json:"write_timeout"`
}

type UploadConfig struct {
	MaxRequestBodyMB     int64 `json:"max_request_body"`
	MaxMultipartMemoryMB int64 `json:"max_multipart_memory"`
}

type Database struct {
	DSN string `json:"dsn"`
}

type RedisConfig struct {
	Password            string        `json:"password"`
	DatabaseID          int           `json:"database_id"`
	HealthCheckInterval time.Duration `json:"health_check_interval"`
	DialTimeout         time.Duration `json:"dial_timeout"`
	ReadTimeout         time.Duration `json:"read_timeout"`
	WriteTimeout        time.Duration `json:"write_timeout"`
	PoolSize            int           `json:"pool_size"`
	Nodes               []RedisNode   `json:"nodes"`
}

type RedisNode struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

func (n RedisNode) Addr() string { return fmt.Sprintf("%s:%d", n.Host, n.Port) }

type R2Config struct {
	AccountID   string `json:"account_id"`
	BucketName  string `json:"bucket_name"`
	AccessKeyID string `json:"access_key_id"`
	SecretKey   string `json:"secret_key"`
	Endpoint    string `json:"endpoint"`
}

type WebPWorkerConfig struct {
	Stream       string        `json:"stream"`        // redis stream name
	Group        string        `json:"group"`         // consumer group name
	Workers      int           `json:"workers"`       // number of concurrent goroutines
	MaxAttempts  int           `json:"max_attempts"`  // max retries before DLQ
	MaxLen       int64         `json:"max_len"`       // stream max length before trim
	BackoffBase  time.Duration `json:"backoff_base"`  // base retry delay
	BlockTimeout time.Duration `json:"block_timeout"` // XREADGROUP block timeout
	Consumer     string        `json:"consumer"`
}

type SentryConfig struct {
	SentryDSN   string `json:"sentry_dsn"`
	Environment string `json:"environment"`
}
