package ratelimiters

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

// BenchmarkRateLimiters compares all rate limiter implementations
func BenchmarkRateLimiters(b *testing.B) {
	config := Config{
		Rate:     1000,
		Duration: time.Second,
		Burst:    1500,
	}

	benchmarks := []struct {
		name    string
		limiter RateLimiter
	}{
		{"Counter", NewCounter(config)},
		{"FixedWindow", NewFixedWindow(config)},
		{"SlidingWindow", NewSlidingWindow(config)},
		{"SlidingWindowLog", NewSlidingWindowLog(config)},
		{"TokenBucket", NewTokenBucket(config)},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			defer bm.limiter.Close()
			ctx := context.Background()
			key := "benchmark-key"

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				bm.limiter.Allow(ctx, key)
			}
		})
	}
}

// BenchmarkRateLimitersParallel tests concurrent performance
func BenchmarkRateLimitersParallel(b *testing.B) {
	config := Config{
		Rate:     10000,
		Duration: time.Second,
		Burst:    15000,
	}

	benchmarks := []struct {
		name    string
		limiter RateLimiter
	}{
		{"Counter", NewCounter(config)},
		{"FixedWindow", NewFixedWindow(config)},
		{"SlidingWindow", NewSlidingWindow(config)},
		{"SlidingWindowLog", NewSlidingWindowLog(config)},
		{"TokenBucket", NewTokenBucket(config)},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			defer bm.limiter.Close()
			ctx := context.Background()
			key := "benchmark-key"

			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					bm.limiter.Allow(ctx, key)
				}
			})
		})
	}
}

// BenchmarkRateLimitersMultipleKeys tests performance with multiple keys
func BenchmarkRateLimitersMultipleKeys(b *testing.B) {
	config := Config{
		Rate:     1000,
		Duration: time.Second,
		Burst:    1500,
	}

	benchmarks := []struct {
		name    string
		limiter RateLimiter
	}{
		{"Counter", NewCounter(config)},
		{"FixedWindow", NewFixedWindow(config)},
		{"SlidingWindow", NewSlidingWindow(config)},
		{"SlidingWindowLog", NewSlidingWindowLog(config)},
		{"TokenBucket", NewTokenBucket(config)},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			defer bm.limiter.Close()
			ctx := context.Background()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				key := fmt.Sprintf("key-%d", i%100) // 100 different keys
				bm.limiter.Allow(ctx, key)
			}
		})
	}
}

// BenchmarkAllowN tests performance of AllowN operations
func BenchmarkAllowN(b *testing.B) {
	config := Config{
		Rate:     10000,
		Duration: time.Second,
		Burst:    15000,
	}

	benchmarks := []struct {
		name    string
		limiter RateLimiter
	}{
		{"Counter", NewCounter(config)},
		{"FixedWindow", NewFixedWindow(config)},
		{"SlidingWindow", NewSlidingWindow(config)},
		{"SlidingWindowLog", NewSlidingWindowLog(config)},
		{"TokenBucket", NewTokenBucket(config)},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			defer bm.limiter.Close()
			ctx := context.Background()
			key := "benchmark-key"

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				bm.limiter.AllowN(ctx, key, 5)
			}
		})
	}
}

// BenchmarkMemoryUsage tests memory efficiency with many keys
func BenchmarkMemoryUsage(b *testing.B) {
	config := Config{
		Rate:     100,
		Duration: time.Second,
		Burst:    150,
	}

	benchmarks := []struct {
		name    string
		limiter RateLimiter
	}{
		{"Counter", NewCounter(config)},
		{"FixedWindow", NewFixedWindow(config)},
		{"SlidingWindow", NewSlidingWindow(config)},
		{"SlidingWindowLog", NewSlidingWindowLog(config)},
		{"TokenBucket", NewTokenBucket(config)},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			defer bm.limiter.Close()
			ctx := context.Background()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				key := fmt.Sprintf("unique-key-%d", i)
				bm.limiter.Allow(ctx, key)
			}
		})
	}
}

