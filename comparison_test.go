package ratelimiters

import (
	"context"
	"fmt"
	"runtime"
	"testing"
	"time"
)

// BenchmarkCoreAlgorithmDifferences tests the fundamental algorithmic operations
// that make each rate limiter unique, not just different usage scenarios

// BenchmarkTimeComplexity tests the core time complexity differences
func BenchmarkTimeComplexity(b *testing.B) {
	config := Config{Rate: 100, Duration: time.Second, Burst: 150}
	ctx := context.Background()
	key := "test"

	b.Run("Counter/O(1)_increment", func(b *testing.B) {
		limiter := NewCounter(config)
		defer limiter.Close()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Counter: Simple atomic increment - O(1)
			limiter.Allow(ctx, key)
		}
	})

	b.Run("FixedWindow/O(1)_with_time_check", func(b *testing.B) {
		limiter := NewFixedWindow(config)
		defer limiter.Close()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// FixedWindow: Time check + atomic increment - O(1)
			limiter.Allow(ctx, key)
		}
	})

	b.Run("SlidingWindow/O(n)_timestamp_cleanup", func(b *testing.B) {
		limiter := NewSlidingWindow(config)
		defer limiter.Close()
		// Pre-populate with timestamps to show O(n) cleanup
		for i := 0; i < 50; i++ {
			limiter.Allow(ctx, key)
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// SlidingWindow: Must scan and remove old timestamps - O(n)
			limiter.Allow(ctx, key)
		}
	})

	b.Run("TokenBucket/O(1)_token_calculation", func(b *testing.B) {
		limiter := NewTokenBucket(config)
		defer limiter.Close()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// TokenBucket: Mathematical token calculation - O(1)
			limiter.Allow(ctx, key)
		}
	})
}

// BenchmarkMemoryComplexity tests how memory usage scales with requests
func BenchmarkMemoryComplexity(b *testing.B) {
	config := Config{Rate: 100, Duration: time.Second, Burst: 150}
	ctx := context.Background()
	key := "test"

	b.Run("Counter/O(1)_memory", func(b *testing.B) {
		limiter := NewCounter(config)
		defer limiter.Close()

		var m1, m2 runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m1)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Counter: Just stores a single integer per key - O(1) memory
			limiter.Allow(ctx, key)
		}
		b.StopTimer()

		runtime.GC()
		runtime.ReadMemStats(&m2)
		b.ReportMetric(float64(m2.Alloc-m1.Alloc), "bytes_per_op")
	})

	b.Run("FixedWindow/O(1)_memory", func(b *testing.B) {
		limiter := NewFixedWindow(config)
		defer limiter.Close()

		var m1, m2 runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m1)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// FixedWindow: Stores count + window start time - O(1) memory
			limiter.Allow(ctx, key)
		}
		b.StopTimer()

		runtime.GC()
		runtime.ReadMemStats(&m2)
		b.ReportMetric(float64(m2.Alloc-m1.Alloc), "bytes_per_op")
	})

	b.Run("SlidingWindow/O(n)_memory", func(b *testing.B) {
		limiter := NewSlidingWindow(config)
		defer limiter.Close()

		var m1, m2 runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m1)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// SlidingWindow: Stores every request timestamp - O(n) memory
			limiter.Allow(ctx, key)
		}
		b.StopTimer()

		runtime.GC()
		runtime.ReadMemStats(&m2)
		b.ReportMetric(float64(m2.Alloc-m1.Alloc), "bytes_per_op")
	})

	b.Run("TokenBucket/O(1)_memory", func(b *testing.B) {
		limiter := NewTokenBucket(config)
		defer limiter.Close()

		var m1, m2 runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m1)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// TokenBucket: Stores token count + last refill time - O(1) memory
			limiter.Allow(ctx, key)
		}
		b.StopTimer()

		runtime.GC()
		runtime.ReadMemStats(&m2)
		b.ReportMetric(float64(m2.Alloc-m1.Alloc), "bytes_per_op")
	})
}

