package ratelimiters

import (
	"context"
	"sync"
	"time"
)

// Counter implements a very naive counter-based rate limiter
// This just counts requests without any time-based reset mechanism
// It's mainly useful for demonstrating the most basic approach
//
// Algorithm: O(1) time, O(1) memory per key
// - Simple atomic increment
// - No time-based logic
// - No automatic cleanup
// - Fastest but least practical
//
// Visual representation:
//
//   ╔═══════════════════════════════════════════════════════════╗
//   ║                       COUNTER LIMITER                     ║
//   ╠═══════════════════════════════════════════════════════════╣
//   ║  Rate Limit: 5 requests                                   ║
//   ║                                                           ║
//   ║  Request 1: ✓  [➊ ② ③ ④ ⑤] count=1                        ║
//   ║  Request 2: ✓  [➊ ➋ ③ ④ ⑤] count=2                        ║
//   ║  Request 3: ✓  [➊ ➋ ➌ ④ ⑤] count=3                        ║
//   ║  Request 4: ✓  [➊ ➋ ➌ ➍ ⑤] count=4                        ║
//   ║  Request 5: ✓  [➊ ➋ ➌ ➍ ➎] count=5 (LIMIT REACHED)        ║
//   ║  Request 6: ✗  [➊ ➋ ➌ ➍ ➎] count=5 (BLOCKED!)             ║
//   ║  Request 7: ✗  [➊ ➋ ➌ ➍ ➎] count=5 (BLOCKED!)             ║
//   ║                                                           ║
//   ║  ⚠️  Counter NEVER resets automatically!                  ║
//   ║      Once limit is reached, ALL future requests fail      ║
//   ║      until manually reset                                 ║
//   ║                                                           ║
//   ╚═══════════════════════════════════════════════════════════╝

type Counter struct {
	config   Config
	mu       sync.RWMutex
	counters map[string]*counterState
	closed   bool
}

type counterState struct {
	count     int
	createdAt time.Time
}

// NewCounter creates a new counter-based rate limiter
func NewCounter(config Config) *Counter {
	return &Counter{
		config:   config,
		counters: make(map[string]*counterState),
	}
}

func (c *Counter) Allow(ctx context.Context, key string) (bool, error) {
	return c.AllowN(ctx, key, 1)
}

func (c *Counter) AllowN(ctx context.Context, key string, n int) (bool, error) {
	if n <= 0 {
		return true, nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return false, errorNotSupported()
	}

	state, exists := c.counters[key]
	if !exists {
		state = &counterState{
			count:     0,
			createdAt: time.Now(),
		}
		c.counters[key] = state
	}

	if state.count+n <= c.config.Rate {
		state.count += n
		return true, nil
	}

	return false, nil
}

func (c *Counter) Wait(ctx context.Context, key string) error {
	return errorNotSupported()
}

func (c *Counter) WaitN(ctx context.Context, key string, n int) error {
	return errorNotSupported()
}

func (c *Counter) Reset(ctx context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return errorClosed()
	}

	delete(c.counters, key)
	return nil
}

func (c *Counter) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.closed {
		c.closed = true
		c.counters = nil
	}
	return nil
}

func (c *Counter) String() string {
	return "Counter"
}
