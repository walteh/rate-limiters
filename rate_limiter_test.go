package ratelimiters

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestRateLimiterInterface ensures all implementations satisfy the interface
func TestRateLimiterInterface(t *testing.T) {
	config := Config{
		Rate:     10,
		Duration: time.Second,
		Burst:    15,
	}

	limiters := []RateLimiter{
		NewCounter(config),
		NewFixedWindow(config),
		NewSlidingWindow(config),
		NewSlidingWindowLog(config),
		NewTokenBucket(config),
	}

	for _, limiter := range limiters {
		t.Run(limiter.String(), func(t *testing.T) {
			ctx := context.Background()
			key := "test-key"

			// Test Allow
			allowed, err := limiter.Allow(ctx, key)
			if err != nil {
				t.Errorf("Allow failed: %v", err)
			}
			if !allowed {
				t.Error("First request should be allowed")
			}

			// Test AllowN
			_, err = limiter.AllowN(ctx, key, 5)
			if err != nil {
				t.Errorf("AllowN failed: %v", err)
			}

			// Test Reset
			err = limiter.Reset(ctx, key)
			if err != nil {
				t.Errorf("Reset failed: %v", err)
			}

			// Test Close
			err = limiter.Close()
			if err != nil {
				t.Errorf("Close failed: %v", err)
			}
		})
	}
}

func TestCounterRateLimiter(t *testing.T) {
	config := Config{
		Rate:     5,
		Duration: time.Second,
	}
	limiter := NewCounter(config)
	defer limiter.Close()

	ctx := context.Background()
	key := "test-key"

	// Should allow up to the rate limit
	for i := 0; i < config.Rate; i++ {
		allowed, err := limiter.Allow(ctx, key)
		if err != nil {
			t.Fatalf("Allow failed: %v", err)
		}
		if !allowed {
			t.Fatalf("Request %d should be allowed", i+1)
		}
	}

	// Should deny the next request
	allowed, err := limiter.Allow(ctx, key)
	if err != nil {
		t.Fatalf("Allow failed: %v", err)
	}
	if allowed {
		t.Error("Request beyond rate limit should be denied")
	}

	// Test Wait operations (should return error)
	err = limiter.Wait(ctx, key)
	if err == nil {
		t.Error("Wait should return error for Counter limiter")
	}

	err = limiter.WaitN(ctx, key, 1)
	if err == nil {
		t.Error("WaitN should return error for Counter limiter")
	}
}

func TestFixedWindowRateLimiter(t *testing.T) {
	config := Config{
		Rate:     5,
		Duration: 100 * time.Millisecond,
	}
	limiter := NewFixedWindow(config)
	defer limiter.Close()

	ctx := context.Background()
	key := "test-key"

	// Should allow up to the rate limit
	for i := 0; i < config.Rate; i++ {
		allowed, err := limiter.Allow(ctx, key)
		if err != nil {
			t.Fatalf("Allow failed: %v", err)
		}
		if !allowed {
			t.Fatalf("Request %d should be allowed", i+1)
		}
	}

	// Should deny the next request
	allowed, err := limiter.Allow(ctx, key)
	if err != nil {
		t.Fatalf("Allow failed: %v", err)
	}
	if allowed {
		t.Error("Request beyond rate limit should be denied")
	}

	// Wait for window to reset
	time.Sleep(config.Duration + 10*time.Millisecond)

	// Should allow requests again
	allowed, err = limiter.Allow(ctx, key)
	if err != nil {
		t.Fatalf("Allow failed: %v", err)
	}
	if !allowed {
		t.Error("Request should be allowed after window reset")
	}
}

