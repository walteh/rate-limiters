package ratelimiters

import (
	"context"
	"sync"
	"time"
)

// SlidingWindow implements a sliding window rate limiter
// This tracks individual request timestamps and provides more accurate rate limiting
type SlidingWindow struct {
	config   Config
	mu       sync.RWMutex
	windows  map[string]*slidingWindowState
	stopChan chan struct{}
	closed   bool
}

type slidingWindowState struct {
	requests []time.Time
}

// NewSlidingWindow creates a new sliding window rate limiter
func NewSlidingWindow(config Config) *SlidingWindow {
	sw := &SlidingWindow{
		config:   config,
		windows:  make(map[string]*slidingWindowState),
		stopChan: make(chan struct{}),
	}

	// Start cleanup goroutine
	go sw.cleanup()

	return sw
}

func (sw *SlidingWindow) Allow(ctx context.Context, key string) (bool, error) {
	return sw.AllowN(ctx, key, 1)
}

func (sw *SlidingWindow) AllowN(ctx context.Context, key string, n int) (bool, error) {
	if n <= 0 {
		return true, nil
	}

	sw.mu.Lock()
	defer sw.mu.Unlock()

	if sw.closed {
		return false, ErrNotSupported{Operation: "AllowN", Limiter: "SlidingWindow (closed)"}
	}

	now := time.Now()
	windowStart := now.Add(-sw.config.Duration)

	state, exists := sw.windows[key]
	if !exists {
		state = &slidingWindowState{
			requests: make([]time.Time, 0),
		}
		sw.windows[key] = state
	}

	// Remove expired requests
	validRequests := make([]time.Time, 0, len(state.requests))
	for _, reqTime := range state.requests {
		if reqTime.After(windowStart) {
			validRequests = append(validRequests, reqTime)
		}
	}
	state.requests = validRequests

	// Check if we can allow N more requests
	if len(state.requests)+n <= sw.config.Rate {
		// Add N requests at the current time
		for i := 0; i < n; i++ {
			state.requests = append(state.requests, now)
		}
		return true, nil
	}

	return false, nil
}

func (sw *SlidingWindow) Wait(ctx context.Context, key string) error {
	return sw.WaitN(ctx, key, 1)
}

func (sw *SlidingWindow) WaitN(ctx context.Context, key string, n int) error {
	if n <= 0 {
		return nil
	}

	for {
		allowed, err := sw.AllowN(ctx, key, n)
		if err != nil {
			return err
		}
		if allowed {
			return nil
		}

		// Calculate how long to wait
		sw.mu.RLock()
		state, exists := sw.windows[key]
		if !exists || len(state.requests) == 0 {
			sw.mu.RUnlock()
			return nil // Should be able to proceed
		}

		// Find the oldest request that would need to expire
		now := time.Now()
		windowStart := now.Add(-sw.config.Duration)

		// Count valid requests
		validCount := 0
		oldestValidRequest := now
		for _, reqTime := range state.requests {
			if reqTime.After(windowStart) {
				validCount++
				if reqTime.Before(oldestValidRequest) {
					oldestValidRequest = reqTime
				}
			}
		}

		sw.mu.RUnlock()

		if validCount+n <= sw.config.Rate {
			continue // Try again
		}

		// Wait until the oldest request expires
		waitUntil := oldestValidRequest.Add(sw.config.Duration)
		waitDuration := time.Until(waitUntil)

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

func (sw *SlidingWindow) Reset(ctx context.Context, key string) error {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	if sw.closed {
		return ErrNotSupported{Operation: "Reset", Limiter: "SlidingWindow (closed)"}
	}

	delete(sw.windows, key)
	return nil
}

func (sw *SlidingWindow) Close() error {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	if !sw.closed {
		sw.closed = true
		close(sw.stopChan)
		sw.windows = nil
	}
	return nil
}

func (sw *SlidingWindow) String() string {
	return "SlidingWindow"
}

// cleanup removes expired requests periodically
func (sw *SlidingWindow) cleanup() {
	ticker := time.NewTicker(sw.config.Duration / 4) // Clean up 4 times per window
	defer ticker.Stop()

	for {
		select {
		case <-sw.stopChan:
			return
		case now := <-ticker.C:
			sw.mu.Lock()
			windowStart := now.Add(-sw.config.Duration)

			for key, state := range sw.windows {
				validRequests := make([]time.Time, 0, len(state.requests))
				for _, reqTime := range state.requests {
					if reqTime.After(windowStart) {
						validRequests = append(validRequests, reqTime)
					}
				}

				if len(validRequests) == 0 {
					delete(sw.windows, key)
				} else {
					state.requests = validRequests
				}
			}
			sw.mu.Unlock()
		}
	}
}
