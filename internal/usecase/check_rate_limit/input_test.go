package check_rate_limit

import (
	"testing"
	"time"

	"github.com/EuricoCruz/rate_limiter_challeng/internal/domain/entity"
	"github.com/stretchr/testify/assert"
)

func TestInputValidate_WithValidData(t *testing.T) {
	input := Input{
		Key:       entity.NewIPKey("192.168.1.1"),
		Limit:     10,
		Window:    time.Second,
		BlockTime: 5 * time.Minute,
	}

	err := input.Validate()
	assert.NoError(t, err)
}

func TestInputValidate_WithInvalidKey(t *testing.T) {
	input := Input{
		Key:       entity.LimiterKey{Type: entity.KeyTypeIP, Value: ""}, // Empty value
		Limit:     10,
		Window:    time.Second,
		BlockTime: 5 * time.Minute,
	}

	err := input.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid limiter key")
}

func TestInputValidate_WithNegativeLimit(t *testing.T) {
	input := Input{
		Key:       entity.NewIPKey("192.168.1.1"),
		Limit:     -1,
		Window:    time.Second,
		BlockTime: 5 * time.Minute,
	}

	err := input.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "limit must be positive")
}

func TestInputValidate_WithZeroWindow(t *testing.T) {
	input := Input{
		Key:       entity.NewIPKey("192.168.1.1"),
		Limit:     10,
		Window:    0, // Zero window
		BlockTime: 5 * time.Minute,
	}

	err := input.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "window must be positive")
}

func TestInputValidate_WithNegativeBlockTime(t *testing.T) {
	input := Input{
		Key:       entity.NewIPKey("192.168.1.1"),
		Limit:     10,
		Window:    time.Second,
		BlockTime: -1 * time.Second, // Negative block time
	}

	err := input.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "block time cannot be negative")
}
