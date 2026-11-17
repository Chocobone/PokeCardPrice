# Task #5 단계4 완료 보고서: 수집기 연결·전체 적재 실측·문서 정리

## 수행 내용

- **수집기 연결**: `cmd/collector`에 카탈로그 작업 `collect-daily`를 등록했다.
  - 매일 05:30 KST에 실행한다. 새 세트·새 카드만 카드 상세를 부른다.
  - 매월 1일은 전체 갱신(`Options.Full`)으로 모든 카드 상세를 다시 받아 희귀도·도감 번호·변형을 맞춘다 (작업지시자 결정: 작업 하나로 매일 실행).
  - `-once`·`COLLECTOR_INTERVAL` 실행은 날짜와 관계없이 `CATALOG_FULL=true`일 때만 전체 갱신한다.
  - 개발용 범위 지정: `CATALOG_LANGUAGES`(en·ja·ko 외 값은 시작 시 오류), `CATALOG_SETS`, `CATALOG_FULL`.
  - 세트 일부가 실패해도 작업은 성공으로 끝내고 `WARN`으로 실패 목록을 남긴다. 전체 실패만 작업 오류이다.
- **별도 장비 실행 (작업지시자 요청)**: `scripts/collector.sh`와 사용 설명서 `mydocs/manual/collector_remote.md`.
  - 명령: `setup`·`start`·`stop`·`restart`·`once [-d]`·`full [-d]`·`status`·`logs`.
  - `COLLECTOR_DATABASE_URL`을 지정하면 그 장비의 DB를 띄우지 않고 외부 DB에 적재한다 (`docker-compose.yml`·`.env.example`에 추가).
  - `once`·`full`은 상시 실행 수집기가 떠 있으면 거부한다 (작업 겹침 방지). `-d`는 `nohup`으로 실행하고 `logs/`에 로그를 남긴다 (`.gitignore`에 `logs/` 추가).
  - 실행 권한(`100755`)과 LF 줄바꿈으로 커밋한다. shellcheck v0.10.0 경고 없음.
- **전체 적재 실측 후 수정 (작업지시자 승인 P1~P4)**: 1차 실측에서 드러난 문제를 고쳤다.
  - P1 소수 도감 번호: TCGdex가 `dexId`를 `384.1`처럼 주는 카드가 있어(`ja` `PCG2-067` 등 4장, 레쿠쟈) 해석 오류로 세트 4개가 실패했다. 실수로 받아 정수 부분만 저장하고, 0 이하·중복은 버리며, 고친 경우 `WARN` 로그를 남긴다.
  - P2 세트 목록 실패: 세트 목록을 받지 못한 언어를 `FailedSets`에 `{lang}/*`로 남긴다. 그전에는 로그에만 남아 결과에서 `ko` 전체 누락이 보이지 않았다.
  - P3 실행 끝 재시도: 모든 세트를 돈 뒤 30초 기다렸다가 실패한 세트와 세트 목록을 받지 못한 언어를 한 번 더 시도한다. 되살린 세트 수는 `SetsRecovered`로 남긴다. 언어 재시도는 P2 제안에서 넓힌 부분이다 (아래 발견 사항).
  - P4 본문 읽기 오류: 응답 본문을 읽다 끊기거나 시간이 초과되면 전송 오류로 보고 재시도한다. JSON 형식·타입 오류는 다시 받아도 같으므로 재시도하지 않는다.
- **문서**:
  - `CLAUDE.md`: 수집기 명령, 실측 시간, 카탈로그 문서·`dbtest.Pool` 안내, 스크립트 줄.
  - `catalog_collector.md`: 4단계 구현 결과·실측.
  - `db_schema.md`: `catalog.sql` 쿼리 목록.
  - `price_data_sources.md`: 미해결 항목 결과.

## 변경 파일

| 파일 | 변경 내용 |
|------|----------|
| `src/backend/cmd/collector/main.go` | `collect-daily` 작업, 매월 1일 전체 갱신 판단 |
| `src/backend/cmd/collector/catalog.go` | 신규. 환경 변수 해석, 전체 갱신 날짜, 카탈로그 실행·결과 로그 |
| `src/backend/cmd/collector/catalog_test.go` | 신규. 환경 변수·날짜 테스트 4개 |
| `src/backend/internal/catalog/collector.go` | 실행 끝 재시도(P3), `{lang}/*` 기록(P2), 소수 도감 번호 처리(P1), `SetsRecovered` |
| `src/backend/internal/catalog/collector_test.go` | 일시 실패를 흉내 내는 테스트 서버, 테스트 3개 추가 |
| `src/backend/internal/tcgdex/types.go` | `DexID []float64`, `DexIDs()` |
| `src/backend/internal/tcgdex/client.go` | 본문 읽기 오류 재시도(P4) |
| `src/backend/internal/tcgdex/client_test.go` | 테스트 3개 추가 |
| `scripts/collector.sh` | 신규. 별도 장비 실행 스크립트 |
| `mydocs/manual/collector_remote.md` | 신규. 별도 장비 실행 설명서 |
| `Docker/docker-compose.yml`, `Docker/.env.example` | `COLLECTOR_DATABASE_URL`, `CATALOG_*` |
| `.gitignore` | `logs/` |
| `CLAUDE.md`, `mydocs/tech/catalog_collector.md`, `db_schema.md`, `price_data_sources.md` | 문서 정리 |
| `mydocs/orders/20261117.md` | 오늘 할일 갱신 |

