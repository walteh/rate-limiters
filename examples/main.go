package main

import (
	"context"
	"fmt"
	"log"
	"time"

	ratelimiters "github.com/walteh/rate-limiters"
)

func main() {
	fmt.Println("Rate Limiter Examples")
	fmt.Println("====================")

	// Common configuration
	config := ratelimiters.Config{
		Rate:     5,               // 5 requests
		Duration: 2 * time.Second, // per 2 seconds
		Burst:    8,               // with burst of 8 (for token bucket)
	}

	// Demonstrate each rate limiter
	demonstrateCounter(config)
	demonstrateFixedWindow(config)
	demonstrateSlidingWindow(config)
	demonstrateSlidingWindowLog(config)
	demonstrateTokenBucket(config)

	// Demonstrate advanced features
	demonstrateWaitFunctionality(config)
	demonstrateMultipleKeys(config)
	demonstrateBurstTraffic(config)
}

func demonstrateCounter(config ratelimiters.Config) {
	fmt.Println("\n1. Counter Rate Limiter (Naive)")
	fmt.Println("- Simple counter, no time-based reset")
	fmt.Println("- Good for: Learning, simple quotas")
	fmt.Println("- Bad for: Time-based rate limiting")

	limiter := ratelimiters.NewCounter(config)
	defer limiter.Close()

	ctx := context.Background()
	key := "user123"

	fmt.Printf("Allowing %d requests:\n", config.Rate+2)
	for i := 0; i < config.Rate+2; i++ {
		allowed, err := limiter.Allow(ctx, key)
		if err != nil {
			log.Printf("Error: %v", err)
			continue
		}
		status := "✓ ALLOWED"
		if !allowed {
			status = "✗ DENIED"
		}
		fmt.Printf("  Request %d: %s\n", i+1, status)
	}
}

func demonstrateFixedWindow(config ratelimiters.Config) {
	fmt.Println("\n2. Fixed Window Rate Limiter")
	fmt.Println("- Resets counter at fixed intervals")
	fmt.Println("- Good for: Simple implementation, predictable reset")
	fmt.Println("- Bad for: Burst traffic at window boundaries")

	limiter := ratelimiters.NewFixedWindow(config)
	defer limiter.Close()

	ctx := context.Background()
	key := "user123"

	fmt.Printf("Allowing %d requests, then waiting for window reset:\n", config.Rate+1)
	for i := 0; i < config.Rate+1; i++ {
		allowed, err := limiter.Allow(ctx, key)
		if err != nil {
			log.Printf("Error: %v", err)
			continue
		}
		status := "✓ ALLOWED"
		if !allowed {
			status = "✗ DENIED"
		}
		fmt.Printf("  Request %d: %s\n", i+1, status)
	}

	fmt.Println("  Waiting for window to reset...")
	time.Sleep(config.Duration + 100*time.Millisecond)

	allowed, err := limiter.Allow(ctx, key)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	status := "✓ ALLOWED"
	if !allowed {
		status = "✗ DENIED"
	}
	fmt.Printf("  After reset: %s\n", status)
}

func demonstrateSlidingWindow(config ratelimiters.Config) {
	fmt.Println("\n3. Sliding Window Rate Limiter")
	fmt.Println("- Tracks individual request timestamps")
	fmt.Println("- Good for: Accurate rate limiting, smooth traffic")
	fmt.Println("- Bad for: Memory usage with many requests")

	limiter := ratelimiters.NewSlidingWindow(config)
	defer limiter.Close()

	ctx := context.Background()
	key := "user123"

	fmt.Printf("Making requests with delays:\n")
	for i := 0; i < config.Rate; i++ {
		allowed, err := limiter.Allow(ctx, key)
		if err != nil {
			log.Printf("Error: %v", err)
			continue
		}
		status := "✓ ALLOWED"
		if !allowed {
			status = "✗ DENIED"
		}
		fmt.Printf("  Request %d: %s\n", i+1, status)
		time.Sleep(200 * time.Millisecond)
	}

	// This should be denied
	allowed, err := limiter.Allow(ctx, key)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	status := "✓ ALLOWED"
	if !allowed {
		status = "✗ DENIED"
	}
	fmt.Printf("  Burst request: %s\n", status)
}

func demonstrateSlidingWindowLog(config ratelimiters.Config) {
	fmt.Println("\n4. Sliding Window Log Rate Limiter")
	fmt.Println("- Memory-efficient sliding window with circular buffer")
	fmt.Println("- Good for: Memory efficiency, accurate rate limiting")
	fmt.Println("- Bad for: Complex implementation")

	limiter := ratelimiters.NewSlidingWindowLog(config)
	defer limiter.Close()

	ctx := context.Background()
	key := "user123"

	fmt.Printf("Testing memory-efficient sliding window:\n")
	for i := 0; i < config.Rate+2; i++ {
		allowed, err := limiter.Allow(ctx, key)
		if err != nil {
			log.Printf("Error: %v", err)
			continue
		}
		status := "✓ ALLOWED"
		if !allowed {
			status = "✗ DENIED"
		}
		fmt.Printf("  Request %d: %s\n", i+1, status)
	}
}

