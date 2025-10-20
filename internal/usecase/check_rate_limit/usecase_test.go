package check_rate_limit

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/EuricoCruz/rate_limiter_challeng/internal/domain/entity"
	"github.com/EuricoCruz/rate_limiter_challeng/internal/domain/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestExecute_InvalidInput_ReturnsError(t *testing.T) {
	// Arrange
	mockStorage := new(MockStorage)
	useCase := NewUseCase(mockStorage)

	input := Input{
		Key:       entity.LimiterKey{Type: entity.KeyTypeIP, Value: ""}, // Invalid key
		Limit:     10,
		Window:    time.Second,
		BlockTime: 5 * time.Minute,
	}

	// Act
	output, err := useCase.Execute(context.Background(), input)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, output)
	assert.Contains(t, err.Error(), "invalid limiter key")
}

func TestExecute_WhenBlocked_ReturnsBlockedOutput(t *testing.T) {
	// Arrange
	mockStorage := new(MockStorage)
	useCase := NewUseCase(mockStorage)

	input := Input{
		Key:       entity.NewIPKey("192.168.1.1"),
		Limit:     10,
		Window:    time.Second,
		BlockTime: 5 * time.Minute,
	}

	mockStorage.On("IsBlocked", mock.Anything, mock.Anything).Return(true, nil)

	// Act
	output, err := useCase.Execute(context.Background(), input)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, output)
	assert.True(t, output.Blocked)
	assert.False(t, output.Allowed)
	assert.NotEmpty(t, output.Message)

	mockStorage.AssertCalled(t, "IsBlocked", mock.Anything, mock.Anything)
}

func TestExecute_WhenAllowed_ReturnsAllowedOutput(t *testing.T) {
	// Arrange
	mockStorage := new(MockStorage)
	useCase := NewUseCase(mockStorage)

	input := Input{
		Key:       entity.NewIPKey("192.168.1.1"),
		Limit:     10,
		Window:    time.Second,
		BlockTime: 5 * time.Minute,
	}

	checkResult := &repository.CheckResult{
		Allowed:       true,
		CurrentTokens: 9.0,
		Limit:         10,
	}

	mockStorage.On("IsBlocked", mock.Anything, mock.Anything).Return(false, nil)
	mockStorage.On("CheckAndConsume", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(checkResult, nil)

	// Act
	output, err := useCase.Execute(context.Background(), input)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, output)
	assert.True(t, output.Allowed)
	assert.False(t, output.Blocked)
	assert.Equal(t, 9.0, output.CurrentTokens)
	assert.Equal(t, 10, output.Limit)

	mockStorage.AssertCalled(t, "IsBlocked", mock.Anything, mock.Anything)
	mockStorage.AssertCalled(t, "CheckAndConsume", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestExecute_WhenRateLimitExceeded_BlocksKey(t *testing.T) {
	// Arrange
	mockStorage := new(MockStorage)
	useCase := NewUseCase(mockStorage)

	input := Input{
		Key:       entity.NewIPKey("192.168.1.1"),
		Limit:     10,
		Window:    time.Second,
		BlockTime: 5 * time.Minute,
	}

	checkResult := &repository.CheckResult{
		Allowed:       false,
		CurrentTokens: 0.0,
		Limit:         10,
	}

	mockStorage.On("IsBlocked", mock.Anything, mock.Anything).Return(false, nil)
	mockStorage.On("CheckAndConsume", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(checkResult, nil)
	mockStorage.On("SetBlock", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// Act
	output, err := useCase.Execute(context.Background(), input)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, output)
	assert.False(t, output.Allowed)
	assert.False(t, output.Blocked)
	assert.NotEmpty(t, output.Message)

	mockStorage.AssertCalled(t, "IsBlocked", mock.Anything, mock.Anything)
	mockStorage.AssertCalled(t, "CheckAndConsume", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	mockStorage.AssertCalled(t, "SetBlock", mock.Anything, mock.Anything, mock.Anything)
}

func TestExecute_StorageIsBlockedError_PropagatesError(t *testing.T) {
	// Arrange
	mockStorage := new(MockStorage)
	useCase := NewUseCase(mockStorage)

	input := Input{
		Key:       entity.NewIPKey("192.168.1.1"),
		Limit:     10,
		Window:    time.Second,
		BlockTime: 5 * time.Minute,
	}

	expectedError := errors.New("storage error")
	mockStorage.On("IsBlocked", mock.Anything, mock.Anything).Return(false, expectedError)

	// Act
	output, err := useCase.Execute(context.Background(), input)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
	assert.Nil(t, output)

	mockStorage.AssertCalled(t, "IsBlocked", mock.Anything, mock.Anything)
}

func TestExecute_StorageCheckAndConsumeError_PropagatesError(t *testing.T) {
	// Arrange
	mockStorage := new(MockStorage)
	useCase := NewUseCase(mockStorage)

	input := Input{
		Key:       entity.NewIPKey("192.168.1.1"),
		Limit:     10,
		Window:    time.Second,
		BlockTime: 5 * time.Minute,
	}

	expectedError := errors.New("storage check error")

	mockStorage.On("IsBlocked", mock.Anything, mock.Anything).Return(false, nil)
	mockStorage.On("CheckAndConsume", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, expectedError)

	// Act
	output, err := useCase.Execute(context.Background(), input)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
	assert.Nil(t, output)

	mockStorage.AssertCalled(t, "IsBlocked", mock.Anything, mock.Anything)
	mockStorage.AssertCalled(t, "CheckAndConsume", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestExecute_StorageSetBlockError_PropagatesError(t *testing.T) {
	// Arrange
	mockStorage := new(MockStorage)
	useCase := NewUseCase(mockStorage)

	input := Input{
		Key:       entity.NewIPKey("192.168.1.1"),
		Limit:     10,
		Window:    time.Second,
		BlockTime: 5 * time.Minute,
	}

	checkResult := &repository.CheckResult{
		Allowed:       false,
		CurrentTokens: 0.0,
		Limit:         10,
	}

	expectedError := errors.New("set block error")

	mockStorage.On("IsBlocked", mock.Anything, mock.Anything).Return(false, nil)
	mockStorage.On("CheckAndConsume", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(checkResult, nil)
	mockStorage.On("SetBlock", mock.Anything, mock.Anything, mock.Anything).Return(expectedError)

	// Act
	output, err := useCase.Execute(context.Background(), input)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
	assert.Nil(t, output)

	mockStorage.AssertCalled(t, "IsBlocked", mock.Anything, mock.Anything)
	mockStorage.AssertCalled(t, "CheckAndConsume", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	mockStorage.AssertCalled(t, "SetBlock", mock.Anything, mock.Anything, mock.Anything)
}
