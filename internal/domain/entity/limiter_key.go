package entity

import "fmt"

// KeyType represents the type of limiter key
type KeyType string

const (
	// KeyTypeIP represents an IP-based rate limit key
	KeyTypeIP KeyType = "ip"
	// KeyTypeToken represents a token-based rate limit key
	KeyTypeToken KeyType = "token"
)

// LimiterKey is a value object that represents a rate limiter key
type LimiterKey struct {
	Type  KeyType // The type of key (IP or Token)
	Value string  // The actual key value
}

// NewIPKey creates a new IP-based limiter key
func NewIPKey(ip string) LimiterKey {
	return LimiterKey{Type: KeyTypeIP, Value: ip}
}

// NewTokenKey creates a new Token-based limiter key
func NewTokenKey(token string) LimiterKey {
	return LimiterKey{Type: KeyTypeToken, Value: token}
}

// String returns the string representation for use as Redis key
func (k LimiterKey) String() string {
	return fmt.Sprintf("rate_limit:%s:%s", k.Type, k.Value)
}

// IsValid validates the value object
func (k LimiterKey) IsValid() bool {
	return k.Type != "" && k.Value != ""
}
