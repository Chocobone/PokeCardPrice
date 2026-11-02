// Package migrations 는 goose 마이그레이션 SQL 파일을 실행 파일에 포함한다.
//
// 파일 이름은 goose 형식(`00001_설명.sql`)을 따르고, 한 파일에 `-- +goose Up`과 `-- +goose Down`을 둔다.
// sqlc 도 이 디렉터리를 스키마로 읽는다(`sqlc.yaml`).
package migrations

import "embed"

// FS 는 이 디렉터리의 마이그레이션 파일이다.
//
//go:embed *.sql
var FS embed.FS
