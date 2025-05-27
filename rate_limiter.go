package ratelimiters

import (
	"context"
	"time"
)

// RateLimiter defines the interface that all rate limiters must implement
type RateLimiter interface {
	// Allow checks if a request should be allowed based on the rate limiting policy
	// Returns true if the request is allowed, false if it should be rate limited
	Allow(ctx context.Context, key string) (bool, error)

	// AllowN checks if N requests should be allowed
	// Returns true if all N requests are allowed, false otherwise
	AllowN(ctx context.Context, key string, n int) (bool, error)

	// Wait blocks until the next request is allowed for the given key
	// Returns an error if the context is cancelled or if waiting is not supported
	Wait(ctx context.Context, key string) error

	// WaitN blocks until N requests are allowed for the given key
	WaitN(ctx context.Context, key string, n int) error

	// Reset clears the rate limiting state for the given key
	Reset(ctx context.Context, key string) error

	// Close cleans up any resources used by the rate limiter
	Close() error

	// String returns a human-readable description of the rate limiter
	String() string
}

// Config holds common configuration for rate limiters
type Config struct {
	// Rate is the number of requests allowed per Duration
	Rate int
	// Duration is the time window for the rate limit
	Duration time.Duration
	// Burst is the maximum number of requests that can be made in a burst
	// (only applicable to some rate limiter types)
	Burst int
}

// ErrNotSupported is returned when a rate limiter doesn't support a particular operation
type ErrNotSupported struct {
	Operation string
	Limiter   string
}

func (e ErrNotSupported) Error() string {
	return "operation " + e.Operation + " not supported by " + e.Limiter + " rate limiter"
}

// ErrRateLimited is returned when a request is rate limited
type ErrRateLimited struct {
	Key        string
	RetryAfter time.Duration
}

func (e ErrRateLimited) Error() string {
	return "rate limited for key: " + e.Key
}
