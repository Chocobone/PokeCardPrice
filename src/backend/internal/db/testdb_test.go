package db

import (
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Chocobone/PokeCardPrice/src/backend/internal/db/dbtest"
)

// testPool 은 이 패키지용 테스트 DB(TEST_DATABASE_URL 그대로)를 비운 상태로 돌려준다.
// 다른 패키지는 dbtest.Pool 에 자기 suffix 를 주어 별도 DB 를 쓴다.
func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	return dbtest.Pool(t, "")
}
