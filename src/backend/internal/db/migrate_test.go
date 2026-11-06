package db

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// appTables 는 마이그레이션이 만드는 테이블이다. 추가하면 여기도 갱신한다.
var appTables = []string{
	"sets", "cards", "card_variants", "card_links", "pokemon_species",
	"price_sources", "external_ids", "price_history", "fx_rates",
	"users", "collection_items", "user_trades", "admin_price_entries",
}

func publicTables(ctx context.Context, t *testing.T, pool *pgxpool.Pool) map[string]bool {
	t.Helper()
	rows, err := pool.Query(ctx, `SELECT table_name FROM information_schema.tables
		WHERE table_schema = 'public' AND table_type = 'BASE TABLE' AND table_name <> 'goose_db_version'`)
	if err != nil {
		t.Fatalf("테이블 목록: %v", err)
	}
	names, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		t.Fatalf("테이블 목록: %v", err)
	}
	m := make(map[string]bool, len(names))
	for _, n := range names {
		m[n] = true
	}
	return m
}

// 빈 DB 에서 전체 up → 버전 0까지 down → 다시 up 이 오류 없이 끝나는지 확인한다.
func TestMigrationsUpDownUp(t *testing.T) {
	pool := testPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	p, sqlDB, err := NewMigrator(pool)
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	sources := p.ListSources()
	latest := sources[len(sources)-1].Version

	checkUp := func(step string) {
		t.Helper()
		if _, err := p.Up(ctx); err != nil {
			t.Fatalf("%s: up: %v", step, err)
		}
		v, err := p.GetDBVersion(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if v != latest {
			t.Fatalf("%s: version = %d, want %d", step, v, latest)
		}
		got := publicTables(ctx, t, pool)
		for _, name := range appTables {
			if !got[name] {
				t.Errorf("%s: 테이블 %s 없음", step, name)
			}
		}
		if len(got) != len(appTables) {
			t.Errorf("%s: 테이블 %d개, want %d개 (%v)", step, len(got), len(appTables), got)
		}
	}

	checkUp("첫 up")

	if _, err := p.DownTo(ctx, 0); err != nil {
		t.Fatalf("down to 0: %v", err)
	}
	if got := publicTables(ctx, t, pool); len(got) != 0 {
		t.Errorf("down 후 남은 테이블: %v", got)
	}
	var ext int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM pg_extension WHERE extname = 'pg_trgm'").Scan(&ext); err != nil {
		t.Fatal(err)
	}
	if ext != 0 {
		t.Errorf("down 후 pg_trgm 확장이 남아 있음")
	}

	checkUp("다시 up")
}

// fixture 는 제약 테스트에 쓰는 기본 행의 ID 이다.
type fixture struct {
	setID, cardID, variantID int64
	sourceID                 int16
}

func seed(ctx context.Context, t *testing.T, pool *pgxpool.Pool) fixture {
	t.Helper()
	var f fixture
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	must(pool.QueryRow(ctx, `INSERT INTO sets (language, tcgdex_id, name) VALUES ('ja', 'SV5K', 'ワイルドフォース') RETURNING id`).Scan(&f.setID))
	must(pool.QueryRow(ctx, `INSERT INTO cards (set_id, language, tcgdex_id, local_id, name) VALUES ($1, 'ja', 'SV5K-001', '001', 'ナゾノクサ') RETURNING id`, f.setID).Scan(&f.cardID))
	must(pool.QueryRow(ctx, `INSERT INTO card_variants (card_id, variant) VALUES ($1, 'normal') RETURNING id`, f.cardID).Scan(&f.variantID))
	must(pool.QueryRow(ctx, `SELECT id FROM price_sources WHERE code = 'tcgcsv'`).Scan(&f.sourceID))
	_, err := pool.Exec(ctx, `INSERT INTO price_history (card_variant_id, source_id, price_date, price, price_basis, market)
		VALUES ($1, $2, '2026-11-05', 1.23, 'market', 1.23)`, f.variantID, f.sourceID)
	must(err)
	_, err = pool.Exec(ctx, `INSERT INTO users (email, password_hash, display_name) VALUES ('Trainer@example.com', 'x', 'trainer')`)
	must(err)
	return f
}

