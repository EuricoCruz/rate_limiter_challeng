package middleware

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/EuricoCruz/rate_limiter_challeng/internal/domain/entity"
	"github.com/EuricoCruz/rate_limiter_challeng/internal/usecase/check_rate_limit"
)

// MockUseCase simula o use case para testes
type MockUseCase struct {
	mock.Mock
}

func (m *MockUseCase) Execute(ctx context.Context, input check_rate_limit.Input) (*check_rate_limit.Output, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*check_rate_limit.Output), args.Error(1)
}

// MockConfig simula a configuração para testes
type MockConfig struct {
	IPLimit     int
	IPWindow    time.Duration
	IPBlockTime time.Duration
}

func (m *MockConfig) GetIPLimit() int {
	return m.IPLimit
}

func (m *MockConfig) GetIPWindow() time.Duration {
	return m.IPWindow
}

func (m *MockConfig) GetIPBlockTime() time.Duration {
	return m.IPBlockTime
}

func (m *MockConfig) GetTokenConfig(token string) (TokenConfig, bool) {
	// Retorna config fake para token "test-token"
	if token == "test-token" {
		return TokenConfig{
			Limit:     100,
			Window:    time.Second,
			BlockTime: 5 * time.Minute,
		}, true
	}
	return TokenConfig{}, false
}

func TestExtractIP_FromRemoteAddr(t *testing.T) {
	// Arrange
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"

	// Act
	ip := extractIP(req)

	// Assert
	assert.Equal(t, "192.168.1.1", ip)
}

func TestExtractIP_FromXForwardedFor(t *testing.T) {
	// Arrange
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Forwarded-For", "1.2.3.4, 5.6.7.8")

	// Act
	ip := extractIP(req)

	// Assert
	assert.Equal(t, "1.2.3.4", ip)
}

func TestExtractIP_FromXRealIP(t *testing.T) {
	// Arrange
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Real-IP", "9.8.7.6")

	// Act
	ip := extractIP(req)

	// Assert
	assert.Equal(t, "9.8.7.6", ip)
}

func TestRateLimiterMiddleware_AllowsRequest(t *testing.T) {
	// Arrange
	mockUseCase := new(MockUseCase)
	mockConfig := &MockConfig{
		IPLimit:     10,
		IPWindow:    time.Second,
		IPBlockTime: 5 * time.Minute,
	}

	// Mock successful rate limit check
	mockUseCase.On("Execute", mock.Anything, mock.AnythingOfType("check_rate_limit.Input")).Return(
		&check_rate_limit.Output{
			Allowed:       true,
			CurrentTokens: 9.0,
			Limit:         10,
			Blocked:       false,
		}, nil,
	).Once()

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()

	nextHandlerCalled := false
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextHandlerCalled = true
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Act
	middleware := createRateLimiterMiddleware(mockUseCase, mockConfig)
	middleware(nextHandler).ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, nextHandlerCalled)
	mockUseCase.AssertExpectations(t)
}

func TestRateLimiterMiddleware_BlocksRequest(t *testing.T) {
	// Arrange
	mockUseCase := new(MockUseCase)
	mockConfig := &MockConfig{
		IPLimit:     10,
		IPWindow:    time.Second,
		IPBlockTime: 5 * time.Minute,
	}

	// Mock blocked rate limit check
	rateLimitMessage := "you have reached the maximum number of requests or actions allowed within a certain time frame"
	mockUseCase.On("Execute", mock.Anything, mock.AnythingOfType("check_rate_limit.Input")).Return(
		&check_rate_limit.Output{
			Allowed:       false,
			CurrentTokens: 0.0,
			Limit:         10,
			Blocked:       true,
			Message:       rateLimitMessage,
		}, nil,
	).Once()

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()

	nextHandlerCalled := false
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextHandlerCalled = true
	})

	// Act
	middleware := createRateLimiterMiddleware(mockUseCase, mockConfig)
	middleware(nextHandler).ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.False(t, nextHandlerCalled)

	body, _ := io.ReadAll(w.Body)
	assert.Contains(t, string(body), rateLimitMessage)
	mockUseCase.AssertExpectations(t)
}

func TestRateLimiterMiddleware_UsesIPByDefault(t *testing.T) {
	// Arrange
	mockUseCase := new(MockUseCase)
	mockConfig := &MockConfig{
		IPLimit:     10,
		IPWindow:    time.Second,
		IPBlockTime: 5 * time.Minute,
	}

	// Mock that expects IP key to be used
	expectedIPKey := entity.NewIPKey("192.168.1.1")
	mockUseCase.On("Execute", mock.Anything, check_rate_limit.Input{
		Key:       expectedIPKey,
		Limit:     10,
		Window:    time.Second,
		BlockTime: 5 * time.Minute,
	}).Return(
		&check_rate_limit.Output{
			Allowed: true,
		}, nil,
	).Once()

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Act
	middleware := createRateLimiterMiddleware(mockUseCase, mockConfig)
	middleware(nextHandler).ServeHTTP(w, req)

	// Assert
	mockUseCase.AssertExpectations(t)
}

func TestRateLimiterMiddleware_UsesTokenWhenProvided(t *testing.T) {
	// Arrange
	mockUseCase := new(MockUseCase)
	mockConfig := &MockConfig{
		IPLimit:     10,
		IPWindow:    time.Second,
		IPBlockTime: 5 * time.Minute,
	}

	// Mock that expects token key to be used with token config
	expectedTokenKey := entity.NewTokenKey("test-token")
	mockUseCase.On("Execute", mock.Anything, check_rate_limit.Input{
		Key:       expectedTokenKey,
		Limit:     100, // Token limit, not IP limit
		Window:    time.Second,
		BlockTime: 5 * time.Minute,
	}).Return(
		&check_rate_limit.Output{
			Allowed: true,
		}, nil,
	).Once()

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	req.Header.Set("API_KEY", "test-token")
	w := httptest.NewRecorder()

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Act
	middleware := createRateLimiterMiddleware(mockUseCase, mockConfig)
	middleware(nextHandler).ServeHTTP(w, req)

	// Assert
	mockUseCase.AssertExpectations(t)
}

func TestRateLimiterMiddleware_TokenOverridesIP(t *testing.T) {
	// Arrange
	mockUseCase := new(MockUseCase)
	mockConfig := &MockConfig{
		IPLimit:     10, // IP limit is 10
		IPWindow:    time.Second,
		IPBlockTime: 5 * time.Minute,
	}

	// Mock that expects token config to override IP config
	// Token limit is 100, IP limit is 10 - should use 100
	mockUseCase.On("Execute", mock.Anything, mock.MatchedBy(func(input check_rate_limit.Input) bool {
		// Verify it's using token key and token limit (100), not IP limit (10)
		return input.Key.Type == entity.KeyTypeToken &&
			input.Key.Value == "test-token" &&
			input.Limit == 100 // Token limit, not IP limit of 10
	})).Return(
		&check_rate_limit.Output{
			Allowed: true,
		}, nil,
	).Once()

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	req.Header.Set("API_KEY", "test-token")
	w := httptest.NewRecorder()

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Act
	middleware := createRateLimiterMiddleware(mockUseCase, mockConfig)
	middleware(nextHandler).ServeHTTP(w, req)

	// Assert
	mockUseCase.AssertExpectations(t)
}

// createRateLimiterMiddleware é uma função helper para criar o middleware nos testes
func createRateLimiterMiddleware(useCase UseCase, config Config) func(http.Handler) http.Handler {
	return RateLimiterMiddlewareHandlerWrapper(useCase, config)
}
