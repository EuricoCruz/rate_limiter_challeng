package check_rate_limit

import (
	"context"

	"github.com/EuricoCruz/rate_limiter_challeng/internal/domain/repository"
)

// RateLimitExceededMessage is the standardized message returned when rate limit is exceeded
const RateLimitExceededMessage = "you have reached the maximum number of requests or actions allowed within a certain time frame"

// UseCase implements the business logic for rate limit checking
type UseCase struct {
	storage repository.Storage
}

// NewUseCase creates a new instance using dependency injection
func NewUseCase(storage repository.Storage) *UseCase {
	return &UseCase{storage: storage}
}

// Execute is the main command that checks if a request should be allowed based on rate limiting rules.
// It follows the Command Pattern and implements the business logic for rate limit verification.
//
// The execution flow:
// 1. Validate input parameters
// 2. Check if the key is currently in a blocked state
// 3. If blocked, return immediate rejection
// 4. Otherwise, attempt to consume a token using Token Bucket algorithm
// 5. If consumption fails, block the key and return rejection
// 6. If consumption succeeds, return success with current state
func (uc *UseCase) Execute(ctx context.Context, input Input) (*Output, error) {
	// 1. Validate input parameters (Single Responsibility Principle)
	if err := input.Validate(); err != nil {
		return nil, err
	}

	// 2. Check if key is currently blocked due to previous violations
	blocked, err := uc.storage.IsBlocked(ctx, input.Key)
	if err != nil {
		return nil, err
	}

	if blocked {
		return uc.createBlockedOutput(), nil
	}

	// 3. Attempt to consume token using Token Bucket algorithm (atomic operation)
	result, err := uc.storage.CheckAndConsume(ctx, input.Key, input.Limit, input.Window)
	if err != nil {
		return nil, err
	}

	// 4. If token consumption failed (rate limit exceeded), block the key
	if !result.Allowed {
		if err := uc.storage.SetBlock(ctx, input.Key, input.BlockTime); err != nil {
			return nil, err
		}

		return uc.createRateLimitExceededOutput(result), nil
	}

	// 5. Token consumption successful - request is allowed
	return uc.createAllowedOutput(result), nil
}

// createBlockedOutput creates an output response when the key is already blocked
func (uc *UseCase) createBlockedOutput() *Output {
	return &Output{
		Allowed: false,
		Blocked: true,
		Message: RateLimitExceededMessage,
	}
}

// createRateLimitExceededOutput creates an output response when rate limit is just exceeded
func (uc *UseCase) createRateLimitExceededOutput(result *repository.CheckResult) *Output {
	return &Output{
		Allowed:       false,
		Blocked:       false, // Key was just blocked, not previously blocked
		CurrentTokens: result.CurrentTokens,
		Limit:         result.Limit,
		Message:       RateLimitExceededMessage,
	}
}

// createAllowedOutput creates an output response when the request is allowed
func (uc *UseCase) createAllowedOutput(result *repository.CheckResult) *Output {
	return &Output{
		Allowed:       true,
		CurrentTokens: result.CurrentTokens,
		Limit:         result.Limit,
		Blocked:       false,
	}
}
