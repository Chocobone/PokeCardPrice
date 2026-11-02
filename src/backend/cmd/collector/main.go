// Card-Market 시세 수집기 진입점.
//
// 기본은 일정(매일 05:30 KST)에 따라 수집 작업을 실행하며 계속 떠 있는다.
// -once 를 주면 모든 작업을 한 번씩 바로 실행하고 끝난다(개발·수동 수집용).
// COLLECTOR_INTERVAL(예: 30s)을 설정하면 일정 대신 그 간격으로 실행한다(개발용).
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata" // 시간대 데이터가 없는 실행 환경에서도 Asia/Seoul 을 읽기 위해 포함한다

	"github.com/go-co-op/gocron/v2"

	"github.com/Chocobone/PokeCardPrice/src/backend/internal/db"
	"github.com/Chocobone/PokeCardPrice/src/backend/internal/scheduler"
)

func main() {
	once := flag.Bool("once", false, "모든 작업을 한 번 실행하고 종료")
	flag.Parse()
	if err := run(*once); err != nil {
		slog.Error("collector stopped", "err", err)
		os.Exit(1)
	}
}

func run(once bool) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	loc, err := time.LoadLocation("Asia/Seoul")
	if err != nil {
		return err
	}
	dsn, err := db.DSNFromEnv()
	if err != nil {
		return err
	}
	pool, err := db.Open(ctx, dsn)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := db.Migrate(ctx, pool); err != nil {
		return err
	}

	daily := gocron.DailyJob(1, gocron.NewAtTimes(gocron.NewAtTime(5, 30, 0)))
	if v := os.Getenv("COLLECTOR_INTERVAL"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("COLLECTOR_INTERVAL: %w", err)
		}
		daily = gocron.DurationJob(d)
		slog.Warn("using COLLECTOR_INTERVAL instead of daily schedule", "interval", d)
	}

	jobs := []scheduler.Job{
		{
			Name: "collect-prices",
			When: daily,
			// 실제 수집은 #5(카탈로그), #6(해외 시세), #7(국내 거래가)에서 구현한다.
			Run: func(ctx context.Context) error {
				slog.Info("no collection tasks yet")
				return nil
			},
		},
	}

	if once {
		for _, j := range jobs {
			slog.Info("run once", "job", j.Name)
			if err := j.Run(ctx); err != nil {
				return fmt.Errorf("%s: %w", j.Name, err)
			}
		}
		return nil
	}

	s, err := scheduler.New(loc, 5*time.Minute, jobs...)
	if err != nil {
		return err
	}
	s.Start()
	<-ctx.Done()
	slog.Info("shutting down, waiting for running jobs")
	if err := s.Shutdown(); err != nil {
		return err
	}
	slog.Info("shutdown complete")
	return nil
}
