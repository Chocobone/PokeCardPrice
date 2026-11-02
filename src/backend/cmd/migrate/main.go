// 마이그레이션 관리 명령.
//
// 사용: go run ./cmd/migrate [status|up|down]   (기본: status)
//
// 서버와 수집기는 시작할 때 up 을 자동 실행한다. 이 명령은 상태 확인과 되돌리기(down)에 쓴다.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/Chocobone/PokeCardPrice/src/backend/internal/db"
)

func main() {
	flag.Parse()
	cmd := flag.Arg(0)
	if cmd == "" {
		cmd = "status"
	}
	if err := run(context.Background(), cmd); err != nil {
		slog.Error("migrate", "cmd", cmd, "err", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, cmd string) error {
	dsn, err := db.DSNFromEnv()
	if err != nil {
		return err
	}
	pool, err := db.Open(ctx, dsn)
	if err != nil {
		return err
	}
	defer pool.Close()

	switch cmd {
	case "up":
		return db.Migrate(ctx, pool)
	case "down", "status":
	default:
		return fmt.Errorf("알 수 없는 명령 %q (status, up, down 중 하나)", cmd)
	}

	p, sqlDB, err := db.NewMigrator(pool)
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	if cmd == "down" {
		r, err := p.Down(ctx)
		if err != nil {
			return err
		}
		fmt.Printf("rolled back: %s\n", r.Source.Path)
	}
	statuses, err := p.Status(ctx)
	if err != nil {
		return err
	}
	for _, s := range statuses {
		applied := "pending"
		if !s.AppliedAt.IsZero() {
			applied = "applied " + s.AppliedAt.Format("2006-01-02 15:04:05")
		}
		fmt.Printf("%5d  %-30s  %s\n", s.Source.Version, s.Source.Path, applied)
	}
	return nil
}