## 실측 (개발 DB, 2026-11-17)

| 항목 | 1차: 빈 DB 첫 적재 (수정 전) | 2차: 갱신 실행 (수정 후) |
|------|---------------------------|------------------------|
| 소요 | 2시간 43분 45초 | 14분 9초 |
| 처리 세트 | 408 | 511 (건너뜀 9, 1개는 재시도로 회복) |
| 새 카드 | 35,343 | 1,964 (`en` 352, `ja` 1,373, `ko` 239) |
| 변형 upsert | 46,774 | 2,156 |
| 재시도 로그 | 85회 (대부분 `503`) | 12회 |
| 최종 실패 | `en` 4개, `ja` 8개, `ko` 전체 | 없음 |

2차 후 DB:

| 언어 | 세트 | 카드 |
|------|------|------|
| `en` | 221 | 23,912 |
| `ja` | 197 | 13,156 |
| `ko` | 91 | 239 |

- 변형 48,930, 종 1,025, `auto` 연결 239/239 (미연결 0), `has_ko_edition` 4,871.
- 2차에서 `ja` `MC`가 `503`으로 실패했다가 실행 끝 재시도(P3)에서 적재되었다. 소수 도감 번호 4장은 `384`로 저장되었다 (P1).
- 이후 매일 실행은 2차보다 짧고(세트 상세 약 510회), 매월 1일 전체 갱신은 약 3시간으로 본다.

## 발견 사항

- **언어 재시도까지 넓혔다.** P3 제안은 "실패한 세트만 한 번 더"였으나, 1차에서 DNS 장애로 `ko` 세트 목록을 받지 못해 `ko` 전체가 빠졌다. 같은 일시 오류이므로 세트 목록을 받지 못한 언어도 실행 끝에 한 번 더 시도하게 했다. 원치 않으면 언어 재시도만 빼면 된다.
- TCGdex는 몇 시간 단위로 `503`을 짧게 내는 때가 있다. 요청 단위 재시도(최대 3회, 1→2→4초)로는 부족하고, 30초 뒤 세트 단위 재시도로 대부분 풀렸다. 그래도 실패한 세트는 다음 날 실행에서 다시 시도된다.
- 세트 상세 404는 9개(`ko` 4개 포함)이며 매 실행 건너뛴다. TCGdex 데이터 문제라 손댈 것이 없다.
- 전체 갱신(약 3시간)이 매월 1일 05:30에 시작하면 08:30 무렵 끝난다. #6 시세 수집 작업과 시간이 겹치지 않게 #6에서 일정을 정한다.

## 검증

- `go build ./...`, `go vet ./...`, `gofmt`: ✅
- `go test ./...` (DB 테스트 포함): ✅
  - `cmd/collector` 4개: 환경 변수 해석·잘못된 언어 거부·`CATALOG_FULL`, 매월 1일 판단
  - `catalog` 신규 3개: 일시 실패 세트가 재시도로 적재, 세트 목록 일시 실패 회복·계속 실패 시 `ja/*`·모든 언어 실패 시 오류, 소수 도감 번호 저장값 `[384]`
  - `tcgdex` 신규 3개: 소수 도감 번호 해석, 잘린 본문 재시도(2회째 성공), JSON 타입 오류는 1회로 끝
  - 기존 테스트 전부 통과 (`TestFailedSetDoesNotBlockOthers` 등은 재시도 후에도 실패하는 경우로 그대로 통과)
- 실제 실행:
  - `scripts/collector.sh once`: 상시 수집기가 떠 있을 때 거부 (종료 코드 1) ✅
  - `scripts/collector.sh stop` → `once -d` (`CATALOG_LANGUAGES=en,ko CATALOG_SETS=sv03.5,SV5K`): 백그라운드 실행, `logs/`에 로그, `catalog: done`, 일회용 컨테이너 삭제 ✅
  - 간격 1초·1회 1.2~1.7초 실행: 앞 실행이 끝난 뒤 다음 실행이 시작되어 겹치지 않음 ✅
  - `scripts/collector.sh start` → `status`: 새 코드로 상시 실행, 다음 실행 `2026-11-18 05:30 KST`, 행 수 표시 ✅
