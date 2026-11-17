# Task #5: 카드 카탈로그 수집기 — 수행 계획서

## 수행 목표

TCGdex 공개 API에서 영어(`en`)·일본어(`ja`)·한국어(`ko`) 세트와 카드 목록을 받아 #4에서 만든 카탈로그 테이블(`sets`, `cards`, `card_variants`, `card_links`, `pokemon_species`)에 적재하고, 다시 실행하면 바뀐 내용만 갱신되도록 한다.
수집기(`cmd/collector`)의 일일 작업으로 등록해, #6(해외 시세)과 #7(일본판 참고가)이 카드·변형 행을 전제로 바로 시작할 수 있게 한다.

## 범위

1. **사전 확인**: TCGdex 이용 조건·호출 한도 재확인(#2 미해결 사항), 세트·카드 응답 필드 재확인(세트 목록, 세트 상세의 카드 요약, 카드 상세의 `variants`·`dexId`).
2. **한글판·일본판 일치 범위 검증**: 표본 세트로 한글판 세트 코드·카드 번호가 일본판과 일치하는 범위를 확인하고 자동 연결 규칙을 정한다(#2·#4 인계).
3. **TCGdex 클라이언트**: HTTP 클라이언트(타임아웃, 재시도, 요청 간격·동시성 상한, 전용 User-Agent)와 응답 타입.
4. **카탈로그 적재**: 언어별 세트 → 카드 → 카드 변형 upsert. 세트 단위 트랜잭션으로 일부 실패가 다른 세트에 번지지 않게 한다.
5. **변형 매핑**: TCGdex `variants`(normal·reverse·holo·firstEdition 등)를 `card_variants.variant` 코드(7종)로 바꾸는 규칙.
6. **한글판 연결**: TCGdex `ko`에 있는 카드는 `card_links`(method `auto`)로 일본판 카드에 연결하고, `ko` 미수록 한글판은 일본판 카드의 `has_ko_edition`만 표시한다(#4 Q1 결정). 판단 근거가 되는 한글 세트 목록은 2번 결과로 정한다.
7. **포켓몬 종 이름**: `pokemon_species`(한·일·영 종 이름) 데이터 소스 후보(PokeAPI 등)를 조사하고 이용 조건을 확인한 뒤 적재한다(#4 Q5 인계).
8. **수집기 연결**: `cmd/collector`의 일일 작업에 카탈로그 수집을 등록하고, `-once`로 수동 실행할 수 있게 한다. 처리 건수·실패 세트를 로그로 남긴다.
9. **테스트·문서**: 클라이언트는 고정 응답(`httptest`)으로, 적재는 테스트 DB로 검증한다. 조사 결과와 규칙은 기술 문서로 남긴다.

### 범위 밖

- 가격 수집, `external_ids`(TCGplayer·Cardmarket 상품 ID) 적재 → #6
- 일본판 참고가 산출, 수동 연결 입력 화면·API → #7
- 검색 쿼리·카드 조회 API → #8
- 카드 이미지 파일 저장(이미지는 URL만 저장)
- TCGdex `ko` 미수록 한글판 카드의 한글 카드 이름 수집 (가능 여부만 조사해 기록)

## 핵심 설계 원칙

- **멱등 적재**: #4의 `Upsert*` 쿼리(`ON CONFLICT ... DO UPDATE`)를 사용해 몇 번을 다시 실행해도 행이 늘지 않게 한다.
- **소스에 대한 배려**: 요청 간격과 동시성 상한을 두고, 이용 조건·호출 한도 확인 결과에 맞춘다. 실패 시 지수 백오프로 제한 횟수만 재시도한다.
- **부분 실패 허용**: 세트 하나의 실패가 전체 수집을 멈추지 않게 하고, 실패한 세트는 로그로 남겨 다음 실행에서 다시 시도한다.
- **삭제하지 않음**: 소스에서 사라진 세트·카드는 지우지 않는다(가격·컬렉션이 참조할 수 있음). 처리 방법은 필요하면 구현 계획서에서 제안한다.
- **문서에 없는 필드 의존 최소화**: `variants_detailed` 같은 비공식 필드에 카탈로그 적재가 의존하지 않게 한다.
- **조사 결과 승인 후 구현**: 일치 범위 검증·종 이름 소스 결과를 먼저 문서로 제시하고, 연결 규칙은 작업지시자 승인 후 구현한다.

## 위험과 대응

| 위험 | 대응 |
|------|------|
| 한글판·일본판 세트 코드·번호 일치 범위가 생각보다 좁음 | 자동 연결은 일치가 확인된 경우로만 한정하고, 나머지는 #7 수동 연결 대상으로 남긴다 |
| TCGdex 호출 한도가 문서에 없음 | 카드 상세 호출 수를 추정해 요청 간격을 정하고, 세트 상세(카드 요약)만으로 필요한 필드가 채워지는지 먼저 확인해 호출 수를 줄인다 |
| 종 이름 데이터 소스의 이용 조건이 불명확 | 후보별 라이선스를 비교해 보고하고, 적합한 소스가 없으면 작업지시자 결정을 받는다 |
| 첫 적재 시간이 길다 (카드 수만 장) | 세트 단위로 진행 상황을 기록하고, 개발 중에는 언어·세트 필터로 일부만 적재할 수 있게 한다 |

## 기존 구현 현황

- `src/backend/cmd/collector/main.go`: 일일 작업 `collect-prices` 자리만 있음 ("no collection tasks yet")
- `src/backend/internal/scheduler/`: gocron 작업 등록·중복 실행 방지·종료 대기
- `src/backend/internal/db/queries/catalog.sql`: `UpsertSet`, `UpsertCard`, `UpsertCardVariant`, `GetCard`, `ListCardVariants`
- `src/backend/migrations/00002_catalog.sql`: `sets`, `cards`, `card_variants`, `card_links`, `pokemon_species`
- `mydocs/tech/price_data_sources.md`: TCGdex 조사 결과, #5 설계 입력·미해결 사항
- `mydocs/tech/db_schema.md`: 10절 #5 인계 사항

## 산출물

- `mydocs/tech/catalog_collector.md`: TCGdex 이용 조건·응답 확인, 한글판·일본판 일치 검증, 변형 매핑, 종 이름 소스 비교 (신규)
- `src/backend/internal/tcgdex/`: TCGdex 클라이언트 (신규, 이름은 구현 계획서에서 확정)
- `src/backend/internal/catalog/`: 카탈로그 적재 로직 (신규, 이름은 구현 계획서에서 확정)
- `src/backend/internal/db/queries/catalog.sql`: 한글판 연결·종 이름 적재에 필요한 쿼리 추가와 생성 코드
- `src/backend/cmd/collector/main.go`: 카탈로그 수집 작업 등록
- 테스트, `mydocs/working/task_m050_5_stage{N}.md`, `mydocs/report/task_m050_5_report.md`

## 구현 계획서

`mydocs/plans/task_m050_5_impl.md` 참조 (본 계획서 승인 후 작성)

## 브랜치

`local/task5`

## 예상 단계: 4단계

1. 사전 확인과 검증: TCGdex 이용 조건·응답, 한글판·일본판 일치 범위, 변형 매핑, 종 이름 소스 → 작업지시자 승인
2. TCGdex 클라이언트와 세트·카드·변형 적재 (en·ja)
3. 한글판 연결(`card_links`, `has_ko_edition`)과 `pokemon_species` 적재
4. 수집기 일정 연결, 전체 적재 실측, 테스트·문서 정리
