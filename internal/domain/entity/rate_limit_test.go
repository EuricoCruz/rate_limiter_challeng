package entity

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCanConsume_WithSufficientTokens(t *testing.T) {
	rateLimit := &RateLimit{
		CurrentTokens: 5.0,
	}
	assert.True(t, rateLimit.CanConsume())
}

func TestCanConsume_WithInsufficientTokens(t *testing.T) {
	rateLimit := &RateLimit{
		CurrentTokens: 0.5,
	}
	assert.False(t, rateLimit.CanConsume())
}

func TestConsumeToken_SuccessDecrementsTokens(t *testing.T) {
	rateLimit := &RateLimit{
		CurrentTokens: 3.0,
	}

	err := rateLimit.ConsumeToken()

	assert.NoError(t, err)
	assert.Equal(t, 2.0, rateLimit.CurrentTokens)
}

func TestConsumeToken_FailsWhenNoTokens(t *testing.T) {
	rateLimit := &RateLimit{
		CurrentTokens: 0.0,
	}

	err := rateLimit.ConsumeToken()

	assert.Error(t, err)
	assert.Equal(t, ErrRateLimitExceeded, err)
	assert.Equal(t, 0.0, rateLimit.CurrentTokens)
}

func TestRefillTokens_AddsTokensBasedOnElapsedTime(t *testing.T) {
	now := time.Now()
	rateLimit := &RateLimit{
		Limit:         10,
		Window:        time.Second,
		CurrentTokens: 0.0,
		LastRefill:    now.Add(-500 * time.Millisecond),
	}

	rateLimit.RefillTokens(now)

	assert.Equal(t, 5.0, rateLimit.CurrentTokens)
}

func TestRefillTokens_DoesNotExceedCapacity(t *testing.T) {
	now := time.Now()
	rateLimit := &RateLimit{
		Limit:         10,
		Window:        time.Second,
		CurrentTokens: 8.0,
		LastRefill:    now.Add(-1 * time.Second),
	}

	rateLimit.RefillTokens(now)

	assert.Equal(t, 10.0, rateLimit.CurrentTokens)
}

func TestRefillTokens_UpdatesLastRefill(t *testing.T) {
	now := time.Now()
	oldRefill := now.Add(-1 * time.Hour)
	rateLimit := &RateLimit{
		Limit:         10,
		Window:        time.Second,
		CurrentTokens: 0.0,
		LastRefill:    oldRefill,
	}

	rateLimit.RefillTokens(now)

	assert.Equal(t, now, rateLimit.LastRefill)
}

func TestNewRateLimit_CreatesInstanceWithInitialValues(t *testing.T) {
	key := NewIPKey("192.168.1.1")
	limit := 10
	window := time.Second
	blockTime := 5 * time.Minute

	rateLimit := NewRateLimit(key, limit, window, blockTime)

	assert.NotNil(t, rateLimit)
	assert.Equal(t, key, rateLimit.Key)
	assert.Equal(t, limit, rateLimit.Limit)
	assert.Equal(t, window, rateLimit.Window)
	assert.Equal(t, blockTime, rateLimit.BlockTime)
	assert.Equal(t, float64(limit), rateLimit.CurrentTokens)                // Should start with full bucket
	assert.WithinDuration(t, time.Now(), rateLimit.LastRefill, time.Second) // Should be set to current time
}
