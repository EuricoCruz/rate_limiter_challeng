package check_rate_limit

import (
	"errors"
	"time"

	"github.com/EuricoCruz/rate_limiter_challeng/internal/domain/entity"
)

// Input represents the input data for rate limit checking (DTO - Data Transfer Object)
type Input struct {
	Key       entity.LimiterKey
	Limit     int
	Window    time.Duration
	BlockTime time.Duration
}

// Validate validates the input data following Single Responsibility Principle
func (i Input) Validate() error {
	if !i.Key.IsValid() {
		return errors.New("invalid limiter key")
	}
	if i.Limit <= 0 {
		return errors.New("limit must be positive")
	}
	if i.Window <= 0 {
		return errors.New("window must be positive")
	}
	if i.BlockTime < 0 {
		return errors.New("block time cannot be negative")
	}
	return nil
}
