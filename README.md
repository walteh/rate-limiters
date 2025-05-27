# Rate Limiters

A comprehensive Go library implementing various rate limiting algorithms for learning and reference purposes. This project demonstrates different approaches to rate limiting, from naive implementations to advanced algorithms like sliding windows and token buckets.

## Features

- **Multiple Algorithms**: Counter, Fixed Window, Sliding Window, Sliding Window Log, and Token Bucket
- **Common Interface**: All rate limiters implement the same interface for easy comparison and swapping
- **Comprehensive Testing**: Full test coverage with concurrent access tests
- **Benchmarks**: Performance comparisons between all implementations
- **Examples**: Detailed examples showing the behavior of each algorithm
- **Production Ready**: Thread-safe implementations with proper resource cleanup

## Rate Limiter Types

### 1. Counter (Naive)
The simplest rate limiter that just counts requests without any time-based reset.

**Pros:**
- Extremely simple implementation
- Very fast
- Good for learning basics

**Cons:**
- No time-based reset mechanism
- Not suitable for real rate limiting
- Once limit is reached, no more requests are allowed

**Use Cases:**
- Learning rate limiting concepts
- Simple quotas that don't reset
- Testing and development

### 2. Fixed Window
Divides time into fixed windows and counts requests per window.

**Pros:**
- Simple to understand and implement
- Predictable reset behavior
- Memory efficient

**Cons:**
- Vulnerable to burst traffic at window boundaries
- Can allow 2x the rate limit in worst case
- Not smooth rate limiting

**Use Cases:**
- Simple rate limiting requirements
- When predictable reset times are important
- Systems with low burst requirements

### 3. Sliding Window
Tracks individual request timestamps and maintains a sliding time window.

**Pros:**
- Accurate rate limiting
- Smooth traffic distribution
- No burst issues at boundaries
- Supports waiting for next available slot

**Cons:**
- Higher memory usage (stores all timestamps)
- More complex implementation
- Cleanup overhead

**Use Cases:**
- High-accuracy rate limiting
- APIs with strict rate requirements
- When smooth traffic distribution is important

### 4. Sliding Window Log
Memory-efficient sliding window using a circular buffer.

**Pros:**
- More memory efficient than basic sliding window
- Accurate rate limiting
- Smooth traffic distribution
- Supports waiting

**Cons:**
- Complex implementation
- Still requires storing timestamps
- Cleanup overhead

**Use Cases:**
- Memory-constrained environments
- High-accuracy rate limiting with efficiency concerns
- Large-scale systems with many keys

### 5. Token Bucket
Allows burst traffic up to bucket capacity with smooth refill rate.

**Pros:**
- Handles burst traffic gracefully
- Smooth rate limiting
- Configurable burst capacity
- Supports waiting
- Industry standard algorithm

**Cons:**
- More complex token calculation
- Requires understanding of burst vs rate concepts

**Use Cases:**
- APIs that need to handle bursts
- Network traffic shaping
- Production rate limiting
- When user experience during bursts matters

## Installation

```bash
go get github.com/walteh/rate-limiters
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "time"
    
    ratelimiters "github.com/walteh/rate-limiters"
)

func main() {
    // Configure rate limiter: 10 requests per second with burst of 15
    config := ratelimiters.Config{
        Rate:     10,
        Duration: time.Second,
        Burst:    15, // Only used by TokenBucket
    }
    
    // Create a token bucket rate limiter
    limiter := ratelimiters.NewTokenBucket(config)
    defer limiter.Close()
    
    ctx := context.Background()
    userID := "user123"
    
    // Check if request is allowed
    allowed, err := limiter.Allow(ctx, userID)
    if err != nil {
        panic(err)
    }
    
    if allowed {
        fmt.Println("Request allowed!")
        // Process the request
    } else {
        fmt.Println("Request rate limited")
        // Return rate limit error to client
    }
}
```

## Interface

All rate limiters implement the `RateLimiter` interface:

```go
type RateLimiter interface {
    // Allow checks if a request should be allowed
    Allow(ctx context.Context, key string) (bool, error)
    
    // AllowN checks if N requests should be allowed
    AllowN(ctx context.Context, key string, n int) (bool, error)
    
    // Wait blocks until the next request is allowed (if supported)
    Wait(ctx context.Context, key string) error
    
    // WaitN blocks until N requests are allowed (if supported)
    WaitN(ctx context.Context, key string, n int) error
    
    // Reset clears the rate limiting state for the given key
    Reset(ctx context.Context, key string) error
    
    // Close cleans up any resources used by the rate limiter
    Close() error
    
    // String returns a human-readable description
    String() string
}
```

## Configuration

```go
type Config struct {
    // Rate is the number of requests allowed per Duration
    Rate int
    
    // Duration is the time window for the rate limit
    Duration time.Duration
    
    // Burst is the maximum number of requests that can be made in a burst
    // (only applicable to TokenBucket)
    Burst int
}
```

## Examples

### Basic Usage