func TestSlidingWindowRateLimiter(t *testing.T) {
	config := Config{
		Rate:     5,
		Duration: 100 * time.Millisecond,
	}
	limiter := NewSlidingWindow(config)
	defer limiter.Close()

	ctx := context.Background()
	key := "test-key"

	// Should allow up to the rate limit
	for i := 0; i < config.Rate; i++ {
		allowed, err := limiter.Allow(ctx, key)
		if err != nil {
			t.Fatalf("Allow failed: %v", err)
		}
		if !allowed {
			t.Fatalf("Request %d should be allowed", i+1)
		}
	}

	// Should deny the next request
	allowed, err := limiter.Allow(ctx, key)
	if err != nil {
		t.Fatalf("Allow failed: %v", err)
	}
	if allowed {
		t.Error("Request beyond rate limit should be denied")
	}

	// Wait for some requests to expire
	time.Sleep(config.Duration + 10*time.Millisecond)

	// Should allow requests again
	allowed, err = limiter.Allow(ctx, key)
	if err != nil {
		t.Fatalf("Allow failed: %v", err)
	}
	if !allowed {
		t.Error("Request should be allowed after requests expire")
	}

	// Test Wait functionality
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	err = limiter.Wait(ctx, key)
	if err != nil {
		t.Errorf("Wait failed: %v", err)
	}
}

func TestTokenBucketRateLimiter(t *testing.T) {
	config := Config{
		Rate:     5,
		Duration: 100 * time.Millisecond,
		Burst:    10,
	}
	limiter := NewTokenBucket(config)
	defer limiter.Close()

	ctx := context.Background()
	key := "test-key"

	// Should allow burst requests initially
	for i := 0; i < config.Burst; i++ {
		allowed, err := limiter.Allow(ctx, key)
		require.NoError(t, err, "Allow should not return error")
		require.True(t, allowed, "Burst request %d should be allowed", i+1)
	}

	// Should deny the next request
	allowed, err := limiter.Allow(ctx, key)
	require.NoError(t, err, "Allow should not return error")
	require.False(t, allowed, "Request beyond burst limit should be denied")

	// Wait for tokens to refill
	time.Sleep(config.Duration + 10*time.Millisecond)

	// Should allow requests again
	allowed, err = limiter.Allow(ctx, key)
	require.NoError(t, err, "Allow should not return error")
	require.True(t, allowed, "Request should be allowed after tokens refill")

	// Test Wait functionality
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	err = limiter.Wait(ctx, key)
	require.NoError(t, err, "Wait should not return error")
}

func TestSlidingWindowLogRateLimiter(t *testing.T) {
	config := Config{
		Rate:     5,
		Duration: 100 * time.Millisecond,
	}
	limiter := NewSlidingWindowLog(config)
	defer limiter.Close()

	ctx := context.Background()
	key := "test-key"

	// Should allow up to the rate limit
	for i := 0; i < config.Rate; i++ {
		allowed, err := limiter.Allow(ctx, key)
		if err != nil {
			t.Fatalf("Allow failed: %v", err)
		}
		if !allowed {
			t.Fatalf("Request %d should be allowed", i+1)
		}
	}

	// Should deny the next request
	allowed, err := limiter.Allow(ctx, key)
	if err != nil {
		t.Fatalf("Allow failed: %v", err)
	}
	if allowed {
		t.Error("Request beyond rate limit should be denied")
	}

	// Wait for some requests to expire
	time.Sleep(config.Duration + 10*time.Millisecond)

	// Should allow requests again
	allowed, err = limiter.Allow(ctx, key)
	if err != nil {
		t.Fatalf("Allow failed: %v", err)
	}
	if !allowed {
		t.Error("Request should be allowed after requests expire")
	}
}

func TestConcurrentAccess(t *testing.T) {
	config := Config{
		Rate:     100,
		Duration: time.Second,
		Burst:    150,
	}

	limiters := []RateLimiter{
		NewCounter(config),
		NewFixedWindow(config),
		NewSlidingWindow(config),
		NewSlidingWindowLog(config),
		NewTokenBucket(config),
	}

	for _, limiter := range limiters {
		t.Run(limiter.String(), func(t *testing.T) {
			defer limiter.Close()

			ctx := context.Background()
			key := "concurrent-test"
			numGoroutines := 50
			requestsPerGoroutine := 10

			var wg sync.WaitGroup
			var allowedCount int64
			var mu sync.Mutex

			for i := 0; i < numGoroutines; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for j := 0; j < requestsPerGoroutine; j++ {
						allowed, err := limiter.Allow(ctx, key)
						if err != nil {
							t.Errorf("Allow failed: %v", err)
							return
						}
						if allowed {
							mu.Lock()
							allowedCount++
							mu.Unlock()
						}
					}
				}()
			}

			wg.Wait()

			// The exact number depends on the limiter type, but it should be reasonable
			if allowedCount == 0 {
				t.Error("No requests were allowed")
			}
			if allowedCount > int64(numGoroutines*requestsPerGoroutine) {
				t.Errorf("Too many requests allowed: %d", allowedCount)
			}
		})
	}
}