// BenchmarkDataStructureOperations tests the core data structure operations
func BenchmarkDataStructureOperations(b *testing.B) {
	config := Config{Rate: 100, Duration: time.Second, Burst: 150}
	ctx := context.Background()
	key := "test"

	b.Run("Counter/atomic_increment", func(b *testing.B) {
		limiter := NewCounter(config)
		defer limiter.Close()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Core operation: atomic integer increment
			limiter.Allow(ctx, key)
		}
	})

	b.Run("FixedWindow/time_based_reset", func(b *testing.B) {
		limiter := NewFixedWindow(config)
		defer limiter.Close()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Core operation: time comparison + conditional reset + increment
			limiter.Allow(ctx, key)
		}
	})

	b.Run("SlidingWindow/slice_append_and_filter", func(b *testing.B) {
		limiter := NewSlidingWindow(config)
		defer limiter.Close()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Core operation: slice append + filter old timestamps
			limiter.Allow(ctx, key)
		}
	})

	b.Run("TokenBucket/mathematical_refill", func(b *testing.B) {
		limiter := NewTokenBucket(config)
		defer limiter.Close()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Core operation: time-based mathematical token calculation
			limiter.Allow(ctx, key)
		}
	})
}

// BenchmarkAlgorithmicAccuracy tests how accurately each algorithm enforces limits
func BenchmarkAlgorithmicAccuracy(b *testing.B) {
	config := Config{Rate: 10, Duration: 100 * time.Millisecond, Burst: 15}
	ctx := context.Background()
	key := "test"

	// Test each algorithm's accuracy in enforcing the rate limit
	algorithms := map[string]func() RateLimiter{
		"Counter":       func() RateLimiter { return NewCounter(config) },
		"FixedWindow":   func() RateLimiter { return NewFixedWindow(config) },
		"SlidingWindow": func() RateLimiter { return NewSlidingWindow(config) },
		"TokenBucket":   func() RateLimiter { return NewTokenBucket(config) },
	}

	for name, factory := range algorithms {
		b.Run(name, func(b *testing.B) {
			limiter := factory()
			defer limiter.Close()

			b.ResetTimer()

			allowedCount := 0
			for i := 0; i < b.N; i++ {
				if i%100 == 0 {
					// Reset every 100 iterations to test fresh windows
					limiter.Reset(ctx, key)
				}

				if allowed, _ := limiter.Allow(ctx, key); allowed {
					allowedCount++
				}
			}

			// Report how many requests were allowed vs theoretical maximum
			theoreticalMax := (b.N / 100) * config.Rate // Rough calculation
			accuracy := float64(allowedCount) / float64(theoreticalMax) * 100
			b.ReportMetric(accuracy, "accuracy%")
		})
	}
}

// BenchmarkConcurrencyModel tests the concurrency handling differences
func BenchmarkConcurrencyModel(b *testing.B) {
	config := Config{Rate: 1000, Duration: time.Second, Burst: 1500}
	ctx := context.Background()
	key := "test"

	b.Run("Counter/simple_mutex", func(b *testing.B) {
		limiter := NewCounter(config)
		defer limiter.Close()
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				// Simple mutex protection around counter
				limiter.Allow(ctx, key)
			}
		})
	})

	b.Run("FixedWindow/mutex_with_time_logic", func(b *testing.B) {
		limiter := NewFixedWindow(config)
		defer limiter.Close()
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				// Mutex protection around time checking + counter
				limiter.Allow(ctx, key)
			}
		})
	})

	b.Run("SlidingWindow/mutex_with_slice_operations", func(b *testing.B) {
		limiter := NewSlidingWindow(config)
		defer limiter.Close()
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				// Mutex protection around slice append/filter operations
				limiter.Allow(ctx, key)
			}
		})
	})

	b.Run("TokenBucket/mutex_with_math", func(b *testing.B) {
		limiter := NewTokenBucket(config)
		defer limiter.Close()
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				// Mutex protection around mathematical calculations
				limiter.Allow(ctx, key)
			}
		})
	})
}

