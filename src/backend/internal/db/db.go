// Package db 는 PostgreSQL 연결 풀과 마이그레이션 실행을 담당한다.
package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"

	"github.com/Chocobone/PokeCardPrice/src/backend/migrations"
)

// DSNFromEnv 는 DATABASE_URL 환경 변수를 읽는다. 없으면 오류를 반환한다.
func DSNFromEnv() (string, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return "", errors.New("DATABASE_URL 환경 변수가 없습니다")
	}
	return dsn, nil
}

// Open 은 연결 풀을 만들고 연결을 확인한다.
func Open(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("연결 풀 생성: %w", err)
	}
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("DB 연결 확인: %w", err)
	}
	return pool, nil
}

// NewMigrator 는 포함된 마이그레이션 파일로 goose Provider 를 만든다.
// 세션 잠금을 사용하므로 서버와 수집기가 동시에 실행해도 안전하다.
// 반환된 *sql.DB 는 호출 측에서 닫는다(풀 자체는 닫히지 않는다).
func NewMigrator(pool *pgxpool.Pool) (*goose.Provider, *sql.DB, error) {
	locker, err := lock.NewPostgresSessionLocker()
	if err != nil {
		return nil, nil, err
	}
	sqlDB := stdlib.OpenDBFromPool(pool)
	p, err := goose.NewProvider(goose.DialectPostgres, sqlDB, migrations.FS, goose.WithSessionLocker(locker))
	if err != nil {
		_ = sqlDB.Close()
		return nil, nil, err
	}
	return p, sqlDB, nil
}

// Migrate 는 적용되지 않은 마이그레이션을 모두 적용한다.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	p, sqlDB, err := NewMigrator(pool)
	if err != nil {
		return err
	}
	defer sqlDB.Close()
	results, err := p.Up(ctx)
	for _, r := range results {
		slog.Info("migration applied", "version", r.Source.Version, "file", r.Source.Path, "duration", r.Duration)
	}
	if err != nil {
		return fmt.Errorf("마이그레이션: %w", err)
	}
	v, err := p.GetDBVersion(ctx)
	if err != nil {
		return err
	}
	slog.Info("migration done", "version", v)
	return nil
}