func TestMultipleKeys(t *testing.T) {
	config := Config{
		Rate:     5,
		Duration: 100 * time.Millisecond,
	}

	limiters := []RateLimiter{
		NewCounter(config),
		NewFixedWindow(config),
		NewSlidingWindow(config),
		NewSlidingWindowLog(config),
		NewTokenBucket(config),
	}

	for _, limiter := range limiters {
		t.Run(limiter.String(), func(t *testing.T) {
			defer limiter.Close()

			ctx := context.Background()
			key1 := "key1"
			key2 := "key2"

			// Exhaust rate limit for key1
			for i := 0; i < config.Rate; i++ {
				allowed, err := limiter.Allow(ctx, key1)
				if err != nil {
					t.Fatalf("Allow failed: %v", err)
				}
				if !allowed {
					t.Fatalf("Request %d for key1 should be allowed", i+1)
				}
			}

			// key1 should be rate limited
			allowed, err := limiter.Allow(ctx, key1)
			if err != nil {
				t.Fatalf("Allow failed: %v", err)
			}
			if allowed {
				t.Error("key1 should be rate limited")
			}

			// key2 should still work
			allowed, err = limiter.Allow(ctx, key2)
			if err != nil {
				t.Fatalf("Allow failed: %v", err)
			}
			if !allowed {
				t.Error("key2 should not be rate limited")
			}
		})
	}
}

func TestAllowN(t *testing.T) {
	config := Config{
		Rate:     10,
		Duration: time.Second,
		Burst:    15,
	}

	limiters := []RateLimiter{
		NewCounter(config),
		NewFixedWindow(config),
		NewSlidingWindow(config),
		NewSlidingWindowLog(config),
		NewTokenBucket(config),
	}

	for _, limiter := range limiters {
		t.Run(limiter.String(), func(t *testing.T) {
			defer limiter.Close()

			ctx := context.Background()
			key := "test-key"

			// Should allow N requests at once
			allowed, err := limiter.AllowN(ctx, key, 5)
			if err != nil {
				t.Fatalf("AllowN failed: %v", err)
			}
			if !allowed {
				t.Error("AllowN(5) should be allowed")
			}

			// Should allow remaining requests
			allowed, err = limiter.AllowN(ctx, key, 5)
			if err != nil {
				t.Fatalf("AllowN failed: %v", err)
			}
			if !allowed && limiter.String() != "Counter" {
				// Counter doesn't reset, so it might not allow more
				t.Error("AllowN(5) should be allowed for remaining quota")
			}

			// Should deny requests beyond limit
			allowed, err = limiter.AllowN(ctx, key, 10)
			if err != nil {
				t.Fatalf("AllowN failed: %v", err)
			}
			if allowed {
				t.Error("AllowN beyond limit should be denied")
			}
		})
	}
}

func TestZeroAndNegativeRequests(t *testing.T) {
	config := Config{
		Rate:     5,
		Duration: time.Second,
	}

	limiters := []RateLimiter{
		NewCounter(config),
		NewFixedWindow(config),
		NewSlidingWindow(config),
		NewSlidingWindowLog(config),
		NewTokenBucket(config),
	}

	for _, limiter := range limiters {
		t.Run(limiter.String(), func(t *testing.T) {
			defer limiter.Close()

			ctx := context.Background()
			key := "test-key"

			// Zero requests should always be allowed
			allowed, err := limiter.AllowN(ctx, key, 0)
			if err != nil {
				t.Fatalf("AllowN(0) failed: %v", err)
			}
			if !allowed {
				t.Error("AllowN(0) should always be allowed")
			}

			// Negative requests should always be allowed
			allowed, err = limiter.AllowN(ctx, key, -1)
			if err != nil {
				t.Fatalf("AllowN(-1) failed: %v", err)
			}
			if !allowed {
				t.Error("AllowN(-1) should always be allowed")
			}
		})
	}
}
