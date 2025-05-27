package ratelimiters

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// TokenBucket implements a token bucket rate limiter
// This allows for burst traffic up to the bucket capacity and smooth rate limiting
//
// Algorithm: O(1) time, O(1) memory per key
// - Mathematical token calculation
// - Excellent burst handling
// - Smooth rate limiting
// - Industry standard approach
//
// Visual representation:
//
//   ╔═══════════════════════════════════════════════════════════════════╗
//   ║  Target: 2 requests/second, Burst: 5 tokens                       ║
//   ║                                                                   ║
//   ║  Example: Single client making requests at 3 req/s                ║
//   ║                                                                   ║
//   ║                 Starting State                   Ending State     ║
//   ║                  ───────────┐                     ───────────┐    ║
//   ║ [0s]         ▶  │ ➊ ➋ ➌ ➍ ➎ │  ▶  3 req       ▶  │ ① ② ③ ➍ ➎ │    ║
//   ║                 └───────────┘     0 blocked      └───────────┘    ║
//   ║                                                                   ║
//   ║                  ───────────┐                     ───────────┐    ║
//   ║ [1s]  add 2  ▶  │ ➊ ➋ ③ ➍ ➎ │  ▶  3 req       ▶  │ ① ② ③ ④ ➎ │    ║
//   ║                 └───────────┘     0 blocked      └───────────┘    ║
//   ║                                                                   ║
//   ║                  ───────────┐                     ───────────┐    ║
//   ║ [2s]  add 2  ▶  │ ➊ ➋ ③ ④ ➎ │  ▶  3 req       ▶  │ ① ② ③ ④ ⑤ │    ║
//   ║                 └───────────┘     0 blocked      └───────────┘    ║
//   ║                                                                   ║
//   ║                  ───────────┐                     ───────────┐    ║
//   ║ [3s]  add 2  ▶  │ ➊ ➋ ③ ④ ⑤ │  ▶  3 req       ▶  │ ① ② ③ ④ ⑤ │    ║
//   ║                 └───────────┘     1 blocked      └───────────┘    ║
//   ║                                                                   ║
//   ╚═══════════════════════════════════════════════════════════════════╝

type TokenBucket struct {
	mu       sync.RWMutex
	buckets  map[string]*bucketState
	stopChan chan struct{}
	closed   bool

	nanosecondRefillInterval uint64
	bucketSize               uint64
}

type bucketState struct {
	availableTokens uint64
	lastFill        time.Time
}

// NewTokenBucket creates a new token bucket rate limiter
func NewTokenBucket(config Config) *TokenBucket {
	tb := &TokenBucket{
		buckets:                  make(map[string]*bucketState),
		bucketSize:               uint64(max(config.Burst, config.Rate)),
		stopChan:                 make(chan struct{}),
		nanosecondRefillInterval: uint64(float64(config.Duration.Nanoseconds()) / float64(config.Rate)),
	}

	// Start cleanup goroutine
	go tb.cleanup()

	return tb
}

func (tb *TokenBucket) Allow(ctx context.Context, key string) (bool, error) {
	return tb.AllowN(ctx, key, 1)
}

func (tb *TokenBucket) AllowN(ctx context.Context, key string, n int) (bool, error) {
	if n <= 0 {
		return true, nil
	}

	tb.mu.Lock()
	defer tb.mu.Unlock()

	if tb.closed {
		return false, errorClosed()
	}

	now := time.Now()
	availableTokens := tb.refreshAvailableTokens(key, now)

	if availableTokens >= uint64(n) {
		tb.consumeAvailableTokens(key, n)
		return true, nil
	}

	return false, nil
}

func (tb *TokenBucket) Wait(ctx context.Context, key string) error {
	return tb.WaitN(ctx, key, 1)
}

func (tb *TokenBucket) WaitN(ctx context.Context, key string, n int) error {
	if n <= 0 {
		return nil
	}

	if n > int(tb.bucketSize) {
		return fmt.Errorf("too many tokens requested: %d > %d", n, tb.bucketSize)
	}

	for {
		// Try to get tokens
		tb.mu.Lock()
		if tb.closed {
			tb.mu.Unlock()
			return errorClosed()
		}

		now := time.Now()
		availableTokens := tb.refreshAvailableTokens(key, now)

		if availableTokens >= uint64(n) {
			tb.consumeAvailableTokens(key, n)
			tb.mu.Unlock()
			return nil
		}

		// Calculate how long to wait for enough tokens
		tokensNeeded := uint64(n) - availableTokens
		waitDuration := time.Duration(tokensNeeded * tb.nanosecondRefillInterval)

		tb.mu.Unlock()

		if waitDuration <= 0 {
			continue // Try again immediately
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitDuration):
			// Continue to try again
		}
	}
}

func (tb *TokenBucket) Reset(ctx context.Context, key string) error {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	if tb.closed {
		return errorClosed()
	}

	delete(tb.buckets, key)
	return nil
}

func (tb *TokenBucket) Close() error {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	if !tb.closed {
		tb.closed = true
		close(tb.stopChan)
		tb.buckets = nil
	}
	return nil
}

func (tb *TokenBucket) String() string {
	return "TokenBucket"
}

func (tb *TokenBucket) consumeAvailableTokens(key string, n int) {
	tb.buckets[key].availableTokens -= uint64(n)
}

func (tb *TokenBucket) refreshAvailableTokens(key string, now time.Time) uint64 {
	bucket, exists := tb.buckets[key]
	if !exists {
		bucket = &bucketState{availableTokens: tb.bucketSize, lastFill: now}
		tb.buckets[key] = bucket
	}

	// Refill tokens based on time elapsed
	elapsed := now.Sub(bucket.lastFill)
	if elapsed > 0 {
		tokensToAdd := uint64(elapsed.Nanoseconds()) / tb.nanosecondRefillInterval
		bucket.availableTokens = min(bucket.availableTokens+tokensToAdd, tb.bucketSize)
		bucket.lastFill = now
	}

	return bucket.availableTokens
}

// cleanup removes unused buckets periodically
func (tb *TokenBucket) cleanup() {
	ticker := time.NewTicker(time.Duration(tb.nanosecondRefillInterval) * 2) // Clean up every 2 windows
	defer ticker.Stop()

	for {
		select {
		case <-tb.stopChan:
			return
		case now := <-ticker.C:
			tb.mu.Lock()
			cutoff := now.Add(-time.Duration(tb.nanosecondRefillInterval) * 5) // Remove buckets unused for 5 windows

			for key, bucket := range tb.buckets {
				if bucket.lastFill.Before(cutoff) {
					delete(tb.buckets, key)
				}
			}
			tb.mu.Unlock()
		}
	}
}
