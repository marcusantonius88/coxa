package infra

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisClient encapsula operações com Redis para idempotência
type RedisClient struct {
	client *redis.Client
}

// NewRedisClient cria um novo cliente Redis
func NewRedisClient(addr string) *RedisClient {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: "",
		DB:       0,
	})
	return &RedisClient{client: client}
}

// SetProcessed marca um evento como processado (para idempotência)
func (rc *RedisClient) SetProcessed(ctx context.Context, eventID string, ttl time.Duration) error {
	err := rc.client.Set(ctx, fmt.Sprintf("processed:%s", eventID), "1", ttl).Err()
	if err != nil {
		return fmt.Errorf("erro ao marcar evento como processado: %w", err)
	}
	return nil
}

// IsProcessed verifica se um evento já foi processado
func (rc *RedisClient) IsProcessed(ctx context.Context, eventID string) (bool, error) {
	val, err := rc.client.Get(ctx, fmt.Sprintf("processed:%s", eventID)).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("erro ao verificar se evento foi processado: %w", err)
	}
	return val == "1", nil
}

// Close fecha a conexão com Redis
func (rc *RedisClient) Close() error {
	return rc.client.Close()
}
