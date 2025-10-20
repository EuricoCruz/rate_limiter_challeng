package entity

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewIPKey_CreatesCorrectKeyType(t *testing.T) {
	key := NewIPKey("192.168.1.1")
	assert.Equal(t, KeyTypeIP, key.Type)
	assert.Equal(t, "192.168.1.1", key.Value)
}

func TestNewTokenKey_CreatesCorrectKeyType(t *testing.T) {
	key := NewTokenKey("abc123")
	assert.Equal(t, KeyTypeToken, key.Type)
	assert.Equal(t, "abc123", key.Value)
}

func TestLimiterKeyString_FormatsAsRedisKey(t *testing.T) {
	ipKey := NewIPKey("192.168.1.1")
	tokenKey := NewTokenKey("abc123")

	assert.Equal(t, "rate_limit:ip:192.168.1.1", ipKey.String())
	assert.Equal(t, "rate_limit:token:abc123", tokenKey.String())
}

func TestLimiterKeyIsValid_ReturnsTrueForValid(t *testing.T) {
	key := LimiterKey{Type: KeyTypeIP, Value: "127.0.0.1"}
	assert.True(t, key.IsValid())
}

func TestLimiterKeyIsValid_ReturnsFalseForInvalid(t *testing.T) {
	cases := []LimiterKey{
		{Type: "", Value: "127.0.0.1"}, // empty type
		{Type: KeyTypeIP, Value: ""},   // empty value
		{Type: "", Value: ""},          // both empty
	}

	for _, c := range cases {
		assert.False(t, c.IsValid())
	}
}