// BenchmarkStateManagement tests how each algorithm manages internal state
func BenchmarkStateManagement(b *testing.B) {
	config := Config{Rate: 100, Duration: time.Second, Burst: 150}
	ctx := context.Background()

	b.Run("Counter/no_cleanup", func(b *testing.B) {
		limiter := NewCounter(config)
		defer limiter.Close()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("user-%d", i%1000)
			// Counter never cleans up old keys - state grows forever
			limiter.Allow(ctx, key)
		}
	})

	b.Run("FixedWindow/periodic_cleanup", func(b *testing.B) {
		limiter := NewFixedWindow(config)
		defer limiter.Close()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("user-%d", i%1000)
			// FixedWindow has background cleanup goroutine
			limiter.Allow(ctx, key)
		}
	})

	b.Run("SlidingWindow/continuous_cleanup", func(b *testing.B) {
		limiter := NewSlidingWindow(config)
		defer limiter.Close()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("user-%d", i%1000)
			// SlidingWindow cleans up on every request + background cleanup
			limiter.Allow(ctx, key)
		}
	})

	b.Run("TokenBucket/lazy_cleanup", func(b *testing.B) {
		limiter := NewTokenBucket(config)
		defer limiter.Close()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("user-%d", i%1000)
			// TokenBucket has background cleanup goroutine
			limiter.Allow(ctx, key)
		}
	})
}

// BenchmarkMathematicalOperations tests the computational complexity
func BenchmarkMathematicalOperations(b *testing.B) {
	config := Config{Rate: 100, Duration: time.Second, Burst: 150}
	ctx := context.Background()
	key := "test"

	b.Run("Counter/no_math", func(b *testing.B) {
		limiter := NewCounter(config)
		defer limiter.Close()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// No mathematical operations, just increment
			limiter.Allow(ctx, key)
		}
	})

	b.Run("FixedWindow/time_division", func(b *testing.B) {
		limiter := NewFixedWindow(config)
		defer limiter.Close()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Time division to determine current window
			limiter.Allow(ctx, key)
		}
	})

	b.Run("SlidingWindow/time_subtraction_filtering", func(b *testing.B) {
		limiter := NewSlidingWindow(config)
		defer limiter.Close()
		// Pre-populate to show filtering cost
		for i := 0; i < 50; i++ {
			limiter.Allow(ctx, key)
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Time subtraction for each timestamp during filtering
			limiter.Allow(ctx, key)
		}
	})

	b.Run("TokenBucket/rate_multiplication", func(b *testing.B) {
		limiter := NewTokenBucket(config)
		defer limiter.Close()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Rate * elapsed time calculation for token refill
			limiter.Allow(ctx, key)
		}
	})
}

// BenchmarkWaitImplementation tests the algorithmic differences in Wait()
func BenchmarkWaitImplementation(b *testing.B) {
	config := Config{Rate: 10, Duration: 100 * time.Millisecond, Burst: 15}

	b.Run("SlidingWindow/timestamp_based_wait", func(b *testing.B) {
		limiter := NewSlidingWindow(config)
		defer limiter.Close()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
			key := fmt.Sprintf("user-%d", i%10)

			// Exhaust the limiter
			for j := 0; j < config.Rate+2; j++ {
				limiter.Allow(ctx, key)
			}

			// SlidingWindow: Calculates wait time based on oldest timestamp
			start := time.Now()
			limiter.Wait(ctx, key)
			waitTime := time.Since(start)

			cancel()
			b.ReportMetric(float64(waitTime.Nanoseconds()), "wait_ns")
		}
	})

	b.Run("TokenBucket/mathematical_wait", func(b *testing.B) {
		limiter := NewTokenBucket(config)
		defer limiter.Close()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
			key := fmt.Sprintf("user-%d", i%10)

			// Exhaust the limiter
			for j := 0; j < config.Burst+2; j++ {
				limiter.Allow(ctx, key)
			}

			// TokenBucket: Calculates wait time mathematically
			start := time.Now()
			limiter.Wait(ctx, key)
			waitTime := time.Since(start)

			cancel()
			b.ReportMetric(float64(waitTime.Nanoseconds()), "wait_ns")
		}
	})
}