func demonstrateTokenBucket(config ratelimiters.Config) {
	fmt.Println("\n5. Token Bucket Rate Limiter")
	fmt.Println("- Allows burst traffic up to bucket capacity")
	fmt.Println("- Good for: Handling bursts, smooth rate limiting")
	fmt.Println("- Bad for: Complex token calculation")

	limiter := ratelimiters.NewTokenBucket(config)
	defer limiter.Close()

	ctx := context.Background()
	key := "user123"

	fmt.Printf("Testing burst capacity (burst=%d):\n", config.Burst)
	for i := 0; i < config.Burst+2; i++ {
		allowed, err := limiter.Allow(ctx, key)
		if err != nil {
			log.Printf("Error: %v", err)
			continue
		}
		status := "✓ ALLOWED"
		if !allowed {
			status = "✗ DENIED"
		}
		fmt.Printf("  Burst request %d: %s\n", i+1, status)
	}

	fmt.Println("  Waiting for token refill...")
	time.Sleep(config.Duration + 100*time.Millisecond)

	allowed, err := limiter.Allow(ctx, key)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	status := "✓ ALLOWED"
	if !allowed {
		status = "✗ DENIED"
	}
	fmt.Printf("  After refill: %s\n", status)
}

func demonstrateWaitFunctionality(config ratelimiters.Config) {
	fmt.Println("\n6. Wait Functionality Demo")
	fmt.Println("- Some limiters support blocking until requests are allowed")

	// Only sliding window and token bucket support waiting
	limiters := []struct {
		name    string
		limiter ratelimiters.RateLimiter
	}{
		{"SlidingWindow", ratelimiters.NewSlidingWindow(config)},
		{"TokenBucket", ratelimiters.NewTokenBucket(config)},
	}

	for _, l := range limiters {
		fmt.Printf("\n%s Wait Demo:\n", l.name)
		defer l.limiter.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		key := "wait-demo"

		// Exhaust the rate limit
		for i := 0; i < config.Rate; i++ {
			l.limiter.Allow(ctx, key)
		}

		fmt.Println("  Rate limit exhausted, now waiting...")
		start := time.Now()
		err := l.limiter.Wait(ctx, key)
		elapsed := time.Since(start)

		if err != nil {
			fmt.Printf("  Wait failed: %v\n", err)
		} else {
			fmt.Printf("  Wait succeeded after %v\n", elapsed.Round(time.Millisecond))
		}
	}
}

func demonstrateMultipleKeys(config ratelimiters.Config) {
	fmt.Println("\n7. Multiple Keys Demo")
	fmt.Println("- Each key has independent rate limiting")

	limiter := ratelimiters.NewTokenBucket(config)
	defer limiter.Close()

	ctx := context.Background()
	users := []string{"alice", "bob", "charlie"}

	fmt.Println("Testing independent rate limits per user:")
	for _, user := range users {
		fmt.Printf("  User %s:\n", user)
		for i := 0; i < 3; i++ {
			allowed, err := limiter.Allow(ctx, user)
			if err != nil {
				log.Printf("Error: %v", err)
				continue
			}
			status := "✓ ALLOWED"
			if !allowed {
				status = "✗ DENIED"
			}
			fmt.Printf("    Request %d: %s\n", i+1, status)
		}
	}
}

func demonstrateBurstTraffic(config ratelimiters.Config) {
	fmt.Println("\n8. Burst Traffic Comparison")
	fmt.Println("- Comparing how different limiters handle burst traffic")

	limiters := []struct {
		name    string
		limiter ratelimiters.RateLimiter
	}{
		{"FixedWindow", ratelimiters.NewFixedWindow(config)},
		{"TokenBucket", ratelimiters.NewTokenBucket(config)},
	}

	for _, l := range limiters {
		fmt.Printf("\n%s burst handling:\n", l.name)
		defer l.limiter.Close()

		ctx := context.Background()
		key := "burst-test"

		// Simulate burst of 10 requests
		allowedCount := 0
		for i := 0; i < 10; i++ {
			allowed, err := l.limiter.Allow(ctx, key)
			if err != nil {
				log.Printf("Error: %v", err)
				continue
			}
			if allowed {
				allowedCount++
			}
		}
		fmt.Printf("  Allowed %d out of 10 burst requests\n", allowedCount)
	}
}
