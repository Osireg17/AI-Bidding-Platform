package lock

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/Osireg17/AI-Bidding-Platform/services/bid-service/internal/domain"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var (
	testManager *RedisLockManager
	testClient  *redis.Client // separate client for test assertions (check key existence, cleanup)
)

func TestMain(m *testing.M) {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}

	logger := zap.NewNop()
	mgr, err := NewRedisLockManager(redisURL, logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "skipping lock tests: %v\n", err)
		os.Exit(0)
	}
	testManager = mgr

	opts, _ := redis.ParseURL(redisURL)
	testClient = redis.NewClient(opts)

	code := m.Run()

	testManager.Close()
	testClient.Close()

	os.Exit(code)
}

// cleanup deletes all test lock keys between tests.
func cleanup(t *testing.T, auctionID int64) {
	t.Helper()
	key := fmt.Sprintf("auction:%d:lock", auctionID)
	testClient.Del(context.Background(), key)
}

func TestAcquireLock_Success(t *testing.T) {
	ctx := context.Background()
	auctionID := int64(1001)
	defer cleanup(t, auctionID)

	err := testManager.AcquireLock(ctx, auctionID, 5*time.Second)
	require.NoError(t, err)

	// Verify the key exists in Redis.
	key := fmt.Sprintf("auction:%d:lock", auctionID)
	val, err := testClient.Get(ctx, key).Result()
	require.NoError(t, err)
	assert.NotEmpty(t, val, "lock value should be a UUID string")
}

func TestAcquireLock_AlreadyHeld(t *testing.T) {
	ctx := context.Background()
	auctionID := int64(1002)
	defer cleanup(t, auctionID)

	// First acquire succeeds.
	err := testManager.AcquireLock(ctx, auctionID, 5*time.Second)
	require.NoError(t, err)

	// Second acquire on same auction fails.
	err = testManager.AcquireLock(ctx, auctionID, 5*time.Second)
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrLockAcquisitionFailed))
}

func TestAcquireLock_DifferentAuctions(t *testing.T) {
	ctx := context.Background()
	auctionA := int64(1003)
	auctionB := int64(1004)
	defer cleanup(t, auctionA)
	defer cleanup(t, auctionB)

	// Locking different auctions should not conflict.
	err := testManager.AcquireLock(ctx, auctionA, 5*time.Second)
	require.NoError(t, err)

	err = testManager.AcquireLock(ctx, auctionB, 5*time.Second)
	require.NoError(t, err)
}

func TestAcquireLock_TTLExpiry(t *testing.T) {
	ctx := context.Background()
	auctionID := int64(1005)
	defer cleanup(t, auctionID)

	// Acquire with a very short TTL.
	err := testManager.AcquireLock(ctx, auctionID, 1*time.Second)
	require.NoError(t, err)

	// Wait for TTL to expire.
	time.Sleep(1100 * time.Millisecond)

	// Should be able to acquire again after expiry.
	err = testManager.AcquireLock(ctx, auctionID, 5*time.Second)
	require.NoError(t, err)
}

func TestReleaseLock_Success(t *testing.T) {
	ctx := context.Background()
	auctionID := int64(1006)
	defer cleanup(t, auctionID)

	err := testManager.AcquireLock(ctx, auctionID, 5*time.Second)
	require.NoError(t, err)

	err = testManager.ReleaseLock(ctx, auctionID)
	require.NoError(t, err)

	// Verify key is gone.
	key := fmt.Sprintf("auction:%d:lock", auctionID)
	exists, err := testClient.Exists(ctx, key).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(0), exists, "key should be deleted after release")
}

func TestReleaseLock_ReacquireAfterRelease(t *testing.T) {
	ctx := context.Background()
	auctionID := int64(1007)
	defer cleanup(t, auctionID)

	err := testManager.AcquireLock(ctx, auctionID, 5*time.Second)
	require.NoError(t, err)

	err = testManager.ReleaseLock(ctx, auctionID)
	require.NoError(t, err)

	// Should be able to acquire again after explicit release.
	err = testManager.AcquireLock(ctx, auctionID, 5*time.Second)
	require.NoError(t, err)
}

func TestReleaseLock_NonExistentKey(t *testing.T) {
	ctx := context.Background()
	auctionID := int64(9999)

	// Releasing a key that doesn't exist should not error.
	err := testManager.ReleaseLock(ctx, auctionID)
	assert.NoError(t, err)
}

func TestNewRedisLockManager_InvalidURL(t *testing.T) {
	logger := zap.NewNop()
	_, err := NewRedisLockManager("not-a-valid-url", logger)
	require.Error(t, err)
}
