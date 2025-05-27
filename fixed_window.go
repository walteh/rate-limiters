package ratelimiters

import (
	"context"
	"sync"
	"time"
)

// FixedWindow implements a naive fixed window rate limiter
// This is the simplest approach but has issues with burst traffic at window boundaries
//
// Algorithm: O(1) time, O(1) memory per key
// - Time-based window reset
// - Vulnerable to boundary attacks
// - Simple but flawed for security
//
// Visual representation:
//
//   ╔═══════════════════════════════════════════════════════════╗
//   ║  Target: 3 requests per 5 seconds                         ║
//   ║                                                           ║
//   ║  Naive Usage (3 requests in 5 seconds):                   ║
//   ║  ┌─────────────────────┐                                  ║
//   ║  │ ⓿ ➊ ➋ ③ ④┊➎ ➏ ➐ ⑧ ⑨ │                                  ║
//   ║  └─────────────────────┘                                  ║
//   ║                                                           ║
//   ║  Problematic Usage (4 requests in 5 seconds):             ║
//   ║  ┌─────────────────────┐                                  ║
//   ║  │ ⓿ ➊ ② ③ ➍┊➎ ➏ ➐ ⑧ ⑨ │                                  ║
//   ║  └─────────────────────┘                                  ║
//   ╚═══════════════════════════════════════════════════════════╝

type FixedWindow struct {
	config   Config
	mu       sync.RWMutex
	windows  map[string]*windowState
	stopChan chan struct{}
	closed   bool
}

type windowState struct {
	count     int
	windowEnd time.Time
}

// NewFixedWindow creates a new fixed window rate limiter
func NewFixedWindow(config Config) *FixedWindow {
	fw := &FixedWindow{
		config:   config,
		windows:  make(map[string]*windowState),
		stopChan: make(chan struct{}),
	}

	// Start cleanup goroutine
	go fw.cleanup()

	return fw
}

func (fw *FixedWindow) Allow(ctx context.Context, key string) (bool, error) {
	return fw.AllowN(ctx, key, 1)
}

func (fw *FixedWindow) AllowN(ctx context.Context, key string, n int) (bool, error) {
	if n <= 0 {
		return true, nil
	}

	fw.mu.Lock()
	defer fw.mu.Unlock()

	if fw.closed {
		return false, errorNotSupported()
	}

	now := time.Now()
	windowStart := now.Truncate(fw.config.Duration)
	windowEnd := windowStart.Add(fw.config.Duration)

	state, exists := fw.windows[key]
	if !exists || now.After(state.windowEnd) {
		// New window or expired window
		state = &windowState{
			count:     0,
			windowEnd: windowEnd,
		}
		fw.windows[key] = state
	}

	if state.count+n <= fw.config.Rate {
		state.count += n
		return true, nil
	}

	return false, nil
}

func (fw *FixedWindow) Wait(ctx context.Context, key string) error {
	return errorNotSupported()
}

func (fw *FixedWindow) WaitN(ctx context.Context, key string, n int) error {
	return errorNotSupported()
}

func (fw *FixedWindow) Reset(ctx context.Context, key string) error {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	if fw.closed {
		return errorNotSupported()
	}

	delete(fw.windows, key)
	return nil
}

func (fw *FixedWindow) Close() error {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	if !fw.closed {
		fw.closed = true
		close(fw.stopChan)
		fw.windows = nil
	}
	return nil
}

func (fw *FixedWindow) String() string {
	return "FixedWindow"
}

// cleanup removes expired windows periodically
func (fw *FixedWindow) cleanup() {
	ticker := time.NewTicker(fw.config.Duration)
	defer ticker.Stop()

	for {
		select {
		case <-fw.stopChan:
			return
		case now := <-ticker.C:
			fw.mu.Lock()
			for key, state := range fw.windows {
				if now.After(state.windowEnd) {
					delete(fw.windows, key)
				}
			}
			fw.mu.Unlock()
		}
	}
}
