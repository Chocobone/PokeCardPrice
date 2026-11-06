# 기술 조사: DB 스키마 설계

> **작성일**: 2026-11-06
> **관련 타스크**: [#4](https://github.com/Chocobone/PokeCardPrice/issues/4) DB 스키마 설계 및 마이그레이션
> **상태**: 1~3단계 승인, 4단계(10절) 작성. 마이그레이션 `00002`~`00004`, 쿼리 `internal/db/queries/` 반영.

---

## 배경

#2에서 정한 데이터 소스(TCGdex, TCGCSV, Frankfurter, 국내 A·B·C안)와 #3에서 정한 도구(goose, sqlc + pgx v5, PostgreSQL 18)를 바탕으로 테이블을 설계한다.
1단계에서는 요구 사항을 한곳에 모아 엔터티로 옮기고, 상세 설계를 좌우하는 결정 사항을 정한다.

- 확인 환경: PostgreSQL 18.6 (`postgres:18-alpine3.24`), 인코딩 UTF8, `en_US.utf8`, sqlc v1.31.1. 확인일 2026-11-06
- 확인용 DB: 임시 데이터베이스 `poc_task4` (2단계 종료 후 삭제)

## 조사 내용

### 1. 요구 사항 추적표

반영 위치의 테이블 이름은 3절 엔터티 초안 기준이다.

| # | 출처 | 요구 | 반영 위치 | 비고 |
|---|------|------|-----------|------|
| R1 | #4 | 세트 | `sets` | |
| R2 | #4 | 카드 | `cards` | |
| R3 | #4 | 가격 소스 | `price_sources` | 참조 테이블 (4절 D1) |
| R4 | #4 | 날짜별 시세 이력 | `price_history` | |
| R5 | #4 | 가격 성격 구분 (실거래·호가·판매가) | `price_sources.price_kind` | 가격 성격은 소스마다 하나로 정해진다 (4절 D5) |
| R6 | #4 | 환율 | `fx_rates` | |
| R7 | #4 | 사용자 | `users` | 인증 관련 추가 컬럼·테이블은 #9 |
| R8 | #4 | 컬렉션 | `collection_items` | |
| R9 | #4 | 시계열 조회용 인덱스 | `price_history` 인덱스 | 2단계에서 대표 쿼리로 확정 |
| R10 | #2 | 카드 언어 (`en`, `ja`, `ko`) | `sets.language`, `cards.language` | 2단계: `cards`에도 두고 복합 외래 키 `(set_id, language)`로 세트와 어긋나지 않게 한다 |
| R11 | #2 | TCGdex 카드 ID | `cards.tcgdex_id` | 카탈로그 원천이라 카드에 직접 둔다 |
| R12 | #2 | TCGplayer `productId`+`subTypeName` 복합 키 | `external_ids (external_id, external_sub_id)` | 변형 단위로 연결 |
| R13 | #2 | Cardmarket 상품 ID | `external_ids` | 변형 단위로 연결 |
| R14 | #2 | 가격 이력: 카드 변형 단위 | `price_history.card_variant_id` | |
| R15 | #2 | 가격 이력: 소스, 통화, 원 가격, 가격 필드 종류 | `price_history` 컬럼, 통화는 `price_sources.currency` | 2단계: 넓은 형태로 바꿔 가격 필드 종류는 컬럼이 된다 (8절) |
| R16 | #2 | 가격 이력: 소스 갱신일, 수집일 | `price_history.price_date`, `source_updated_at`, `collected_at` | 유일 키의 날짜는 소스 기준일 (Q2 결정) |
| R17 | #2 | 가격 성격 값 4종 (`판매가`, `참고가(일본판 환산)`, `사용자 등록가`, `관리자 입력가`) | `price_sources.price_kind` CHECK | |
| R18 | #2 | 환율: 통화쌍, 값, 값 기준일, 제공처 | `fx_rates` 컬럼 | |
| R19 | #2 | 사용자 거래가: 사용자, 카드, 가격(KRW), 거래일, 상태 등급, 검증 상태 | `user_trades` | |
| R20 | #2 | 판매자 정보 컬럼 없음 | 해당 컬럼 없음 | |
| R21 | #5 | 한글판 → 일본판 연결 (자동 매칭 실패 시 수동 연결) | `card_links` | 연결 방법(`auto`·`manual`) 기록 |
| R22 | #5 | TCGdex `ko`에 없는 한글판은 "한글판 존재"만 표시 | `cards.has_ko_edition` (일본판 카드) | 5절 Q1 |
| R23 | #6 | 소스 갱신일이 오래된 가격은 현재가로 표시하지 않음 | `price_history.price_date` 비교 | 판단 기준(며칠)은 #6·#8 |
| R24 | #6 | 환율 값이 없으면 직전 값 사용 | `fx_rates (base, quote, rate_date)` 인덱스 | 조회 쿼리에서 처리 |
| R25 | #7 | 일본판 참고가를 원화로 환산 | 조회 시 `price_history`(일본판 변형) × `fx_rates`로 계산 | Q3 결정: 저장하지 않음 (2026-11-06) |
| R26 | #7 | 관리자 입력가: 가격·출처·확인일 | `admin_price_entries` | 원본 입력 테이블 (4절 D5) |
| R27 | #7 | 사용자 거래가 집계 결과 | `price_history` (소스 `user_aggregate`) | 원본은 `user_trades` |
| R28 | #8 | 이름·세트·번호 검색 | `cards.name` trigram 인덱스, `cards (set_id, local_id)` | 4절 D6 |
| R29 | #8 | 7일·30일·90일·1년 추이, 변동률, 급등락 순위 | `price_history` 인덱스 | 2단계 |
| R30 | #10 | 보유 카드·수량 | `collection_items` | |
| R31 | #10 | 총 평가액, 기간별 증감 | `collection_items` × `price_history` 계산 | 현재 보유 수량 기준. 보유 이력은 저장하지 않음 (5절 Q4) |
| R32 | #9 | 비밀번호 해시 저장 | `users.password_hash` | 해시 방식은 #9 |
| R33 | #7 | 관리자만 관리자 입력가 등록 | `users.role` | 권한 검사는 #9 |

**반영하지 않는 요구**

| 요구 | 사유 | 담당 |
|------|------|------|
| 토큰·세션 저장 | 인증 방식 미정 | #9 (필요 시 마이그레이션 추가) |
| 조회 결과 캐시, 순위 집계 테이블 | 성능 측정 후 판단 | #11 |

### 2. 엔터티 초안 (ERD)

```mermaid
erDiagram
    sets ||--o{ cards : "수록"
    cards ||--o{ card_variants : "변형"
    cards ||--o| card_links : "한글판→일본판"
    price_sources ||--o{ external_ids : ""
    card_variants ||--o{ external_ids : "외부 상품 연결"
    card_variants ||--o{ price_history : ""
    price_sources ||--o{ price_history : ""
    users ||--o{ collection_items : "보유"
    card_variants ||--o{ collection_items : ""
    users ||--o{ user_trades : "거래가 등록"
    card_variants ||--o{ user_trades : ""
    users ||--o{ admin_price_entries : "관리자 입력"
    card_variants ||--o{ admin_price_entries : ""

    sets {
        bigint id PK
        text language "en, ja, ko"
        text tcgdex_id "언어 안에서 유일"
        text code "세트 코드 (SV5K 등)"
        text name
        date release_date
    }
    cards {
        bigint id PK
        bigint set_id FK
        text tcgdex_id UK
        text local_id "수록 번호"
        text name
        text rarity
        text image_url
        bool has_ko_edition "일본판 카드에만 의미"
    }
    card_variants {
        bigint id PK
        bigint card_id FK
        text variant "normal, holofoil, reverse_holofoil 등"
    }
    card_links {
        bigint card_id PK "한글판 카드"
        bigint linked_card_id FK "일본판 카드"
        text method "auto, manual"
    }
    price_sources {
        bigint id PK
        text code UK "tcgcsv, tcgdex_cardmarket 등"
        text price_kind "sale, jp_reference, user, admin"
        text name
    }
    external_ids {
        bigint source_id FK
        text external_id
        text external_sub_id "TCGplayer subTypeName"
        bigint card_variant_id FK
    }
    price_history {
        bigint card_variant_id FK
        bigint source_id FK
        text price_field "market, avg, trend 등"
        date price_date
        text currency "USD, EUR, KRW"
        numeric price
        timestamptz source_updated_at
        timestamptz collected_at
    }
    fx_rates {
        text base_currency
        text quote_currency
        date rate_date
        text provider
        numeric rate
        timestamptz fetched_at
    }
    users {
        uuid id PK
        text email UK
        text password_hash
        text display_name
        text role "user, admin"
    }
    collection_items {
        bigint id PK
        uuid user_id FK
        bigint card_variant_id FK
        text condition
        int quantity
    }
    user_trades {
        bigint id PK
        uuid user_id FK
        bigint card_variant_id FK
        numeric price_krw
        date trade_date
        text condition
        text verification_status
    }
    admin_price_entries {
        bigint id PK
        bigint card_variant_id FK
        numeric price_krw
        text source_note "출처"
        date checked_date "확인일"
        uuid created_by FK
    }
```

- 컬럼은 엔터티 구분에 필요한 것만 적었다. 생성·수정 시각, NULL 허용, 기본값, 제약은 2단계 상세 정의에서 정한다.
- `price_history`, `fx_rates`의 기본 키(대리 키를 둘지, 유일 키를 기본 키로 쓸지)는 2단계에서 정한다.

### 3. 엔터티별 설명

| 엔터티 | 역할 | 주요 설계 판단 |
|--------|------|----------------|
| `sets` | 언어별 세트 | TCGdex 세트 ID는 언어마다 체계가 달라 `(language, tcgdex_id)`로 유일하게 한다. 한글판·일본판 연결에 쓸 세트 코드를 따로 둔다 |
| `cards` | 카드 | 수록 번호는 `SV5K-001`, `TG01`처럼 숫자가 아닐 수 있어 `text`로 둔다 |
| `card_variants` | 가격의 기준 단위 | TCGdex `variants_detailed`는 문서에 없는 필드라, 변형을 외부 필드에 묶지 않고 자체 코드로 둔다. 외부 상품과는 `external_ids`로 연결한다 |
| `card_links` | 한글판 → 일본판 연결 | 한글판 카드 1장당 연결 1개. 자동 매칭이 틀리면 수동으로 바꿀 수 있도록 방법을 기록한다. 영어판과는 연결하지 않는다(#2 결론) |
| `price_sources` | 가격 소스와 가격 성격 | 이름·가격 성격·출처 표기 문구처럼 속성이 있어 참조 테이블로 둔다 |
| `external_ids` | 소스별 외부 상품 ID | TCGplayer는 `productId` + `subTypeName`, Cardmarket은 상품 ID. 같은 외부 상품이 두 변형에 연결되지 않게 `(source_id, external_id, external_sub_id)`를 유일하게 한다 |
| `price_history` | 날짜별 가격 | 원래 통화와 원래 값 그대로 저장한다. 원화 환산은 `fx_rates`로 조회할 때 계산한다 |
| `fx_rates` | 환율 | 값 기준일과 제공처를 함께 저장한다. 1차 Frankfurter, 대체 한국수출입은행 |
| `users` | 사용자 | 최소 컬럼만 둔다. `role`은 관리자 입력가 권한 구분용이다 |
| `collection_items` | 보유 카드 | 같은 카드 변형·상태 등급은 한 행에 수량으로 관리한다 |
| `user_trades` | 사용자 거래가 원본 | 사용자 소유 데이터이고 검증 상태가 바뀌므로 `price_history`와 분리한다. #7이 집계해 `price_history`에 넣는다 |
| `admin_price_entries` | 관리자 입력가 원본 | 출처·확인일·입력자를 남긴다. 시세로 쓰는 값은 `price_history`에도 넣어, 조회 쿼리가 소스 종류와 상관없이 같은 테이블을 읽게 한다 |

### 4. 결정 사항 비교

#### D1. 열거 값 표현

`poc_task4`에서 세 방식에 값 1개를 추가하고 되돌리는 작업을 해 보았다. 샘플 테이블은 20만 행이다.

| 확인 항목 | PostgreSQL `enum` | `text` + `CHECK` | 참조 테이블 (FK) |
|-----------|-------------------|------------------|------------------|
| 값 추가 | `ALTER TYPE ... ADD VALUE`. 트랜잭션 안에서 실행은 되지만 **커밋 전에는 새 값을 쓸 수 없다** (`unsafe use of new value`). goose는 기본으로 마이그레이션마다 트랜잭션을 쓰므로, 값 추가와 사용을 다른 마이그레이션으로 나눠야 한다 | 제약 삭제 후 다시 추가. 20만 행에서 10.7 ms(`NOT VALID` + `VALIDATE` 방식 11.9 ms) | `INSERT` 1행 |
| 값 삭제 (`Down`) | **불가** (`dropping an enum value is not implemented`). 타입을 다시 만들고 컬럼을 바꿔야 한다 | 제약을 이전 목록으로 되돌리면 된다 | `DELETE`. 쓰이는 값이면 FK가 막아 준다 |
| 잘못된 값 거부 | 됨 | 됨 | 됨 |
| 정렬 | 정의 순서로 정렬된다 | 문자열 순 | 문자열 순 (정렬 컬럼을 따로 둘 수 있음) |
| 한글 표시 이름 등 속성 | 둘 수 없음 | 둘 수 없음 | 컬럼으로 둘 수 있음 |
| sqlc 생성 타입 | `type PriceKind string`과 값 상수, `Scan`·`Valid`·`AllPriceKindValues`, `NullPriceKind`까지 생성 | `string` | `string` |

- sqlc는 참조 테이블 이름이 enum 이름과 겹치면(`price_kinds` 테이블과 `price_kind` 타입) `struct name conflicts with enum name` 오류로 생성을 멈췄다. 두 방식을 섞을 때는 이름을 다르게 지어야 한다.

**선정안**

- **값 목록이 짧고 거의 바뀌지 않는 값**(언어, 통화, 가격 성격, 상태 등급, 검증 상태, 사용자 역할, 연결 방법)은 **`text` + `CHECK`**로 한다. 되돌리기가 쉽고, goose 트랜잭션 제약이 없다.
- **속성이 있는 값**(가격 소스)은 **참조 테이블**로 한다.
- `enum`은 쓰지 않는다. 생성되는 Go 상수는 편하지만, 값을 되돌릴 수 없어 "모든 마이그레이션에 `Down`" 원칙과 맞지 않는다. 필요한 Go 상수는 4단계에서 직접 정의한다(값 몇 개 수준).

#### D2. 기본 키

| 대상 | 선정안 | 근거 |
|------|--------|------|
| 카탈로그·가격 (`sets`, `cards`, `card_variants`, `price_sources`, `price_history` 등) | `bigint generated always as identity` | 내부 조인이 많고 행 수가 가장 많은 `price_history`가 참조한다. 8바이트로 `uuid`(16바이트)보다 인덱스가 작다. 카드 ID가 URL에 노출되어도 문제가 없다 (공개 카탈로그) |
| `users` | `uuid`, 기본값 `uuidv7()` | 순번이 노출되면 가입자 수를 추정할 수 있다. PostgreSQL 18의 `uuidv7()` 동작을 확인했다 (`01a1afb2-9943-77c8-...`). 시간순이라 B-tree 삽입이 무작위 `uuid`보다 유리하다 |
| `collection_items`, `user_trades`, `admin_price_entries` | `bigint identity` | API에서 수정·삭제할 때 항상 사용자 소유 여부를 함께 조건으로 건다(#10) |

- sqlc는 `uuid`를 `pgtype.UUID`로 생성한다. `google/uuid`(이미 간접 의존성)로 바꿀지는 금액 타입과 함께 4단계 `overrides`에서 정한다.

#### D3. 가격 이력 형태

| 형태 | 내용 | 장점 | 단점 |
|------|------|------|------|
| **긴 형태** | `(변형, 소스, 가격 필드, 날짜)`마다 1행 | 소스마다 필드가 달라도(TCGCSV `low`·`mid`·`high`·`market`·`directLow`, Cardmarket `avg`·`low`·`trend`·`avg1`·`avg7`·`avg30`) 테이블이 같다. 필드를 추가해도 마이그레이션이 없다. #2 설계 입력(가격 필드 종류 컬럼)과 맞는다 | 필드 수만큼 행이 늘어난다. 행마다 헤더(약 24바이트)와 키 컬럼이 반복된다 |
| **넓은 형태** | `(변형, 소스, 날짜)`마다 1행, 필드별 NULL 허용 컬럼 | 행 수가 필드 수 분의 1. 빈 컬럼은 NULL 비트맵이라 거의 공간을 쓰지 않는다 | 소스 필드가 바뀌면 마이그레이션이 필요하다. 필드별 추이 조회 쿼리가 컬럼 이름에 묶인다 |

**잠정 선정안: 긴 형태.** 단, 행 수가 문제가 되면 **저장할 필드를 소스별 대표 필드로 줄이는 것**을 먼저 검토한다. 2단계에서 두 형태의 1년치 크기를 `poc_task4`에서 실제로 재서 확정한다.

> **2단계 측정 결과로 넓은 형태로 바꾼다.** 근거는 8절이다.

#### D4. 금액 Go 타입 (sqlc `overrides`)

sqlc 기본 생성 결과: `numeric` → `pgtype.Numeric`, `date` → `pgtype.Date`, `timestamptz` → `pgtype.Timestamptz` (NULL 허용 여부와 상관없이 같은 타입).

| 후보 | 장점 | 단점 |
|------|------|------|
| `pgtype.Numeric` (기본) | 추가 의존성 없음, 정밀도 보존 | 산술 연산이 없다. JSON 직렬화와 비교·계산마다 변환이 필요하다 |
| `string` | 정밀도 보존, JSON에 그대로 쓸 수 있다 | 계산하려면 파싱해야 한다. 타입으로 금액임이 드러나지 않는다 |
| decimal 라이브러리 (`shopspring/decimal` + pgx 어댑터 등) | 정밀 산술(환산, 평균, 변동률), JSON 지원 | 의존성 추가 |

판단 기준: Go에서 하는 계산의 양(환산·집계를 SQL에서 할지 Go에서 할지), JSON 응답 형태, 의존성 수. **4단계에서 실제 생성 코드로 비교해 확정한다.**

DB 쪽 정밀도는 2단계에서 정한다. 현재 후보는 가격 `numeric(14,2)`(USD·EUR 센트 단위, KRW 정수), 환율 `numeric(18,8)`이다.

#### D5. 가격 성격·사용자 거래가·관리자 입력가의 위치

| 결정 | 선정안 | 근거 |
|------|--------|------|
| 가격 성격 | `price_sources.price_kind`에 둔다 | 확정 소스마다 가격 성격이 하나다(TCGCSV·Cardmarket → 판매가, 일본판 환산 → 참고가, 사용자 집계 → 사용자 등록가, 관리자 입력 → 관리자 입력가). 가격 행마다 반복하면 소스와 성격이 어긋나는 행이 생길 수 있다 |
| 사용자 거래가 | 원본은 `user_trades`, 집계 결과만 `price_history` | 원본은 사용자 소유이고 수정·삭제·검증 상태 변경이 있다. 시세 조회는 `price_history` 한 곳에서 한다 |
| 관리자 입력가 | 원본은 `admin_price_entries`, 시세 값은 `price_history` | 출처·확인일은 관리자 입력에만 있는 정보라, `price_history`에 NULL 컬럼을 늘리지 않는다. 사용자 거래가와 같은 구조(원본 → 시세)로 맞춘다 |

#### D6. 카드 이름 검색

`poc_task4`에서 `pg_trgm` GIN 인덱스로 한글·영어·일본어 이름을 검색했다.

- 확장 설치 가능 (`postgres:18-alpine3.24`에 포함).
- `ILIKE '%상해%'`가 인덱스(`Bitmap Index Scan`)를 사용했고 `이상해씨`, `이상해풀`을 찾았다. `'%리자%'` → `리자몽 ex`, `'%char%'` → `Charizard`.
- 한글 두 글자 검색어는 trigram을 만들 수 없어 인덱스가 걸러 주지 못하고, 인덱스 전체를 읽는다. 표본이 5행이라 성능은 판단하지 못했다. **2단계에서 카드 수 규모로 확인한다.**

**선정안**: `pg_trgm` GIN 인덱스. 확장 생성은 `00002_catalog.sql`에 넣는다.

### 5. 작업지시자 확인 사항

**Q1~Q4 모두 제안대로 결정 (작업지시자, 2026-11-06).**

| # | 사항 | 제안 | 영향 |
|---|------|------|------|
| Q1 | TCGdex `ko`에 없는 한글판 카드(전체의 약 97.5%)를 카드 행으로 만들지 | **만들지 않는다.** 일본판 카드에 `has_ko_edition`만 표시한다(#2 설계 입력과 같음). `ko`에 있는 239장만 한글판 카드 행으로 만들고 `card_links`로 일본판에 연결한다 | #5 범위, #13 화면 (한글판 카드는 대부분 일본판 카드 페이지로 보이게 됨) |
| Q2 | `price_history` 유일 키의 날짜를 **소스 기준일**로 할지 **수집일**로 할지 | **소스 기준일.** 소스가 갱신되지 않은 날 수집해도 같은 행이 갱신될 뿐 새 행이 생기지 않고, 오래된 가격을 날짜로 바로 알 수 있다. 수집 시각은 `collected_at`에 따로 남긴다 | #6 적재 로직, #8 추이 차트 (소스가 갱신되지 않은 날은 빈 날짜가 된다) |
| Q3 | 일본판 참고가(#7 A안)를 KRW로 **저장**할지, **조회할 때 계산**할지 | **조회할 때 계산한다.** 일본판 가격(USD·EUR)은 이미 `price_history`에 있고, 환산은 `fx_rates`로 언제든 다시 할 수 있다. 저장하면 같은 정보를 두 번 갖게 되고, 환율을 고치면 다시 계산해야 한다. API는 계산 결과를 가격 성격 `참고가(일본판 환산)`로 표시한다 | **#2에서 정한 "환산해 저장" 방침을 바꾸는 것**. 승인되면 #7 A안은 저장 작업 대신 조회 쿼리가 되고, `price_sources`에 `jp_reference` 행은 표시용으로만 남긴다 |
| Q4 | 컬렉션 기간별 증감(#10)을 **현재 보유 수량 기준**으로 계산할지 | **현재 보유 수량 기준**으로 계산한다(현재 수량 × 날짜별 가격). 보유 이력 테이블은 두지 않는다 | 카드를 언제 샀는지를 반영한 손익은 계산할 수 없다. 필요하면 #10에서 `acquired_at`·매입가 컬럼을 추가한다 |

### 6. 상세 정의 (2단계)

아래 DDL이 3단계 마이그레이션의 기준이다. `poc_task4`에서 이 DDL로 12개 테이블을 모두 만들고 실제 카탈로그(TCGdex 세트 335개, 카드 36,756장)를 넣어 확인했다.
구현 계획서는 테이블마다 컬럼 표를 쓰기로 했으나, 컬럼·제약을 표와 DDL에 이중으로 적으면 어긋나기 쉬워 DDL을 정의로 삼고 판단 근거만 표로 정리한다.

```sql
CREATE EXTENSION IF NOT EXISTS pg_trgm WITH SCHEMA public;

CREATE TABLE sets (
  id           bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  language     text NOT NULL CONSTRAINT sets_language_check CHECK (language IN ('en','ja','ko')),
  tcgdex_id    text NOT NULL,
  name         text NOT NULL,
  release_date date,
  card_count   integer,
  logo_url     text,
  created_at   timestamptz NOT NULL DEFAULT now(),
  updated_at   timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT sets_language_tcgdex_id_key UNIQUE (language, tcgdex_id),
  CONSTRAINT sets_id_language_key UNIQUE (id, language)
);

CREATE TABLE cards (
  id             bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  set_id         bigint NOT NULL,
  language       text NOT NULL,
  tcgdex_id      text NOT NULL,
  local_id       text NOT NULL,
  name           text NOT NULL,
  rarity         text,
  image_url      text,
  dex_ids        integer[],
  has_ko_edition boolean NOT NULL DEFAULT false,
  created_at     timestamptz NOT NULL DEFAULT now(),
  updated_at     timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT cards_set_id_language_fkey FOREIGN KEY (set_id, language) REFERENCES sets (id, language) ON DELETE RESTRICT,
  CONSTRAINT cards_language_tcgdex_id_key UNIQUE (language, tcgdex_id),
  CONSTRAINT cards_set_id_local_id_key UNIQUE (set_id, local_id),
  CONSTRAINT cards_has_ko_edition_check CHECK (NOT has_ko_edition OR language = 'ja')
);
CREATE INDEX cards_name_trgm_idx ON cards USING gin (name gin_trgm_ops);

CREATE TABLE card_variants (
  id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  card_id    bigint NOT NULL REFERENCES cards (id) ON DELETE CASCADE,
  variant    text NOT NULL CONSTRAINT card_variants_variant_check CHECK (variant IN
               ('normal','holofoil','reverse_holofoil','first_edition','first_edition_holofoil','unlimited','unlimited_holofoil')),
  created_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT card_variants_card_id_variant_key UNIQUE (card_id, variant)
);

CREATE TABLE card_links (
  card_id        bigint PRIMARY KEY REFERENCES cards (id) ON DELETE CASCADE,
  linked_card_id bigint NOT NULL REFERENCES cards (id) ON DELETE CASCADE,
  method         text NOT NULL CONSTRAINT card_links_method_check CHECK (method IN ('auto','manual')),
  created_at     timestamptz NOT NULL DEFAULT now(),
  updated_at     timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT card_links_not_self_check CHECK (card_id <> linked_card_id)
);
CREATE INDEX card_links_linked_card_id_idx ON card_links (linked_card_id);

CREATE TABLE price_sources (
  id          smallint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  code        text NOT NULL CONSTRAINT price_sources_code_key UNIQUE,
  name        text NOT NULL,
  price_kind  text NOT NULL CONSTRAINT price_sources_price_kind_check CHECK (price_kind IN ('sale','jp_reference','user','admin')),
  currency    text NOT NULL CONSTRAINT price_sources_currency_check CHECK (currency IN ('USD','EUR','KRW')),
  attribution text
);
INSERT INTO price_sources (code, name, price_kind, currency, attribution) VALUES
  ('tcgcsv',            'TCGplayer (TCGCSV)',  'sale',         'USD', 'TCGplayer 가격 (TCGCSV 경유)'),
  ('tcgdex_cardmarket', 'Cardmarket (TCGdex)', 'sale',         'EUR', 'Cardmarket 가격 (TCGdex 경유)'),
  ('jp_reference',      '일본판 환산 참고가',   'jp_reference', 'KRW', '일본판 가격 원화 환산 (Frankfurter 환율)'),
  ('user_aggregate',    '사용자 등록가 집계',   'user',         'KRW', NULL),
  ('admin',             '관리자 입력가',        'admin',        'KRW', NULL);

CREATE TABLE external_ids (
  source_id       smallint NOT NULL REFERENCES price_sources (id),
  external_id     text NOT NULL,
  external_sub_id text NOT NULL DEFAULT '',
  card_variant_id bigint NOT NULL REFERENCES card_variants (id) ON DELETE CASCADE,
  created_at      timestamptz NOT NULL DEFAULT now(),
  updated_at      timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (source_id, external_id, external_sub_id)
);
CREATE INDEX external_ids_card_variant_id_idx ON external_ids (card_variant_id);

CREATE TABLE price_history (
  card_variant_id   bigint NOT NULL REFERENCES card_variants (id) ON DELETE CASCADE,
  price_date        date NOT NULL,
  source_id         smallint NOT NULL REFERENCES price_sources (id),
  sample_count      integer,
  collected_at      timestamptz NOT NULL DEFAULT now(),
  source_updated_at timestamptz,
  price             numeric(12,2) NOT NULL,
  price_basis       text NOT NULL,
  low numeric(12,2), mid numeric(12,2), high numeric(12,2), market numeric(12,2), direct_low numeric(12,2),
  avg numeric(12,2), trend numeric(12,2), avg1 numeric(12,2), avg7 numeric(12,2), avg30 numeric(12,2),
  PRIMARY KEY (card_variant_id, source_id, price_date),
  CONSTRAINT price_history_price_basis_check CHECK (price_basis IN ('market','mid','low','trend','avg','median','manual')),
  CONSTRAINT price_history_price_check CHECK (price >= 0),
  CONSTRAINT price_history_sample_count_check CHECK (sample_count IS NULL OR sample_count > 0)
);
CREATE INDEX price_history_source_id_price_date_idx ON price_history (source_id, price_date) INCLUDE (card_variant_id, price);

CREATE TABLE fx_rates (
  base_currency  text NOT NULL CONSTRAINT fx_rates_base_currency_check CHECK (base_currency IN ('USD','EUR','KRW')),
  quote_currency text NOT NULL CONSTRAINT fx_rates_quote_currency_check CHECK (quote_currency IN ('USD','EUR','KRW')),
  rate_date      date NOT NULL,
  provider       text NOT NULL CONSTRAINT fx_rates_provider_check CHECK (provider IN ('frankfurter','koreaexim')),
  rate           numeric(18,8) NOT NULL CONSTRAINT fx_rates_rate_check CHECK (rate > 0),
  fetched_at     timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (base_currency, quote_currency, rate_date, provider),
  CONSTRAINT fx_rates_pair_check CHECK (base_currency <> quote_currency)
);

CREATE TABLE users (
  id            uuid PRIMARY KEY DEFAULT uuidv7(),
  email         text NOT NULL,
  password_hash text NOT NULL,
  display_name  text NOT NULL,
  role          text NOT NULL DEFAULT 'user' CONSTRAINT users_role_check CHECK (role IN ('user','admin')),
  created_at    timestamptz NOT NULL DEFAULT now(),
  updated_at    timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX users_email_key ON users (lower(email));

CREATE TABLE collection_items (
  id              bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  user_id         uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  card_variant_id bigint NOT NULL REFERENCES card_variants (id) ON DELETE RESTRICT,
  condition       text NOT NULL CONSTRAINT collection_items_condition_check CHECK (condition IN ('near_mint','lightly_played','moderately_played','heavily_played','damaged')),
  quantity        integer NOT NULL CONSTRAINT collection_items_quantity_check CHECK (quantity > 0),
  created_at      timestamptz NOT NULL DEFAULT now(),
  updated_at      timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT collection_items_user_variant_condition_key UNIQUE (user_id, card_variant_id, condition)
);

CREATE TABLE user_trades (
  id                  bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  user_id             uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  card_variant_id     bigint NOT NULL REFERENCES card_variants (id) ON DELETE RESTRICT,
  price_krw           numeric(12,0) NOT NULL CONSTRAINT user_trades_price_krw_check CHECK (price_krw > 0),
  trade_date          date NOT NULL,
  condition           text NOT NULL CONSTRAINT user_trades_condition_check CHECK (condition IN ('near_mint','lightly_played','moderately_played','heavily_played','damaged')),
  verification_status text NOT NULL DEFAULT 'pending' CONSTRAINT user_trades_verification_status_check CHECK (verification_status IN ('pending','accepted','outlier','rejected')),
  created_at          timestamptz NOT NULL DEFAULT now(),
  updated_at          timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX user_trades_card_variant_id_trade_date_idx ON user_trades (card_variant_id, trade_date);
CREATE INDEX user_trades_user_id_idx ON user_trades (user_id);

CREATE TABLE admin_price_entries (
  id              bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  card_variant_id bigint NOT NULL REFERENCES card_variants (id) ON DELETE RESTRICT,
  price_krw       numeric(12,0) NOT NULL CONSTRAINT admin_price_entries_price_krw_check CHECK (price_krw > 0),
  source_note     text NOT NULL,
  source_url      text,
  checked_date    date NOT NULL,
  created_by      uuid REFERENCES users (id) ON DELETE SET NULL,
  created_at      timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX admin_price_entries_card_variant_id_checked_date_idx ON admin_price_entries (card_variant_id, checked_date);

-- Q5 결정으로 추가 (2026-11-06): 한글 종 이름으로 카드를 찾기 위한 테이블. 데이터 적재는 #5
CREATE TABLE pokemon_species (
  dex_id     integer PRIMARY KEY CONSTRAINT pokemon_species_dex_id_check CHECK (dex_id > 0),
  name_en    text NOT NULL,
  name_ja    text,
  name_ko    text,
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX pokemon_species_name_ko_trgm_idx ON pokemon_species USING gin (name_ko gin_trgm_ops);
CREATE INDEX cards_dex_ids_idx ON cards USING gin (dex_ids);
```

**1단계 초안에서 바뀐 점**

| 대상 | 변경 | 근거 |
|------|------|------|
| `cards.language` | 추가. `(set_id, language)` 복합 외래 키로 `sets`를 참조 | 검색 필터에 조인이 필요 없고, 세트와 언어가 어긋날 수 없다. 일본판·한글판은 TCGdex 카드 ID가 같아(`SV5K-001`) `(language, tcgdex_id)`로 유일해야 한다 |
| `cards.dex_ids` | 추가 (`integer[]`, NULL 허용) | TCGdex `dexId`. 한글 이름 검색 보완(9절 Q5)에 쓴다 |
| `price_sources.currency` | 통화를 가격 행에서 소스로 옮김 | 소스마다 통화가 하나다. 가격 행에 두면 소스와 통화가 어긋난 행이 생길 수 있다 |
| `price_sources.id` | `smallint` | 행이 5개뿐이고, 1년에 5천만 행인 `price_history`가 참조해 행마다 6바이트를 줄인다 |
| `price_history` | 넓은 형태 + 대표값(`price`, `price_basis`) | 8절 |
| `price_history` 기본 키 | 대리 키 없이 `(card_variant_id, source_id, price_date)` | 이 키로만 조회·갱신한다. 대리 키를 두면 인덱스가 하나 더 생긴다 |
| `fx_rates` 기본 키 | `(base_currency, quote_currency, rate_date, provider)` | 직전 값 조회가 이 인덱스의 역순 스캔으로 끝난다 (7절 S5) |
| `sets.code` | 삭제 | 일본판·한글판은 TCGdex 세트 ID가 같아(`SV5K`) 별도 코드가 필요 없다 |
| `user_trades.price_krw`, `admin_price_entries.price_krw` | `numeric(12,0)` | 원화는 소수점이 없다 |

**제약과 삭제 동작**

| 관계 | 삭제 동작 | 근거 |
|------|-----------|------|
| 세트 → 카드 | `RESTRICT` | 카탈로그 삭제는 드물고, 실수로 지우면 가격 이력까지 사라진다 |
| 카드 → 변형, 변형 → 가격·외부 ID, 카드 → 카드 연결 | `CASCADE` | 수집 데이터는 카드에 딸린다 |
| 변형 → 컬렉션·사용자 거래가·관리자 입력가 | `RESTRICT` | 사용자가 입력한 데이터는 카탈로그 정리로 사라지면 안 된다 |
| 사용자 → 컬렉션·사용자 거래가 | `CASCADE` | 탈퇴 시 개인 데이터 삭제 |
| 사용자 → 관리자 입력가 (`created_by`) | `SET NULL` | 입력한 관리자가 없어져도 시세 근거는 남긴다 |

**대표값 규칙 (`price`, `price_basis`)**

넓은 형태에서 소스마다 필드가 다르므로, 화면·순위·평가액이 공통으로 쓸 값을 `price`에 두고 어느 필드에서 왔는지 `price_basis`에 남긴다. 상세 필드는 원래 값 그대로 함께 저장한다.

| 소스 | `price` | 값이 없을 때 |
|------|---------|--------------|
| `tcgcsv` | `marketPrice` (`market`) | `midPrice` (`mid`). TCGCSV `marketPrice` NULL 비율은 `Pokemon` 1.8%, `Pokemon Japan` 21.6% (2026-11-06 전체 조회) |
| `tcgdex_cardmarket` | `trend` | `avg` |
| `user_aggregate` | 중앙값 (`median`), 건수는 `sample_count` | 행을 만들지 않음 (#7 최소 건수) |
| `admin` | 입력값 (`manual`) | - |

- `jp_reference`는 Q3 결정에 따라 `price_history`에 행을 만들지 않는다. 표시 이름·출처 문구용으로 `price_sources`에만 둔다.
- 대체 필드의 순서는 #6에서 바꿀 수 있다. 바꾸어도 `price_basis`가 남아 있어 과거 행을 구분할 수 있다.

### 7. 인덱스와 대표 쿼리

`poc_task4`에서 넓은 형태 30일치(2,886,870행), 사용자 1명·보유 200개, 환율 평일 22일치로 측정했다. 시간은 `EXPLAIN ANALYZE`의 실행 시간이다.

| # | 쿼리 | 사용 인덱스 | 실행 시간 |
|---|------|-------------|-----------|
| S1 | 변형 1개·소스 1개의 1년 추이 | `price_history_pkey` 범위 스캔 | 0.30 ms (30행) |
| S2 | 카드 1장의 변형 × 소스별 최신가 (`DISTINCT ON`) | `card_variants_card_id_variant_key` → `price_history_pkey` | 1.07 ms |
| S3 | 급등 순위: TCGCSV 최신일 대비 7일 전 변동률 상위 20 | `price_history_source_id_price_date_idx` (Index Only Scan, Heap Fetches 0) | **24.5 ms** |
| S3' | 같은 쿼리, 보조 인덱스 없이 | 순차 스캔 | **174.2 ms** (30일치. 1년이면 행 수에 비례해 늘어남) |
| S4 | 컬렉션 평가액: 보유 200개 × 최신가 × 최신 환율 | `price_history_pkey` 역순 스캔, `fx_rates_pkey` 역순 스캔 | 3.71 ms |
| S5 | 주말(11/01) 환율 → 직전 영업일(10/30) 값 | `fx_rates_pkey` | 직전 값 반환 확인 |
| S6 | 세트 + 번호 검색 (`ja`, `SV5K`, `001`) | `sets_language_tcgdex_id_key` → `cards_set_id_local_id_key` | 0.07 ms |
| S7 | 이름 검색 `pikachu` (7자) | `cards_name_trgm_idx` | 0.45 ms (243행) |
| S8 | 이름 검색 `ピカ` (2자) | 순차 스캔 (trigram을 만들 수 없음) | 10.3 ms (74행) |

- 보조 인덱스 `(source_id, price_date) INCLUDE (card_variant_id, price)`는 순위·변동률 쿼리를 위한 것이다. 기본 키만으로는 날짜로 찾을 수 없어 순차 스캔이 된다. 대신 1년에 약 1.9 GB를 더 쓴다(8절).
- 두 글자 검색어는 36,756장 전체를 순차 스캔해도 약 10 ms라 문제가 없다.
- `collection_items`는 `(user_id, card_variant_id, condition)` 유일 제약의 인덱스가 사용자별 조회를 맡는다. 측정에서는 행이 200개라 순차 스캔을 골랐다.

### 8. 시세 이력 규모와 형태 확정

**실제 데이터 양 (2026-11-06)**

| 소스 | 확인 방법 | 하루 변형 행 | 값이 있는 가격 필드 |
|------|-----------|--------------|---------------------|
| TCGCSV `Pokemon` | 그룹 221개 전체 `prices` 조회 | 45,715 (상품 31,664) | 137,145 (`low`·`mid`·`high`) + `market` 44,875 + `directLow` 17,881 = 199,901 |
| TCGCSV `Pokemon Japan` | 그룹 461개 전체 조회 | 29,345 (상품 26,430) | 88,035 + `market` 23,019 = 111,054 |
| TCGdex Cardmarket `en` | 카드 23,736장 중 표본 40장 | 약 42,700 (가격 있는 카드 90% × 정상·holo 2행) | 카드당 10.4 → 약 222,000 |
| TCGdex Cardmarket `ja` | 카드 12,781장 중 표본 40장 | 약 20,400 (가격 있는 카드 80% × 2행) | 카드당 7.8 → 약 80,000 |
| **합계** | | **약 138,000** | **약 613,000** |

- TCGCSV 수치는 카드가 아닌 상품(박스·팩 등)을 포함한 상한이다. 수집기는 카드 변형과 연결된 상품만 저장하므로 실제로는 더 적다.
- TCGCSV 조회는 682회, 요청 간 120 ms, 전용 User-Agent로 했다. TCGdex는 카드 80장을 요청 간 500 ms로 조회했다.

**행당 크기 (`poc_task4` 측정)**

| 형태 | 측정 행 수 | 테이블 | 기본 키 인덱스 | 보조 인덱스 |
|------|-----------|--------|----------------|-------------|
| 넓은 형태 (`price_history`) | 2,886,870 (30일) | 337 MB, 행당 122.5 B | 111 MB, 행당 40.3 B | 105 MB, 행당 38.1 B |
| 긴 형태 (`price_history_long`) | 3,487,175 (7일) | 255 MB, 행당 76.6 B | 160 MB, 행당 48.2 B | 없음 |

**1년 추정**

| 형태 | 1년 행 수 | 테이블 | 인덱스 | 합계 | 보존하는 필드 |
|------|-----------|--------|--------|------|---------------|
| **넓은 형태 (선정)** | 5,040만 | 6.2 GB | 4.0 GB (기본 키 2.0 + 보조 1.9) | **약 10 GB** | 전부 |
| 긴 형태, 전체 필드 | 2억 2,370만 | 17.1 GB | 10.8 GB (기본 키만) | **약 28 GB** + 순위용 인덱스 | 전부 |
| 긴 형태, 소스별 대표 필드 1개 | 5,040만 | 3.9 GB | 2.4 GB (기본 키만) | 약 6.3 GB + 순위용 인덱스 | 대표값만 |

**결정: 넓은 형태로 바꾼다 (D3 변경).**

1. 같은 필드를 보존할 때 긴 형태의 약 2.8배 공간을 쓴다. 긴 형태는 행마다 헤더와 키(변형·소스·필드 이름·날짜)를 반복한다.
2. 대표 필드만 남기는 긴 형태는 넓은 형태보다 작지만, Cardmarket `avg7`·`avg30`처럼 다시 만들 수 없는 값을 잃는다. 넓은 형태에서 빈 필드는 NULL 비트맵만 쓴다.
3. 화면·순위·평가액은 대표값 `price` 하나만 읽으면 되어 쿼리가 단순하다.
4. 단점은 소스 필드가 늘면 컬럼 추가 마이그레이션이 필요하다는 점이다. 소스가 두 개로 확정되어 있어 받아들일 만하다.

**파티셔닝: 도입하지 않는다.** 1년 5천만 행은 기본 키 범위 스캔(S1·S2·S4)에 영향이 없고, 순위 쿼리(S3)는 날짜 하루치만 읽는다. 다만 다음 두 경우에는 다시 판단한다.

- TCGCSV 일별 압축 파일(2024-02-08~)로 과거 이력을 채우는 경우 (#6): 약 1,000일 × 75,000행 = 7,500만 행, 약 15 GB가 더해진다.
- 보존 기간을 1년보다 길게 잡는 경우

### 9. 2단계 작업지시자 확인 사항

**Q5·Q6 모두 제안대로 결정 (작업지시자, 2026-11-06).** Q5에 따라 6절 DDL 끝에 `pokemon_species`와 `cards.dex_ids` 인덱스를 더했다.

| # | 사항 | 제안 | 영향 |
|---|------|------|------|
| Q5 | **한글 이름 검색이 거의 되지 않는다.** Q1 결정으로 한글 이름이 있는 카드는 TCGdex `ko` 239장뿐이고, `피카츄`·`피카`는 0건이었다 (`ピカチュウ` 63건, `pikachu` 243건) | **포켓몬 종 이름 테이블을 추가한다.** `pokemon_species (dex_id PK, name_ko, name_ja, name_en)`를 두고 `cards.dex_ids`와 연결해, 한글 종 이름으로 일본판·영어판 카드를 찾게 한다. 이번 타스크에서는 테이블과 `dex_ids` 컬럼만 만들고, 데이터 소스(PokeAPI 등)·이용 조건 확인과 적재는 #5, 검색 쿼리는 #8에서 한다 | 테이블 1개 추가 (13개). 트레이너스·에너지 카드는 `dexId`가 없어 한글 검색 대상에서 빠진다 |
| Q6 | 상태 등급 값 | TCGplayer 기준 5단계 `near_mint`, `lightly_played`, `moderately_played`, `heavily_played`, `damaged`. TCGCSV 가격이 Near Mint 기준이라 비교하기 쉽다. **감정 카드(PSA 등)는 다루지 않는다** | `collection_items`, `user_trades`의 `CHECK`. 국내 거래 관행(S·A·B급)과 다르므로 #13 화면에서 한글 설명이 필요하다 |

**후속 타스크에 넘길 사항 (결정 불필요)**

- #6: TCGdex Cardmarket 가격은 카드 단건 조회에만 있어, 하루 수집에 카드 수만큼(`en`·`ja` 약 3만 6천 회) 요청이 필요하다. 동시성 2, 요청 간격을 두면 수 시간이 걸릴 수 있다.
- #6: Cardmarket의 `-holo` 필드는 카드 변형이 `normal` + `reverse_holofoil`이면 리버스 홀로, `holofoil`만 있으면 홀로 변형에 넣어야 한다. 연결 규칙은 #6에서 표본으로 확인한다.
- #5: 한글판 카드 연결(`card_links`)이 한글판 → 일본판 방향인지는 DB 제약으로 막지 않는다. 수집기가 언어를 확인한다.

### 10. Go 타입과 기본 쿼리 (4단계)

#### 금액 Go 타입 결정 (D4)

세 후보로 같은 쿼리의 sqlc 코드를 생성하고, 임시 DB에서 pgx v5.12.0으로 저장·조회했다 (2026-11-06).

| 확인 항목 | `pgtype.Numeric` (기본) | `string` | `shopspring/decimal` |
|-----------|------------------------|----------|----------------------|
| 생성 타입 (NOT NULL / NULL) | `pgtype.Numeric` / 같음 | `string` / `*string` | `decimal.Decimal` / `*decimal.Decimal` |
| 저장 `1.239` → `numeric(12,2)` | `1.24` | `1.24` | `1.24` |
| 잘못된 값 | 값을 만들 때 `Scan`이 오류 반환 (문서 기준, 미측정) | DB에서 `22P02` 오류 (`"abc"`). `"1e3"`은 통과 | `RequireFromString`에서 거부 |
| JSON 출력 | 숫자 `1.24` | 문자열 `"1.24"` (자릿수 유지) | 문자열 `"1.24"`. 뒤 0은 빠짐(`1500.00` → `"1500"`) |
| Go 산술 | 없음 (`Float64Value()` 변환) | 없음 | 있음 (`Mul`, `Round` 등) |
| pgx 연결 설정 | 불필요 | 불필요 | **불필요** (`sql.Scanner`·`driver.Valuer`로 동작 확인. `pgx-shopspring-decimal` 어댑터 없이 됨) |
| 추가 의존성 | 없음 | 없음 | `github.com/shopspring/decimal` 1개 (외부 의존성 없음). goose가 이미 간접 의존 |

**결정: `shopspring/decimal` v1.6.0.**

1. 원화 환산(Q3), 사용자 거래가 이상치 제거(#7), 평가액(#10)을 SQL로 하지 못하는 경우에도 Go에서 정확히 계산할 수 있다. `pgtype.Numeric`과 `string`은 `float64`로 바꿔야 해서 오차가 생긴다.
2. 타입으로 금액임이 드러나고, 잘못된 문자열이 DB까지 가지 않는다.
3. 추가 의존성이 하나이고 pgx 연결 설정이 필요 없다.

- 버전: 처음에 v1.4.0으로 고정하자 goose가 v3.29.0에서 v3.28.0으로 내려갔다(goose가 decimal v1.5.0 이상을 요구). 그래서 최신 v1.6.0(2026-10-18)을 쓴다.
- JSON 형식(문자열/숫자, 소수 자릿수)은 API 응답 구조체에서 정한다(#8). DB 구조체(`dbq`)에는 JSON 태그를 만들지 않는다. `dbq.User`를 그대로 직렬화하면 `password_hash`가 나가기 때문이다.

#### sqlc 설정 (`src/backend/sqlc.yaml`)

| DB 타입 | Go 타입 (NOT NULL / NULL) |
|---------|---------------------------|
| `numeric` | `decimal.Decimal` / `*decimal.Decimal` |
| `uuid` | `uuid.UUID` / `*uuid.UUID` (`github.com/google/uuid` v1.6.0, 기존 간접 의존을 직접 의존으로) |
| `timestamptz`, `date` | `time.Time` / `*time.Time` |
| 그 밖의 NULL 허용 컬럼 (`text`, `integer`) | `*string`, `*int32` (`emit_pointers_for_null_types`) |

- 생성 코드에 `pgtype`이 남지 않는다. NULL은 모두 `nil`로 다룬다.
- `timestamptz`는 `db_type: timestamptz`로 써야 적용된다. `pg_catalog.timestamptz`로 쓰면 무시되어 `pgtype.Timestamptz`가 그대로 생성되었다.

#### 기본 쿼리 (`src/backend/internal/db/queries/`)

| 파일 | 쿼리 | 쓰는 타스크 |
|------|------|-------------|
| `catalog.sql` | `UpsertSet`, `UpsertCard`, `UpsertCardVariant`, `GetCard`, `ListCardVariants` | #5, #8 |
| `prices.sql` | `GetPriceSourceByCode`, `UpsertExternalID`, `FindVariantByExternalID`, `UpsertPriceHistory`(`:batchexec`, pgx 배치), `ListPriceHistory`, `UpsertFxRate`, `GetFxRateOnOrBefore` | #6, #7, #8 |
| `users.sql` | `CreateUser`, `GetUserByEmail`, `UpsertCollectionItem`, `ListCollectionItems`, `CreateUserTrade` | #9, #10 |

- 저장 쿼리는 모두 `ON CONFLICT ... DO UPDATE`라 수집기를 다시 실행해도 행이 늘지 않는다.
- `UpsertCard`는 `has_ko_edition`을 바꾸지 않는다. 한글판 연결 작업(#5)이 따로 갱신한다.
- 순위·변동률·평가액·일본판 환산 쿼리(7절 S2~S4)는 화면 요구가 정해지는 #7·#8·#10에서 추가한다.

#### 후속 타스크 인계

| 타스크 | 넘기는 내용 |
|--------|-------------|
| #5 카탈로그 수집기 | `UpsertSet`·`UpsertCard`·`UpsertCardVariant`로 적재. 한글판은 `card_links`(method `auto`)와 `cards.has_ko_edition`. `cards.dex_ids`와 `pokemon_species`(한·일·영 종 이름) 적재, 종 이름 데이터 소스의 이용 조건 확인 |
| #6 해외 시세 수집기 | 외부 ID는 `external_ids`(TCGplayer `productId` + `subTypeName`). 가격은 `UpsertPriceHistory` 배치로 저장하고 `price_date`는 소스 기준일. 대표값 규칙은 6절 표. Cardmarket `-holo` 필드의 변형 연결 규칙 확인. TCGdex Cardmarket은 하루 약 3만 6천 회 요청 |
| #7 국내 거래가 | 일본판 참고가는 저장하지 않고 조회 시 환산 (Q3). 사용자 거래가 집계는 `price_history`(소스 `user_aggregate`, `price_basis` `median`, `sample_count`). 관리자 입력은 `admin_price_entries` + `price_history`(소스 `admin`) |
| #8 시세 API | 가격 성격·통화·출처는 `price_sources`에서 읽는다. 순위 쿼리는 `price_history_source_id_price_date_idx` 사용(7절 S3). 한글 검색은 `pokemon_species.name_ko` → `cards.dex_ids` (`&&` 연산, GIN 인덱스) |
| #9 인증 | `users`(`uuidv7`, 이메일은 `lower(email)` 유일). 토큰·세션 테이블은 필요 시 마이그레이션 추가 |
| #10 컬렉션 API | `collection_items`는 변형·상태 등급마다 1행. 사용자 데이터가 있는 변형을 지우면 SQLSTATE `23001`(restrict_violation) |

## 결론

1. 테이블은 13개다(`pokemon_species` 포함): `sets`, `cards`, `card_variants`, `card_links`, `price_sources`, `external_ids`, `price_history`, `fx_rates`, `users`, `collection_items`, `user_trades`, `admin_price_entries`. Q5가 승인되면 `pokemon_species`가 더해져 13개가 된다.
2. 열거 값은 `text` + `CHECK`로, 가격 소스는 참조 테이블로 한다. `enum`은 값을 되돌릴 수 없어 쓰지 않는다.
3. 기본 키는 `bigint identity`로 하고, `users`만 `uuidv7()`, `price_sources`는 `smallint`로 한다.
4. **가격 이력은 넓은 형태로 확정한다** (2단계 측정). 1년 약 5천만 행, 약 10 GB로 추정하며 파티셔닝은 하지 않는다. 화면·순위는 대표값 `price`를 읽는다.
5. 금액은 DB에서 `numeric(12,2)`(원화 원본 입력은 `numeric(12,0)`), 환율은 `numeric(18,8)`이다. Go 타입은 `shopspring/decimal`로 정했다 (10절).
6. 사용자 거래가와 관리자 입력가는 원본 테이블에 두고, 시세로 쓰는 값만 `price_history`에 넣는다.
7. Q1~Q4는 제안대로 결정되었다 (2026-11-06). Q3에 따라 일본판 참고가는 저장하지 않고 조회할 때 환산하며, #7 A안은 조회 쿼리로 바뀐다.
8. Q5·Q6은 제안대로 결정되었다 (2026-11-06). 한글 이름 검색용 `pokemon_species`를 더해 테이블은 13개이고, 상태 등급은 TCGplayer 5단계이며 감정 카드는 다루지 않는다.

## 적용 방향

- 마이그레이션: `src/backend/migrations/00002_catalog.sql`, `00003_prices.sql`, `00004_users.sql` (6절 DDL과 같음)
- 쿼리·생성 코드: `src/backend/internal/db/queries/*.sql` → `src/backend/internal/db/dbq/`
- 테스트: `src/backend/internal/db/migrate_test.go`, `queries_test.go` (테스트 DB `${POSTGRES_DB}_test`)
- 후속 타스크 인계는 10절 마지막 표

## 참고 자료

- PostgreSQL 18 문서: `ALTER TYPE` (https://www.postgresql.org/docs/18/sql-altertype.html), UUID 함수 (https://www.postgresql.org/docs/18/functions-uuid.html), `pg_trgm` (https://www.postgresql.org/docs/18/pgtrgm.html)
- goose 트랜잭션 비활성화 주석 `-- +goose NO TRANSACTION`: https://pressly.github.io/goose/documentation/annotations/
- `mydocs/tech/price_data_sources.md` "소스 확정안", "후속 타스크 설계 입력" 절
- `mydocs/tech/domestic_price_sources.md` "국내 거래가 대체 방안" 절
- `mydocs/tech/backend_stack.md` sqlc 기본 타입 메모