// 마이그레이션 후 주요 제약이 잘못된 행을 거부하는지 확인한다.
func TestSchemaConstraints(t *testing.T) {
	pool := testPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if err := Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	f := seed(ctx, t, pool)

	const (
		uniqueViolation = "23505"
		checkViolation  = "23514"
		fkViolation     = "23503"
		// ON DELETE RESTRICT 위반은 foreign_key_violation 이 아니라 restrict_violation 이다.
		restrictViolation = "23001"
	)
	cases := []struct {
		name string
		sql  string
		args []any
		code string
	}{
		{"가격 이력 중복 (변형·소스·날짜)",
			`INSERT INTO price_history (card_variant_id, source_id, price_date, price, price_basis) VALUES ($1, $2, '2026-11-05', 2, 'mid')`,
			[]any{f.variantID, f.sourceID}, uniqueViolation},
		{"음수 가격",
			`INSERT INTO price_history (card_variant_id, source_id, price_date, price, price_basis) VALUES ($1, $2, '2026-11-06', -1, 'market')`,
			[]any{f.variantID, f.sourceID}, checkViolation},
		{"알 수 없는 대표값 필드",
			`INSERT INTO price_history (card_variant_id, source_id, price_date, price, price_basis) VALUES ($1, $2, '2026-11-06', 1, 'foo')`,
			[]any{f.variantID, f.sourceID}, checkViolation},
		{"세트와 다른 카드 언어",
			`INSERT INTO cards (set_id, language, tcgdex_id, local_id, name) VALUES ($1, 'en', 'x-1', '002', 'x')`,
			[]any{f.setID}, fkViolation},
		{"일본판이 아닌 카드의 한글판 표시",
			`INSERT INTO sets (language, tcgdex_id, name) VALUES ('en', 'sv01', 'x') ON CONFLICT DO NOTHING;
			 INSERT INTO cards (set_id, language, tcgdex_id, local_id, name, has_ko_edition)
			 SELECT id, 'en', 'sv01-001', '001', 'x', true FROM sets WHERE language = 'en' AND tcgdex_id = 'sv01'`,
			nil, checkViolation},
		{"세트 안 번호 중복",
			`INSERT INTO cards (set_id, language, tcgdex_id, local_id, name) VALUES ($1, 'ja', 'SV5K-001b', '001', 'x')`,
			[]any{f.setID}, uniqueViolation},
		{"알 수 없는 변형",
			`INSERT INTO card_variants (card_id, variant) VALUES ($1, 'shadowless')`,
			[]any{f.cardID}, checkViolation},
		{"자기 자신과 연결",
			`INSERT INTO card_links (card_id, linked_card_id, method) VALUES ($1, $1, 'manual')`,
			[]any{f.cardID}, checkViolation},
		{"외부 ID 중복 연결",
			`INSERT INTO external_ids (source_id, external_id, external_sub_id, card_variant_id) VALUES ($1, '502552', 'Normal', $2), ($1, '502552', 'Normal', $2)`,
			[]any{f.sourceID, f.variantID}, uniqueViolation},
		{"같은 통화쌍 환율",
			`INSERT INTO fx_rates (base_currency, quote_currency, rate_date, provider, rate) VALUES ('KRW', 'KRW', '2026-11-05', 'frankfurter', 1)`,
			nil, checkViolation},
		{"이메일 대소문자만 다른 중복",
			`INSERT INTO users (email, password_hash, display_name) VALUES ('trainer@EXAMPLE.com', 'x', 'y')`,
			nil, uniqueViolation},
		{"보유 수량 0",
			`INSERT INTO collection_items (user_id, card_variant_id, condition, quantity) SELECT id, $1, 'near_mint', 0 FROM users`,
			[]any{f.variantID}, checkViolation},
		{"알 수 없는 상태 등급",
			`INSERT INTO user_trades (user_id, card_variant_id, price_krw, trade_date, condition) SELECT id, $1, 1000, '2026-11-05', 'S' FROM users`,
			[]any{f.variantID}, checkViolation},
		{"사용자 거래가가 있는 변형 삭제",
			`INSERT INTO user_trades (user_id, card_variant_id, price_krw, trade_date, condition) SELECT id, $1, 1000, '2026-11-05', 'near_mint' FROM users;
			 DELETE FROM card_variants WHERE id = $1`,
			[]any{f.variantID}, restrictViolation},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tx, err := pool.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer tx.Rollback(ctx)
			err = execAll(ctx, tx, tc.sql, tc.args)
			var pgErr *pgconn.PgError
			if !errors.As(err, &pgErr) {
				t.Fatalf("err = %v, want SQLSTATE %s", err, tc.code)
			}
			if pgErr.Code != tc.code {
				t.Fatalf("SQLSTATE = %s (%s), want %s", pgErr.Code, pgErr.Message, tc.code)
			}
		})
	}

	t.Run("가격 소스 초기 행", func(t *testing.T) {
		var n int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM price_sources`).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 5 {
			t.Errorf("price_sources = %d행, want 5", n)
		}
	})

	t.Run("사용자 ID 는 uuidv7", func(t *testing.T) {
		var ver int
		if err := pool.QueryRow(ctx, `SELECT uuid_extract_version(id) FROM users LIMIT 1`).Scan(&ver); err != nil {
			t.Fatal(err)
		}
		if ver != 7 {
			t.Errorf("uuid version = %d, want 7", ver)
		}
	})

	t.Run("카드 삭제 시 변형·가격 이력 함께 삭제", func(t *testing.T) {
		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback(ctx)
		if _, err := tx.Exec(ctx, `DELETE FROM cards WHERE id = $1`, f.cardID); err != nil {
			t.Fatal(err)
		}
		var n int
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM price_history WHERE card_variant_id = $1`, f.variantID).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 0 {
			t.Errorf("price_history 남은 행 = %d, want 0", n)
		}
	})
}

// execAll 은 세미콜론으로 나뉜 문장을 차례로 실행한다. 인자는 자리표시자($n)가 있는 문장에만 넘긴다.
func execAll(ctx context.Context, tx pgx.Tx, sql string, args []any) error {
	for _, stmt := range strings.Split(sql, ";") {
		if strings.TrimSpace(stmt) == "" {
			continue
		}
		var a []any
		if strings.Contains(stmt, "$") {
			a = args
		}
		if _, err := tx.Exec(ctx, stmt, a...); err != nil {
			return err
		}
	}
	return nil
}
