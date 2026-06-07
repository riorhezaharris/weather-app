package cache

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/riorhezaharris/weather-app/internal/model"
)

var ErrCacheMiss = errors.New("cache miss")

type RedisCache struct {
	client *redis.Client
	ttl    time.Duration
}

func NewRedisCache(addr string, ttl time.Duration) *RedisCache {
	return &RedisCache{
		client: redis.NewClient(&redis.Options{Addr: addr}),
		ttl:    ttl,
	}
}

func (c *RedisCache) Get(ctx context.Context, city string) (*model.WeatherRecord, error) {
	val, err := c.client.Get(ctx, city).Result()
	if errors.Is(err, redis.Nil) {
		return nil, ErrCacheMiss
	}
	if err != nil {
		return nil, err
	}
	var record model.WeatherRecord
	if err := json.Unmarshal([]byte(val), &record); err != nil {
		return nil, err
	}
	return &record, nil
}

func (c *RedisCache) Set(ctx context.Context, record *model.WeatherRecord) error {
	data, err := json.Marshal(record)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, record.City, data, c.ttl).Err()
}

func (c *RedisCache) Close() error {
	return c.client.Close()
}
