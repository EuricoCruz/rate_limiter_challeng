package check_rate_limit

import (
	"context"
	"time"

	"github.com/EuricoCruz/rate_limiter_challeng/internal/domain/entity"
	"github.com/EuricoCruz/rate_limiter_challeng/internal/domain/repository"
	"github.com/stretchr/testify/mock"
)

// MockStorage is a mock implementation of the Storage interface for testing purposes
type MockStorage struct {
	mock.Mock
}

// CheckAndConsume mocks the CheckAndConsume method from Storage interface
func (m *MockStorage) CheckAndConsume(ctx context.Context, key entity.LimiterKey, limit int, window time.Duration) (*repository.CheckResult, error) {
	args := m.Called(ctx, key, limit, window)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.CheckResult), args.Error(1)
}

// SetBlock mocks the SetBlock method from Storage interface
func (m *MockStorage) SetBlock(ctx context.Context, key entity.LimiterKey, blockTime time.Duration) error {
	args := m.Called(ctx, key, blockTime)
	return args.Error(0)
}

// IsBlocked mocks the IsBlocked method from Storage interface
func (m *MockStorage) IsBlocked(ctx context.Context, key entity.LimiterKey) (bool, error) {
	args := m.Called(ctx, key)
	return args.Bool(0), args.Error(1)
}

// Close mocks the Close method from Storage interface
func (m *MockStorage) Close() error {
	args := m.Called()
	return args.Error(0)
}
