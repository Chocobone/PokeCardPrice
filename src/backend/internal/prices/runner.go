package prices

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime/debug"
	"sync"
	"time"
)

// Runner 는 작업들을 고루틴으로 동시에 실행한다.
// 소스마다 호출 규칙(요청 간격·동시성)이 다르고 서로 다른 서버를 부르므로 동시에 돌려도 소스 예절을 어기지 않는다.
type Runner struct {
	sources []Source
}

// NewRunner 는 sources 를 실행하는 Runner 를 만든다.
func NewRunner(sources ...Source) *Runner { return &Runner{sources: sources} }

// Run 은 opt.Sources 에 해당하는 작업을 모두 동시에 실행하고 끝날 때까지 기다린다.
// 작업 하나가 실패하거나 패닉이 나도 나머지는 끝까지 실행한다. 결과는 sources 순서이다.
// 오류는 실패한 작업들의 오류를 합친 것이다.
func (r *Runner) Run(ctx context.Context, opt Options) ([]Result, error) {
	var run []Source
	opt.done = map[string]chan struct{}{}
	for _, s := range r.sources {
		if _, dup := opt.done[s.Code()]; dup || !opt.wantSource(s.Code()) {
			continue
		}
		run = append(run, s)
		opt.done[s.Code()] = make(chan struct{})
	}
	results := make([]Result, len(run))
	errs := make([]error, len(run))
	var wg sync.WaitGroup
	for i, s := range run {
		wg.Go(func() {
			defer close(opt.done[s.Code()])
			results[i], errs[i] = collectSafely(ctx, s, opt)
		})
	}
	wg.Wait()

	for i, res := range results {
		if errs[i] != nil {
			slog.Error("price source failed", "source", res.Source, "result", res, "err", errs[i])
			errs[i] = fmt.Errorf("%s: %w", res.Source, errs[i])
		} else {
			slog.Info("price source done", "source", res.Source, "result", res)
		}
	}
	return results, errors.Join(errs...)
}

// collectSafely 는 작업 하나를 실행하고 소요 시간을 채운다. 패닉은 오류로 바꾼다.
func collectSafely(ctx context.Context, s Source, opt Options) (res Result, err error) {
	start := time.Now()
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("panic: %v\n%s", p, debug.Stack())
		}
		res.Source = s.Code()
		res.Duration = time.Since(start)
	}()
	slog.Info("price source start", "source", s.Code())
	return s.Collect(ctx, opt)
}
