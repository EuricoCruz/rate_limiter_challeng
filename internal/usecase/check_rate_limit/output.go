package check_rate_limit

// Output represents the result of a rate limit check operation
type Output struct {
	// Allowed indicates whether the request should be permitted to proceed.
	// true = request is allowed, false = request should be blocked
	Allowed bool

	// CurrentTokens shows the number of tokens available in the bucket after the check.
	// This helps with debugging and monitoring rate limit status.
	CurrentTokens float64

	// Limit is the configured maximum number of requests allowed per time window.
	// Useful for displaying rate limit information to clients.
	Limit int

	// Blocked indicates if the key is currently in a blocked state due to previous rate limit violations.
	// true = key is temporarily blocked, false = key is not blocked
	// This is different from Allowed - a key can be blocked even if it has tokens available.
	Blocked bool

	// Message contains a human-readable explanation about the rate limit decision.
	// When rate limit is exceeded, this will contain the standardized message:
	// "you have reached the maximum number of requests or actions allowed within a certain time frame"
	Message string
}
