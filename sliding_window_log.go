package ratelimiters

import (
	"context"
	"sync"
	"time"
)

// SlidingWindowLog implements a sliding window rate limiter using a more efficient log structure
// This is similar to SlidingWindow but uses a circular buffer for better memory efficiency
type SlidingWindowLog struct {
	config   Config
	mu       sync.RWMutex
	windows  map[string]*slidingLogState
	stopChan chan struct{}
	closed   bool
}

type slidingLogState struct {
	requests []time.Time
	head     int // Points to the oldest request
	count    int // Number of valid requests
}

// NewSlidingWindowLog creates a new sliding window log rate limiter
func NewSlidingWindowLog(config Config) *SlidingWindowLog {
	swl := &SlidingWindowLog{
		config:   config,
		windows:  make(map[string]*slidingLogState),
		stopChan: make(chan struct{}),
	}

	// Start cleanup goroutine
	go swl.cleanup()

	return swl
}

func (swl *SlidingWindowLog) Allow(ctx context.Context, key string) (bool, error) {
	return swl.AllowN(ctx, key, 1)
}

func (swl *SlidingWindowLog) AllowN(ctx context.Context, key string, n int) (bool, error) {
	if n <= 0 {
		return true, nil
	}

	swl.mu.Lock()
	defer swl.mu.Unlock()

	if swl.closed {
		return false, ErrNotSupported{Operation: "AllowN", Limiter: "SlidingWindowLog (closed)"}
	}

	now := time.Now()
	windowStart := now.Add(-swl.config.Duration)

	state, exists := swl.windows[key]
	if !exists {
		state = &slidingLogState{
			requests: make([]time.Time, swl.config.Rate*2), // Pre-allocate with some buffer
			head:     0,
			count:    0,
		}
		swl.windows[key] = state
	}

	// Clean expired requests
	swl.cleanExpiredRequests(state, windowStart)

	// Check if we can allow N more requests
	if state.count+n <= swl.config.Rate {
		// Add N requests at the current time
		for i := 0; i < n; i++ {
			swl.addRequest(state, now)
		}
		return true, nil
	}

	return false, nil
}

func (swl *SlidingWindowLog) Wait(ctx context.Context, key string) error {
	return swl.WaitN(ctx, key, 1)
}

func (swl *SlidingWindowLog) WaitN(ctx context.Context, key string, n int) error {
	if n <= 0 {
		return nil
	}

	for {
		allowed, err := swl.AllowN(ctx, key, n)
		if err != nil {
			return err
		}
		if allowed {
			return nil
		}

		// Calculate how long to wait
		swl.mu.RLock()
		state, exists := swl.windows[key]
		if !exists || state.count == 0 {
			swl.mu.RUnlock()
			return nil // Should be able to proceed
		}

		// Find the oldest request that would need to expire
		now := time.Now()
		windowStart := now.Add(-swl.config.Duration)

		oldestValidRequest := now
		found := false

		for i := 0; i < state.count; i++ {
			idx := (state.head + i) % len(state.requests)
			reqTime := state.requests[idx]
			if reqTime.After(windowStart) {
				if !found || reqTime.Before(oldestValidRequest) {
					oldestValidRequest = reqTime
					found = true
				}
			}
		}

		swl.mu.RUnlock()

		if !found {
			continue // Try again
		}

		// Wait until the oldest request expires
		waitUntil := oldestValidRequest.Add(swl.config.Duration)
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

func (swl *SlidingWindowLog) Reset(ctx context.Context, key string) error {
	swl.mu.Lock()
	defer swl.mu.Unlock()

	if swl.closed {
		return ErrNotSupported{Operation: "Reset", Limiter: "SlidingWindowLog (closed)"}
	}

	delete(swl.windows, key)
	return nil
}

func (swl *SlidingWindowLog) Close() error {
	swl.mu.Lock()
	defer swl.mu.Unlock()

	if !swl.closed {
		swl.closed = true
		close(swl.stopChan)
		swl.windows = nil
	}
	return nil
}

func (swl *SlidingWindowLog) String() string {
	return "SlidingWindowLog"
}

// addRequest adds a new request to the log
// Must be called with lock held
func (swl *SlidingWindowLog) addRequest(state *slidingLogState, reqTime time.Time) {
	if state.count < len(state.requests) {
		// There's space in the buffer
		idx := (state.head + state.count) % len(state.requests)
		state.requests[idx] = reqTime
		state.count++
	} else {
		// Buffer is full, overwrite the oldest entry
		state.requests[state.head] = reqTime
		state.head = (state.head + 1) % len(state.requests)
	}
}

// cleanExpiredRequests removes expired requests from the log
// Must be called with lock held
func (swl *SlidingWindowLog) cleanExpiredRequests(state *slidingLogState, windowStart time.Time) {
	newCount := 0
	newHead := state.head

	for i := 0; i < state.count; i++ {
		idx := (state.head + i) % len(state.requests)
		if state.requests[idx].After(windowStart) {
			// This request is still valid
			if newCount != i {
				// Move it to the correct position
				newIdx := (newHead + newCount) % len(state.requests)
				state.requests[newIdx] = state.requests[idx]
			}
			newCount++
		} else if newCount == 0 {
			// All requests so far are expired, move head
			newHead = (newHead + 1) % len(state.requests)
		}
	}

	state.head = newHead
	state.count = newCount
}

// cleanup removes expired requests and unused windows periodically
func (swl *SlidingWindowLog) cleanup() {
	ticker := time.NewTicker(swl.config.Duration / 4) // Clean up 4 times per window
	defer ticker.Stop()

	for {
		select {
		case <-swl.stopChan:
			return
		case now := <-ticker.C:
			swl.mu.Lock()
			windowStart := now.Add(-swl.config.Duration)

			for key, state := range swl.windows {
				swl.cleanExpiredRequests(state, windowStart)

				if state.count == 0 {
					delete(swl.windows, key)
				}
			}
			swl.mu.Unlock()
		}
	}
}
