package syncx

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestGroup_FirstErrorWins(t *testing.T) {
	g, _ := WithContext(context.Background())
	g.Go(func() error { time.Sleep(20 * time.Millisecond); return errors.New("late") })
	g.Go(func() error { return errors.New("early") })
	g.Go(func() error { return nil })
	if err := g.Wait(); err == nil {
		t.Fatal("expected an error")
	}
}

func TestGroup_CancelsContextOnError(t *testing.T) {
	g, ctx := WithContext(context.Background())
	g.Go(func() error { return errors.New("boom") })
	// Second goroutine waits on context cancellation.
	done := make(chan struct{})
	g.Go(func() error {
		select {
		case <-ctx.Done():
			close(done)
		case <-time.After(time.Second):
			t.Error("ctx never cancelled")
		}
		return nil
	})
	_ = g.Wait()
	select {
	case <-done:
	default:
		t.Fatal("watcher goroutine never observed cancellation")
	}
}

func TestSingleFlight_CoalescesDuplicates(t *testing.T) {
	var sf SingleFlight
	var calls int32
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, _ = sf.Do("k", func() (any, error) {
				atomic.AddInt32(&calls, 1)
				time.Sleep(20 * time.Millisecond)
				return "v", nil
			})
		}()
	}
	wg.Wait()
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("expected 1 call, got %d", got)
	}
}
