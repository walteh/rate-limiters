package ratelimiters

import (
	"context"
	"math"
	"sync"
	"time"
)

// TokenBucket implements a token bucket rate limiter
// This allows for burst traffic up to the bucket capacity and smooth rate limiting
type TokenBucket struct {
	config   Config
	mu       sync.RWMutex
	buckets  map[string]*bucketState
	stopChan chan struct{}
	closed   bool
}

type bucketState struct {
	tokens   float64
	lastFill time.Time
}

// NewTokenBucket creates a new token bucket rate limiter
func NewTokenBucket(config Config) *TokenBucket {
	tb := &TokenBucket{
		config:   config,
		buckets:  make(map[string]*bucketState),
		stopChan: make(chan struct{}),
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
		return false, ErrNotSupported{Operation: "AllowN", Limiter: "TokenBucket (closed)"}
	}

	now := time.Now()
	bucket := tb.getBucket(key, now)

	if bucket.tokens >= float64(n) {
		bucket.tokens -= float64(n)
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

	for {
		// Try to get tokens
		tb.mu.Lock()
		if tb.closed {
			tb.mu.Unlock()
			return ErrNotSupported{Operation: "WaitN", Limiter: "TokenBucket (closed)"}
		}

		now := time.Now()
		bucket := tb.getBucket(key, now)

		if bucket.tokens >= float64(n) {
			bucket.tokens -= float64(n)
			tb.mu.Unlock()
			return nil
		}

		// Calculate how long to wait for enough tokens
		tokensNeeded := float64(n) - bucket.tokens
		refillRate := float64(tb.config.Rate) / tb.config.Duration.Seconds()
		waitDuration := time.Duration(tokensNeeded/refillRate*1000) * time.Millisecond

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
		return ErrNotSupported{Operation: "Reset", Limiter: "TokenBucket (closed)"}
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

// getBucket gets or creates a bucket for the given key and updates its tokens
// Must be called with lock held
func (tb *TokenBucket) getBucket(key string, now time.Time) *bucketState {
	bucket, exists := tb.buckets[key]
	if !exists {
		bucket = &bucketState{
			tokens:   float64(tb.getBurstSize()),
			lastFill: now,
		}
		tb.buckets[key] = bucket
		return bucket
	}

	// Refill tokens based on time elapsed
	elapsed := now.Sub(bucket.lastFill)
	if elapsed > 0 {
		refillRate := float64(tb.config.Rate) / tb.config.Duration.Seconds()
		tokensToAdd := refillRate * elapsed.Seconds()
		bucket.tokens = math.Min(bucket.tokens+tokensToAdd, float64(tb.getBurstSize()))
		bucket.lastFill = now
	}

	return bucket
}

// getBurstSize returns the burst size, using config.Burst if set, otherwise config.Rate
func (tb *TokenBucket) getBurstSize() int {
	if tb.config.Burst > 0 {
		return tb.config.Burst
	}
	return tb.config.Rate
}

// cleanup removes unused buckets periodically
func (tb *TokenBucket) cleanup() {
	ticker := time.NewTicker(tb.config.Duration * 2) // Clean up every 2 windows
	defer ticker.Stop()

	for {
		select {
		case <-tb.stopChan:
			return
		case now := <-ticker.C:
			tb.mu.Lock()
			cutoff := now.Add(-tb.config.Duration * 5) // Remove buckets unused for 5 windows

			for key, bucket := range tb.buckets {
				if bucket.lastFill.Before(cutoff) {
					delete(tb.buckets, key)
				}
			}
			tb.mu.Unlock()
		}
	}
}