// BenchmarkHighContentionSingleKey tests performance under high contention
func BenchmarkHighContentionSingleKey(b *testing.B) {
	config := Config{
		Rate:     1000,
		Duration: time.Second,
		Burst:    1500,
	}

	benchmarks := []struct {
		name    string
		limiter RateLimiter
	}{
		{"Counter", NewCounter(config)},
		{"FixedWindow", NewFixedWindow(config)},
		{"SlidingWindow", NewSlidingWindow(config)},
		{"SlidingWindowLog", NewSlidingWindowLog(config)},
		{"TokenBucket", NewTokenBucket(config)},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			defer bm.limiter.Close()
			ctx := context.Background()
			key := "contention-key"

			var wg sync.WaitGroup
			numGoroutines := 100

			b.ResetTimer()
			for i := 0; i < numGoroutines; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for j := 0; j < b.N/numGoroutines; j++ {
						bm.limiter.Allow(ctx, key)
					}
				}()
			}
			wg.Wait()
		})
	}
}

// BenchmarkReset tests performance of reset operations
func BenchmarkReset(b *testing.B) {
	config := Config{
		Rate:     1000,
		Duration: time.Second,
		Burst:    1500,
	}

	benchmarks := []struct {
		name    string
		limiter RateLimiter
	}{
		{"Counter", NewCounter(config)},
		{"FixedWindow", NewFixedWindow(config)},
		{"SlidingWindow", NewSlidingWindow(config)},
		{"SlidingWindowLog", NewSlidingWindowLog(config)},
		{"TokenBucket", NewTokenBucket(config)},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			defer bm.limiter.Close()
			ctx := context.Background()

			// Pre-populate with some data
			for i := 0; i < 100; i++ {
				key := fmt.Sprintf("key-%d", i)
				bm.limiter.Allow(ctx, key)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				key := fmt.Sprintf("key-%d", i%100)
				bm.limiter.Reset(ctx, key)
			}
		})
	}
}

// BenchmarkMixedOperations tests realistic mixed workload
func BenchmarkMixedOperations(b *testing.B) {
	config := Config{
		Rate:     1000,
		Duration: time.Second,
		Burst:    1500,
	}

	benchmarks := []struct {
		name    string
		limiter RateLimiter
	}{
		{"Counter", NewCounter(config)},
		{"FixedWindow", NewFixedWindow(config)},
		{"SlidingWindow", NewSlidingWindow(config)},
		{"SlidingWindowLog", NewSlidingWindowLog(config)},
		{"TokenBucket", NewTokenBucket(config)},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			defer bm.limiter.Close()
			ctx := context.Background()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				key := fmt.Sprintf("key-%d", i%50)

				switch i % 10 {
				case 0, 1, 2, 3, 4, 5, 6, 7: // 80% Allow operations
					bm.limiter.Allow(ctx, key)
				case 8: // 10% AllowN operations
					bm.limiter.AllowN(ctx, key, 3)
				case 9: // 10% Reset operations
					bm.limiter.Reset(ctx, key)
				}
			}
		})
	}
}

// BenchmarkScalability tests how performance scales with number of keys
func BenchmarkScalability(b *testing.B) {
	config := Config{
		Rate:     100,
		Duration: time.Second,
		Burst:    150,
	}

	keyCountsToTest := []int{1, 10, 100, 1000, 10000}

	benchmarks := []struct {
		name    string
		limiter RateLimiter
	}{
		{"Counter", NewCounter(config)},
		{"FixedWindow", NewFixedWindow(config)},
		{"SlidingWindow", NewSlidingWindow(config)},
		{"SlidingWindowLog", NewSlidingWindowLog(config)},
		{"TokenBucket", NewTokenBucket(config)},
	}

	for _, keyCount := range keyCountsToTest {
		for _, bm := range benchmarks {
			name := fmt.Sprintf("%s-%dkeys", bm.name, keyCount)
			b.Run(name, func(b *testing.B) {
				defer bm.limiter.Close()
				ctx := context.Background()

				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					key := fmt.Sprintf("key-%d", i%keyCount)
					bm.limiter.Allow(ctx, key)
				}
			})
		}
	}
}
