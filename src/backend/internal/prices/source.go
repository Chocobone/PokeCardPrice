// Package prices 는 해외 시세(TCGCSV, TCGdex Cardmarket)와 환율을 수집해 저장한다 (#6).
//
// 소스마다 Source 어댑터 하나를 두고, Runner 가 고루틴으로 동시에 실행한다.
// 한 소스의 실패·패닉은 다른 소스를 멈추지 않는다. 결과는 소스별 요약 로그로만 남긴다 (Q3).
// 가격은 소스 통화 그대로 저장하고, 원화는 조회할 때 fx_rates 로 환산한다.
// 설계 근거: mydocs/tech/price_collector.md
package prices

import (
	"context"
	"log/slog"
	"slices"
	"time"
)

// Source 는 수집 작업 하나이다. Collect 는 외부 호출부터 저장까지 끝까지 처리한다.
type Source interface {
	// Code 는 작업 이름이다. 가격 소스는 price_sources.code(tcgcsv, tcgdex_cardmarket), 환율은 fx.
	Code() string
	// Collect 는 부분 실패가 있어도 가능한 만큼 저장하고, 실패 항목을 Result.Failed 에 담는다.
	// 오류는 그 소스 전체가 진행할 수 없었거나 실패 항목이 남았을 때 돌려준다.
	Collect(ctx context.Context, opt Options) (Result, error)
}

// Options 는 수집 범위를 정한다 (개발용). 비어 있으면 전체.
type Options struct {
	// Sources 는 실행할 작업 Code 목록이다.
	Sources []string
	// Languages 는 카드 언어(en, ja)이다.
	Languages []string
	// SetIDs 는 TCGdex 세트 ID 이다 (언어 공통).
	SetIDs []string
}

// wantSource 는 code 작업을 실행할지 알려 준다.
func (o Options) wantSource(code string) bool {
	return len(o.Sources) == 0 || slices.Contains(o.Sources, code)
}

// Result 는 작업 하나의 실행 요약이다.
type Result struct {
	Source   string
	Fetched  int      // 받은 가격(환율) 행
	Linked   int      // 카드 변형과 연결된 행
	Saved    int      // 저장한 행
	Skipped  int      // 연결하지 못했거나 값이 없어 저장하지 않은 행
	Failed   []string // 실패한 항목 (재시도 후에도 남은 것)
	Duration time.Duration
}

// LogValue 는 요약 로그에 남길 필드이다. 실패 항목은 앞 20개만 남긴다.
func (r Result) LogValue() slog.Value {
	attrs := []slog.Attr{
		slog.Int("fetched", r.Fetched),
		slog.Int("linked", r.Linked),
		slog.Int("saved", r.Saved),
		slog.Int("skipped", r.Skipped),
		slog.Int("failed", len(r.Failed)),
		slog.Duration("duration", r.Duration.Round(time.Millisecond)),
	}
	if len(r.Failed) > 0 {
		attrs = append(attrs, slog.Any("failed_items", r.Failed[:min(len(r.Failed), 20)]))
	}
	return slog.GroupValue(attrs...)
}
