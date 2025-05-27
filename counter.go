package ratelimiters

import (
	"context"
	"sync"
	"time"
)

// Counter implements a very naive counter-based rate limiter
// This just counts requests without any time-based reset mechanism
// It's mainly useful for demonstrating the most basic approach
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
		return false, ErrNotSupported{Operation: "AllowN", Limiter: "Counter (closed)"}
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
	return ErrNotSupported{Operation: "Wait", Limiter: "Counter"}
}

func (c *Counter) WaitN(ctx context.Context, key string, n int) error {
	return ErrNotSupported{Operation: "WaitN", Limiter: "Counter"}
}

func (c *Counter) Reset(ctx context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return ErrNotSupported{Operation: "Reset", Limiter: "Counter (closed)"}
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
