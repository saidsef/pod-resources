package main

import (
	"context"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"
)

func TestMonitorRunsTheCheckOnEveryTick(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var checks atomic.Int64
		done := make(chan struct{})
		go func() {
			defer close(done)
			monitor(ctx, 120*time.Second, func() { checks.Add(1) })
		}()

		synctest.Sleep(6*time.Minute + time.Second)
		synctest.Wait()

		if got := checks.Load(); got != 3 {
			t.Errorf("the check ran %d times in six minutes on a two minute interval, want 3", got)
		}

		cancel()
		<-done
	})
}

func TestMonitorWaitsForTheFirstInterval(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var checks atomic.Int64
		done := make(chan struct{})
		go func() {
			defer close(done)
			monitor(ctx, 120*time.Second, func() { checks.Add(1) })
		}()

		synctest.Sleep(119 * time.Second)
		synctest.Wait()

		if got := checks.Load(); got != 0 {
			t.Errorf("the check ran %d times before the first interval elapsed, want 0", got)
		}

		cancel()
		<-done
	})
}

func TestMonitorStopsWhenTheContextIsCancelled(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var checks atomic.Int64
		done := make(chan struct{})
		go func() {
			defer close(done)
			monitor(ctx, 120*time.Second, func() { checks.Add(1) })
		}()

		synctest.Sleep(121 * time.Second)
		synctest.Wait()
		cancel()
		<-done

		ran := checks.Load()
		synctest.Sleep(10 * time.Minute)
		synctest.Wait()

		if got := checks.Load(); got != ran {
			t.Errorf("the check ran %d more times after cancellation, want none", got-ran)
		}
	})
}

func TestDurationSecondsParses(t *testing.T) {
	if _, err := time.ParseDuration(DURATION_SECONDS); err != nil {
		t.Errorf("DURATION_SECONDS %q does not parse: %v", DURATION_SECONDS, err)
	}
}
