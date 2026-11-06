package db

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// testPool 은 TEST_DATABASE_URL 의 테스트 DB 에 연결하고, public 스키마를 비운 상태로 돌려준다.
// TEST_DATABASE_URL 이 없으면 테스트를 건너뛴다. DB 가 없으면 같은 서버의 postgres DB 로 접속해 만든다.
// 개발 DB(DATABASE_URL)와 이름이 같으면 데이터를 지우지 않도록 실패시킨다.
func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL 이 없어 DB 테스트를 건너뜁니다")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cfg, err := pgx.ParseConfig(url)
	if err != nil {
		t.Fatalf("TEST_DATABASE_URL 파싱: %v", err)
	}
	if dev := os.Getenv("DATABASE_URL"); dev != "" {
		devCfg, err := pgx.ParseConfig(dev)
		if err == nil && devCfg.Host == cfg.Host && devCfg.Port == cfg.Port && devCfg.Database == cfg.Database {
			t.Fatalf("TEST_DATABASE_URL 이 개발 DB(%s)와 같습니다. 테스트는 DB 를 비우므로 다른 DB 를 지정하세요", cfg.Database)
		}
	}
	ensureDatabase(ctx, t, cfg)

	pool, err := Open(ctx, url)
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
func ensureDatabase(ctx context.Context, t *testing.T, cfg *pgx.ConnConfig) {
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
