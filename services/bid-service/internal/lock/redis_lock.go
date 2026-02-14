package lock

import (
	"context"
	"fmt"
	"time"

	"github.com/Osireg17/AI-Bidding-Platform/services/bid-service/internal/domain"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// RedisLockManager implements domain.LockManager using Redis SET NX EX.
type RedisLockManager struct {
	client *redis.Client
	logger *zap.Logger
}

func NewRedisLockManager(redisURL string, logger *zap.Logger) (*RedisLockManager, error) {
	url, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	client := redis.NewClient(url)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logger.Info("Redis lock manager initialized")

	return &RedisLockManager{
		client,
		logger,
	}, nil
}

// AcquireLock sets a Redis key with NX (only if not exists) and EX (expiry).
// Key format: auction:{auctionID}:lock
// Value: a UUID so we can identify the lock holder.
func (m *RedisLockManager) AcquireLock(ctx context.Context, auctionID int64, ttl time.Duration) error {
	key, value := fmt.Sprintf("auction:%d:lock", auctionID), uuid.New().String()
	ok, err := m.client.SetNX(ctx, key, value, ttl).Result()
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	if !ok {
		return domain.ErrLockAcquisitionFailed
	}

	m.logger.Debug("lock acquired", zap.Int64("auction_id", auctionID), zap.String("key", key))

	return nil
}

// ReleaseLock deletes the lock key for the given auction.
func (m *RedisLockManager) ReleaseLock(ctx context.Context, auctionID int64) error {
	key := fmt.Sprintf("auction:%d:lock", auctionID)
	err := m.client.Del(ctx, key).Err()
	if err != nil {
		m.logger.Warn("failed to release lock", zap.Int64("auction_id", auctionID), zap.Error(err))
		return fmt.Errorf("failed to release lock: %w", err)
	}

	m.logger.Debug("lock released", zap.Int64("auction_id", auctionID), zap.String("key", key))

	return nil
}

// Close shuts down the Redis client connection.
func (m *RedisLockManager) Close() error {
	return m.client.Close()
}
