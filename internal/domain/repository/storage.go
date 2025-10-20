package repository

import (
	"context"
	"time"

	"github.com/EuricoCruz/rate_limiter_challeng/internal/domain/entity"
)

// Storage defines the contract for persistence following Dependency Inversion Principle
// This interface allows the application business rules (use cases) to depend on abstractions
// rather than concrete implementations, enabling easy swapping of storage mechanisms.
type Storage interface {
	// CheckAndConsume verifies if a request can consume a token and consumes it atomically.
	// This method implements the core Token Bucket algorithm check in a thread-safe manner.
	// Returns CheckResult with information about whether the request was allowed and current state.
	CheckAndConsume(
		ctx context.Context,
		key entity.LimiterKey,
		limit int,
		window time.Duration,
	) (*CheckResult, error)

	// SetBlock blocks a key for a specified duration when rate limit is exceeded.
	// This prevents additional requests from the same key during the block period.
	SetBlock(
		ctx context.Context,
		key entity.LimiterKey,
		blockTime time.Duration,
	) error

	// IsBlocked checks if a key is currently blocked due to rate limit violation.
	// Returns true if the key is blocked, false otherwise.
	IsBlocked(ctx context.Context, key entity.LimiterKey) (bool, error)

	// Close closes any connections or resources used by the storage implementation.
	// Should be called during application shutdown for proper cleanup.
	Close() error
}

// CheckResult contains the result of a rate limit check operation
type CheckResult struct {
	Allowed       bool    // Whether the request is allowed to proceed
	CurrentTokens float64 // Current number of tokens available in the bucket
	Limit         int     // The configured limit for this key
}
