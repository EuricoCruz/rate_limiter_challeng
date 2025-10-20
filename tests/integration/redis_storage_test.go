//go:build integration
// +build integration

package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/EuricoCruz/rate_limiter_challeng/internal/adapter/storage/redis"
	"github.com/EuricoCruz/rate_limiter_challeng/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedisStorage_IsBlocked_ReturnsFalseWhenNotBlocked(t *testing.T) {
	// Arrange
	client := setupRedis(t)
	redisStorage := redis.NewRedisStorage(client)
	defer redisStorage.Close()

	key := entity.NewIPKey("192.168.1.1")
	ctx := context.Background()

	// Act
	blocked, err := redisStorage.IsBlocked(ctx, key)

	// Assert
	require.NoError(t, err)
	assert.False(t, blocked)
}

func TestRedisStorage_SetBlock_CreatesBlockedKey(t *testing.T) {
	// Arrange
	client := setupRedis(t)
	redisStorage := redis.NewRedisStorage(client)
	defer redisStorage.Close()

	key := entity.NewIPKey("192.168.1.1")
	blockTime := 2 * time.Second
	ctx := context.Background()

	// Act
	err := redisStorage.SetBlock(ctx, key, blockTime)
	require.NoError(t, err)

	blocked, err := redisStorage.IsBlocked(ctx, key)

	// Assert
	require.NoError(t, err)
	assert.True(t, blocked)
}

func TestRedisStorage_SetBlock_ExpiresAfterBlockTime(t *testing.T) {
	// Arrange
	client := setupRedis(t)
	redisStorage := redis.NewRedisStorage(client)
	defer redisStorage.Close()

	key := entity.NewIPKey("192.168.1.1")
	blockTime := 1 * time.Second
	ctx := context.Background()

	// Act
	err := redisStorage.SetBlock(ctx, key, blockTime)
	require.NoError(t, err)

	// Verify it's blocked initially
	blocked, err := redisStorage.IsBlocked(ctx, key)
	require.NoError(t, err)
	assert.True(t, blocked, "Key should be blocked initially")

	// Wait for expiration
	time.Sleep(1500 * time.Millisecond)

	// Assert - should no longer be blocked
	blocked, err = redisStorage.IsBlocked(ctx, key)
	require.NoError(t, err)
	assert.False(t, blocked, "Key should not be blocked after expiration")
}

func TestRedisStorage_CheckAndConsume_AllowsFirstNRequests(t *testing.T) {
	// Arrange
	client := setupRedis(t)
	redisStorage := redis.NewRedisStorage(client)
	defer redisStorage.Close()

	key := entity.NewIPKey("192.168.1.1")
	limit := 5
	window := time.Second
	ctx := context.Background()

	// Act & Assert - First 5 requests should be allowed
	for i := 0; i < 5; i++ {
		result, err := redisStorage.CheckAndConsume(ctx, key, limit, window)
		require.NoError(t, err)
		assert.True(t, result.Allowed, "Request %d should be allowed", i+1)
		assert.Equal(t, limit, result.Limit)
	}

	// 6th request should be blocked
	result, err := redisStorage.CheckAndConsume(ctx, key, limit, window)
	require.NoError(t, err)
	assert.False(t, result.Allowed, "6th request should be blocked")
	assert.Equal(t, limit, result.Limit)
}

func TestRedisStorage_CheckAndConsume_RefillsTokensOverTime(t *testing.T) {
	// Arrange
	client := setupRedis(t)
	redisStorage := redis.NewRedisStorage(client)
	defer redisStorage.Close()

	key := entity.NewIPKey("192.168.1.1")
	limit := 10
	window := time.Second
	ctx := context.Background()

	// Act - Consume all tokens
	for i := 0; i < limit; i++ {
		result, err := redisStorage.CheckAndConsume(ctx, key, limit, window)
		require.NoError(t, err)
		assert.True(t, result.Allowed, "Request %d should be allowed", i+1)
	}

	// Verify bucket is exhausted
	result, err := redisStorage.CheckAndConsume(ctx, key, limit, window)
	require.NoError(t, err)
	assert.False(t, result.Allowed, "Request should be blocked after consuming all tokens")

	// Wait for refill (500ms should refill ~5 tokens)
	time.Sleep(500 * time.Millisecond)

	// Assert - Should be able to consume again after refill
	result, err = redisStorage.CheckAndConsume(ctx, key, limit, window)
	require.NoError(t, err)
	assert.True(t, result.Allowed, "Should be able to consume after token refill")
}

func TestRedisStorage_CheckAndConsume_DoesNotExceedCapacity(t *testing.T) {
	// Arrange
	client := setupRedis(t)
	redisStorage := redis.NewRedisStorage(client)
	defer redisStorage.Close()

	key := entity.NewIPKey("192.168.1.1")
	limit := 5
	window := time.Second
	ctx := context.Background()

	// Act - Wait 2 seconds (should refill 10 tokens, but should cap at limit)
	time.Sleep(2 * time.Second)

	result, err := redisStorage.CheckAndConsume(ctx, key, limit, window)
	require.NoError(t, err)

	// Assert - Should be capped at limit, not exceeded
	assert.True(t, result.Allowed)
	assert.Equal(t, float64(limit-1), result.CurrentTokens, "CurrentTokens should be 4.0 (limit-1), not 9.0")
	assert.Equal(t, limit, result.Limit)
}

func TestRedisStorage_CheckAndConsume_TracksCurrentTokens(t *testing.T) {
	// Arrange
	client := setupRedis(t)
	redisStorage := redis.NewRedisStorage(client)
	defer redisStorage.Close()

	key := entity.NewIPKey("192.168.1.1")
	limit := 10
	window := time.Second
	ctx := context.Background()

	// Act & Assert - Track token consumption
	expectedTokens := []float64{9.0, 8.0, 7.0, 6.0, 5.0}

	for i, expected := range expectedTokens {
		result, err := redisStorage.CheckAndConsume(ctx, key, limit, window)
		require.NoError(t, err)
		assert.True(t, result.Allowed, "Request %d should be allowed", i+1)
		assert.Equal(t, expected, result.CurrentTokens, "CurrentTokens should be %.1f after %d requests", expected, i+1)
		assert.Equal(t, limit, result.Limit)
	}
}
