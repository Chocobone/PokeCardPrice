// Package scheduler 는 수집 작업의 일정 등록과 실행을 담당한다(gocron v2).
package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-co-op/gocron/v2"
)

// Job 은 예약 실행할 작업이다.
type Job struct {
	Name string
	// When 은 실행 일정이다. 예: gocron.DailyJob(1, gocron.NewAtTimes(gocron.NewAtTime(5, 30, 0)))
	When gocron.JobDefinition
	// Run 은 작업 본문이다. 종료 시 ctx 가 취소되므로, 오래 걸리는 작업은 ctx 를 확인해 멈출 수 있다.
	Run func(ctx context.Context) error
}

// Scheduler 는 gocron 스케줄러를 감싼다.
type Scheduler struct {
	s gocron.Scheduler
}

// New 는 loc 시간대 기준 스케줄러를 만들고 작업을 등록한다.
// 같은 작업은 겹쳐 실행되지 않으며(이전 실행이 끝나지 않았으면 이번 실행을 건너뜀),
// 종료 시 진행 중인 작업을 stopTimeout 까지 기다린다.
func New(loc *time.Location, stopTimeout time.Duration, jobs ...Job) (*Scheduler, error) {
	s, err := gocron.NewScheduler(gocron.WithLocation(loc), gocron.WithStopTimeout(stopTimeout))
	if err != nil {
		return nil, err
	}
	for _, j := range jobs {
		job := j
		_, err := s.NewJob(job.When,
			gocron.NewTask(func(ctx context.Context) {
				start := time.Now()
				slog.Info("job start", "job", job.Name)
				if err := job.Run(ctx); err != nil {
					slog.Error("job failed", "job", job.Name, "err", err, "duration", time.Since(start))
					return
				}
				slog.Info("job done", "job", job.Name, "duration", time.Since(start))
			}),
			gocron.WithName(job.Name),
			gocron.WithSingletonMode(gocron.LimitModeReschedule),
		)
		if err != nil {
			_ = s.Shutdown()
			return nil, fmt.Errorf("작업 %q 등록: %w", job.Name, err)
		}
	}
	return &Scheduler{s: s}, nil
}

// Start 는 일정에 따라 작업 실행을 시작하고, 각 작업의 다음 실행 시각을 기록한다.
func (s *Scheduler) Start() {
	s.s.Start()
	for _, j := range s.s.Jobs() {
		next, _ := j.NextRun()
		slog.Info("job scheduled", "job", j.Name(), "next_run", next)
	}
}

// Shutdown 은 새 실행을 멈추고 진행 중인 작업이 끝나기를 기다린다.
func (s *Scheduler) Shutdown() error {
	return s.s.Shutdown()
}
