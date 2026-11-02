package scheduler

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-co-op/gocron/v2"
)

// 작업이 간격보다 오래 걸려도 겹치지 않고, Shutdown 이 진행 중인 작업을 기다리며 ctx 를 취소하는지 확인한다.
func TestNoOverlapAndGracefulShutdown(t *testing.T) {
	var running, maxRunning, done atomic.Int32
	var canceled atomic.Bool
	started := make(chan struct{}, 10)

	s, err := New(time.UTC, 5*time.Second, Job{
		Name: "slow",
		When: gocron.DurationJob(50 * time.Millisecond),
		Run: func(ctx context.Context) error {
			n := running.Add(1)
			if n > maxRunning.Load() {
				maxRunning.Store(n)
			}
			started <- struct{}{}
			select {
			case <-time.After(300 * time.Millisecond):
			case <-ctx.Done():
				canceled.Store(true)
				time.Sleep(100 * time.Millisecond) // 정리 작업 흉내
			}
			running.Add(-1)
			done.Add(1)
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	s.Start()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("job did not start")
	}
	time.Sleep(150 * time.Millisecond) // 다음 틱이 여러 번 지나가도록 둔다

	if err := s.Shutdown(); err != nil {
		t.Fatal(err)
	}
	if m := maxRunning.Load(); m != 1 {
		t.Errorf("max concurrent runs = %d, want 1", m)
	}
	if r := running.Load(); r != 0 {
		t.Errorf("still running after Shutdown: %d", r)
	}
	if done.Load() < 1 {
		t.Error("Shutdown returned before the running job finished")
	}
	if !canceled.Load() {
		t.Error("job context was not canceled on Shutdown")
	}
}

func TestNewRejectsInvalidJob(t *testing.T) {
	_, err := New(time.UTC, time.Second, Job{
		Name: "bad",
		When: gocron.CronJob("not a cron", false),
		Run:  func(context.Context) error { return nil },
	})
	if err == nil {
		t.Fatal("want error for invalid cron expression")
	}
}
