// Package dbtest 는 DB 테스트용 헬퍼이다. 테스트 코드에서만 쓴다.
//
// internal/db 의 테스트도 이 패키지를 쓰므로, 순환 참조를 피하려고 internal/db 를 import 하지 않는다.
// 마이그레이션은 호출 측에서 db.Migrate 로 적용한다.
package dbtest

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Pool 은 TEST_DATABASE_URL 의 테스트 DB 에 연결하고, public 스키마를 비운 상태로 돌려준다.
//
//   - TEST_DATABASE_URL 이 없으면 테스트를 건너뛴다.
//   - suffix 가 있으면 DB 이름 뒤에 "_{suffix}" 를 붙인 DB 를 쓴다. go test ./... 는 패키지를 동시에
//     실행하므로, 패키지마다 다른 suffix 를 주어 서로의 스키마를 지우지 않게 한다.
//   - DB 가 없으면 같은 서버의 postgres DB 로 접속해 만든다.
//   - 개발 DB(DATABASE_URL)와 같은 DB 이면 데이터를 지우지 않도록 실패시킨다.
func Pool(t testing.TB, suffix string) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL 이 없어 DB 테스트를 건너뜁니다")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatalf("TEST_DATABASE_URL 파싱: %v", err)
	}
	if suffix != "" {
		cfg.ConnConfig.Database += "_" + suffix
	}
	cc := cfg.ConnConfig
	if dev := os.Getenv("DATABASE_URL"); dev != "" {
		devCfg, err := pgx.ParseConfig(dev)
		if err == nil && devCfg.Host == cc.Host && devCfg.Port == cc.Port && devCfg.Database == cc.Database {
			t.Fatalf("테스트 DB 가 개발 DB(%s)와 같습니다. 테스트는 DB 를 비우므로 다른 DB 를 지정하세요", cc.Database)
		}
	}
	ensureDatabase(ctx, t, cc)

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("테스트 DB 연결: %v", err)
	}
	t.Cleanup(pool.Close)

	// 이전 테스트가 남긴 테이블·확장·goose 버전 테이블을 모두 지운다.
	if _, err := pool.Exec(ctx, "DROP SCHEMA public CASCADE; CREATE SCHEMA public"); err != nil {
		t.Fatalf("테스트 DB 초기화: %v", err)
	}
	return pool
}

// ensureDatabase 는 cfg.Database 가 없으면 만든다.
func ensureDatabase(ctx context.Context, t testing.TB, cfg *pgx.ConnConfig) {
	t.Helper()
	admin := cfg.Copy()
	admin.Database = "postgres"
	conn, err := pgx.ConnectConfig(ctx, admin)
	if err != nil {
		t.Fatalf("postgres DB 연결: %v", err)
	}
	defer conn.Close(ctx)

	var exists bool
	if err := conn.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)", cfg.Database).Scan(&exists); err != nil {
		t.Fatalf("테스트 DB 확인: %v", err)
	}
	if exists {
		return
	}
	if _, err := conn.Exec(ctx, "CREATE DATABASE "+pgx.Identifier{cfg.Database}.Sanitize()); err != nil {
		t.Fatalf("테스트 DB 생성: %v", err)
	}
}
