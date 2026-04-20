// Package syncx is a minimal, dependency-free subset of golang.org/x/sync
// (errgroup + singleflight) sufficient for this project's needs.
package syncx

import (
	"context"
	"sync"
)

// -------------------- errgroup --------------------

// Group runs goroutines and returns the first error, cancelling its context on error.
type Group struct {
	wg      sync.WaitGroup
	errOnce sync.Once
	err     error
	cancel  func()
}

// WithContext returns a Group whose context is cancelled on the first error or on Wait.
func WithContext(ctx context.Context) (*Group, context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	return &Group{cancel: cancel}, ctx
}

// Go launches f as a goroutine within the group.
func (g *Group) Go(f func() error) {
	g.wg.Add(1)
	go func() {
		defer g.wg.Done()
		if err := f(); err != nil {
			g.errOnce.Do(func() {
				g.err = err
				if g.cancel != nil {
					g.cancel()
				}
			})
		}
	}()
}

// Wait blocks until all goroutines have completed.
func (g *Group) Wait() error {
	g.wg.Wait()
	if g.cancel != nil {
		g.cancel()
	}
	return g.err
}

// -------------------- singleflight --------------------

type call struct {
	wg  sync.WaitGroup
	val any
	err error
}

// SingleFlight coalesces concurrent duplicate calls by key.
type SingleFlight struct {
	mu    sync.Mutex
	calls map[string]*call
}

// Do runs fn once per key at a time; duplicate callers wait for the first.
func (s *SingleFlight) Do(key string, fn func() (any, error)) (any, error, bool) {
	s.mu.Lock()
	if s.calls == nil {
		s.calls = make(map[string]*call)
	}
	if c, ok := s.calls[key]; ok {
		s.mu.Unlock()
		c.wg.Wait()
		return c.val, c.err, true
	}
	c := &call{}
	c.wg.Add(1)
	s.calls[key] = c
	s.mu.Unlock()

	c.val, c.err = fn()
	c.wg.Done()

	s.mu.Lock()
	delete(s.calls, key)
	s.mu.Unlock()
	return c.val, c.err, false
}