```go
// Create different types of rate limiters
limiters := []ratelimiters.RateLimiter{
    ratelimiters.NewCounter(config),
    ratelimiters.NewFixedWindow(config),
    ratelimiters.NewSlidingWindow(config),
    ratelimiters.NewSlidingWindowLog(config),
    ratelimiters.NewTokenBucket(config),
}

for _, limiter := range limiters {
    defer limiter.Close()
    
    allowed, err := limiter.Allow(ctx, "user123")
    if err != nil {
        log.Printf("Error with %s: %v", limiter.String(), err)
        continue
    }
    
    fmt.Printf("%s: %v\n", limiter.String(), allowed)
}
```

### Waiting for Available Slots

```go
// Only SlidingWindow, SlidingWindowLog, and TokenBucket support waiting
limiter := ratelimiters.NewTokenBucket(config)
defer limiter.Close()

ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

// This will block until a request slot is available or context times out
err := limiter.Wait(ctx, "user123")
if err != nil {
    if err == context.DeadlineExceeded {
        fmt.Println("Timed out waiting for rate limit")
    } else {
        fmt.Printf("Wait failed: %v\n", err)
    }
} else {
    fmt.Println("Request slot available!")
    // Process the request
}
```

### Multiple Keys (Per-User Rate Limiting)

```go
limiter := ratelimiters.NewTokenBucket(config)
defer limiter.Close()

users := []string{"alice", "bob", "charlie"}

for _, user := range users {
    allowed, err := limiter.Allow(ctx, user)
    if err != nil {
        log.Printf("Error for user %s: %v", user, err)
        continue
    }
    
    if allowed {
        fmt.Printf("User %s: request allowed\n", user)
    } else {
        fmt.Printf("User %s: rate limited\n", user)
    }
}
```

### Batch Requests

```go
limiter := ratelimiters.NewTokenBucket(config)
defer limiter.Close()

// Try to allow 5 requests at once
allowed, err := limiter.AllowN(ctx, "user123", 5)
if err != nil {
    log.Printf("Error: %v", err)
    return
}

if allowed {
    fmt.Println("Batch of 5 requests allowed")
    // Process all 5 requests
} else {
    fmt.Println("Batch request denied - not enough quota")
}
```

## Running Examples

```bash
# Run the example program
go run examples/main.go

# Run tests
go test -v

# Run benchmarks
go test -bench=. -benchmem

# Run specific benchmark
go test -bench=BenchmarkRateLimiters -benchmem
```

## Benchmarks

The library includes comprehensive benchmarks comparing all implementations:

```bash
go test -bench=. -benchmem
```

Example benchmark results:
```
BenchmarkRateLimiters/Counter-8                 50000000    25.2 ns/op     0 B/op    0 allocs/op
BenchmarkRateLimiters/FixedWindow-8             20000000    65.1 ns/op     0 B/op    0 allocs/op
BenchmarkRateLimiters/SlidingWindow-8            5000000   285.3 ns/op    48 B/op    1 allocs/op
BenchmarkRateLimiters/SlidingWindowLog-8        10000000   156.7 ns/op     0 B/op    0 allocs/op
BenchmarkRateLimiters/TokenBucket-8             30000000    45.8 ns/op     0 B/op    0 allocs/op
```

## Error Handling

The library defines specific error types:

```go
// ErrNotSupported is returned when an operation is not supported
type ErrNotSupported struct {
    Operation string
    Limiter   string
}

// ErrRateLimited is returned when a request is rate limited
type ErrRateLimited struct {
    Key        string
    RetryAfter time.Duration
}
```

## Thread Safety

All rate limiter implementations are thread-safe and can be used concurrently from multiple goroutines.

## Resource Cleanup

All rate limiters that use background goroutines (FixedWindow, SlidingWindow, SlidingWindowLog, TokenBucket) implement proper cleanup. Always call `Close()` when done:

```go
limiter := ratelimiters.NewTokenBucket(config)
defer limiter.Close() // Important: cleanup resources
```

## Choosing the Right Algorithm

| Algorithm | Memory | Accuracy | Burst Handling | Complexity | Use Case |
|-----------|--------|----------|----------------|------------|----------|
| Counter | Low | Poor | Poor | Very Low | Learning, simple quotas |
| Fixed Window | Low | Good | Poor | Low | Simple rate limiting |
| Sliding Window | High | Excellent | Good | Medium | High accuracy needs |
| Sliding Window Log | Medium | Excellent | Good | High | Memory-efficient accuracy |
| Token Bucket | Low | Excellent | Excellent | Medium | Production systems |

**Recommendations:**
- **Learning**: Start with Counter and Fixed Window
- **Production APIs**: Use Token Bucket
- **High accuracy needs**: Use Sliding Window or Sliding Window Log
- **Memory constrained**: Use Fixed Window or Token Bucket
- **Burst traffic**: Use Token Bucket

## Contributing

This project is designed for learning and reference. Contributions are welcome, especially:

- Additional rate limiting algorithms
- Performance improvements
- Better examples and documentation
- Bug fixes

## License

MIT License - see LICENSE file for details.