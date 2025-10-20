package entity

import (
	"errors"
	"math"
	"time"
)

var ErrRateLimitExceeded = errors.New("rate limit exceeded")

// RateLimit represents the rate limiting rules
type RateLimit struct {
	Key           LimiterKey
	Limit         int           // Requests allowed per window
	Window        time.Duration // Time window (e.g., 1 second)
	BlockTime     time.Duration // Block time after exceeding
	CurrentTokens float64       // Available tokens in bucket
	LastRefill    time.Time     // Last token refill
}

// NewRateLimit creates a new RateLimit instance with initial values
func NewRateLimit(key LimiterKey, limit int, window, blockTime time.Duration) *RateLimit {
	now := time.Now()
	return &RateLimit{
		Key:           key,
		Limit:         limit,
		Window:        window,
		BlockTime:     blockTime,
		CurrentTokens: float64(limit), // Start with full bucket
		LastRefill:    now,
	}
}

// CanConsume verifies if can consume one token (business rule)
func (r *RateLimit) CanConsume() bool {
	return r.CurrentTokens >= 1
}

// ConsumeToken consumes one token from the bucket
func (r *RateLimit) ConsumeToken() error {
	if !r.CanConsume() {
		return ErrRateLimitExceeded
	}
	r.CurrentTokens -= 1
	return nil
}

// RefillTokens calculates and adds tokens based on elapsed time using Token Bucket Algorithm
//
// This method implements the core logic of the Token Bucket algorithm:
// 1. Calculate how much time has passed since last refill
// 2. Determine how many tokens should be added based on elapsed time
// 3. Add tokens without exceeding the bucket capacity
// 4. Update the last refill timestamp
//
// Example:
//
//	RateLimit{Limit: 10, Window: 1*time.Second, CurrentTokens: 0, LastRefill: now-500ms}
//	RefillTokens(now) would add 5 tokens (0.5s * 10 tokens/s = 5 tokens)
func (r *RateLimit) RefillTokens(now time.Time) {
	// 1. Calculate elapsed time in seconds since last refill
	// This tells us how much time has passed and how many tokens we should generate
	elapsed := now.Sub(r.LastRefill).Seconds()

	// 2. Calculate refill rate (tokens per second)
	// The rate at which tokens should be added to the bucket
	// If Limit=10 and Window=1s, then refillRate=10 tokens/second
	// If Limit=100 and Window=60s, then refillRate=1.67 tokens/second
	refillRate := float64(r.Limit) / r.Window.Seconds()

	// 3. Calculate how many tokens to add based on elapsed time
	// tokensToAdd = time_elapsed × tokens_per_second
	// Example: 0.5 seconds elapsed × 10 tokens/sec = 5 tokens to add
	tokensToAdd := elapsed * refillRate

	// 4. Add tokens without exceeding the bucket capacity
	// Take the minimum between (current + new tokens) and the maximum capacity
	// This ensures we never go above the Limit even if a lot of time has passed
	r.CurrentTokens = math.Min(float64(r.Limit), r.CurrentTokens+tokensToAdd)

	// 5. Update the last refill timestamp to the current time
	// This ensures accurate calculation for the next refill operation
	r.LastRefill = now
}
