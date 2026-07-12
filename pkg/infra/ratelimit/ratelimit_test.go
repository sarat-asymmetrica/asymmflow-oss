package ratelimit

import (
	"sync"
	"testing"
	"time"
)

func TestAllow_ExhaustsAndRefills(t *testing.T) {
	rl := New()
	refill := 50 * time.Millisecond

	for i := 0; i < 3; i++ {
		if !rl.Allow("k", 3, refill) {
			t.Fatalf("call %d should be allowed (bucket starts full)", i+1)
		}
	}
	if rl.Allow("k", 3, refill) {
		t.Fatalf("4th call should be refused (bucket empty)")
	}

	time.Sleep(refill + 20*time.Millisecond)
	if !rl.Allow("k", 3, refill) {
		t.Fatalf("call after refill interval should be allowed")
	}
}

func TestAllow_KeysAreIndependent(t *testing.T) {
	rl := New()
	refill := time.Minute

	for i := 0; i < 2; i++ {
		rl.Allow("a", 2, refill)
	}
	if rl.Allow("a", 2, refill) {
		t.Fatalf("key a should be exhausted")
	}
	if !rl.Allow("b", 2, refill) {
		t.Fatalf("key b must not be affected by key a")
	}
}

func TestAllow_ConcurrentCallersNeverExceedBudget(t *testing.T) {
	rl := New()
	const budget = 10

	var wg sync.WaitGroup
	allowed := make(chan struct{}, 100)
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if rl.Allow("shared", budget, time.Hour) {
				allowed <- struct{}{}
			}
		}()
	}
	wg.Wait()
	close(allowed)

	count := 0
	for range allowed {
		count++
	}
	if count != budget {
		t.Fatalf("allowed %d calls, budget is %d", count, budget)
	}
}
