# 수집기 별도 장비 실행 가이드

개발 PC 대신 상시 켜 두는 장비(리눅스 서버, 미니 PC, NAS 등)에서 수집기를 돌리는 방법이다.
실행은 `scripts/collector.sh` 하나로 한다.

## 수집기가 하는 일

| 작업 | 시각 | 내용 | 소요 시간 (참고) |
|------|------|------|-----------|
| `collect-daily` | 매일 05:30 KST | 카탈로그: 종 이름 적재 → 세트 목록·세트 상세(약 520회) → 신규 카드만 카드 상세 → 한글판 연결. 이어서 해외 시세: 환율(Frankfurter 2회) ‖ TCGplayer 가격(TCGCSV 약 1,370회) ‖ Cardmarket 가격(TCGdex 카드 상세 약 3만 5천 회) | 카탈로그 수 분 + 시세 약 2시간 30분 |
| 〃 (전체 갱신) | 매월 1일 05:30 KST | 위와 같고, 카탈로그가 모든 카드의 카드 상세를 다시 받는다 | 카탈로그 3~4시간 + 시세 약 2시간 30분 |
| 첫 적재 | 직접 실행 | 빈 DB에서 모든 카드 상세를 받고 시세를 수집한다 | 6시간 안팎 |

- 작업은 겹쳐 실행되지 않는다. 이전 실행이 끝나지 않았으면 그 회차를 건너뛴다.
- 일부 세트만 실패하면 로그에 `WARN`을 남기고 다음 실행에서 다시 시도한다.
- 카탈로그가 실패해도 시세는 이미 적재된 카드로 수집한다.
- 외부 호출 규칙은 `mydocs/tech/catalog_collector.md`(TCGdex 요청 간격 250ms, 동시성 2)와 `mydocs/tech/price_collector.md`(TCGCSV 요청 간 120ms, 동시성 1) 참고. TCGdex는 카탈로그와 시세가 같은 요청 간격을 나눠 쓴다.

## 준비물

- Docker Engine과 Compose v2 (`docker compose version`으로 확인)
- bash 4.4 이상, git
- 디스크 여유 공간 약 2GB (Go 이미지·빌드 캐시·DB)
- 인터넷 (TCGdex `api.tcgdex.net` 접속)

## 설치

```bash
git clone https://github.com/Chocobone/PokeCardPrice.git
cd PokeCardPrice
git checkout devel
scripts/collector.sh setup        # Docker 확인, Docker/.env 생성, 이미지 빌드
```

`Docker/.env`를 열어 `POSTGRES_PASSWORD`를 바꾼다.

## DB 위치 선택

| 방식 | 설정 | 설명 |
|------|------|------|
| 이 장비에 DB (기본) | `COLLECTOR_DATABASE_URL` 비움 | 스크립트가 compose의 `db`를 함께 띄운다. 데이터는 Docker 볼륨 `pokecardprice_db-data`에 남는다 |
| 다른 장비의 DB | `Docker/.env`에 `COLLECTOR_DATABASE_URL=postgres://사용자:비밀번호@호스트:5432/DB?sslmode=disable` | 수집기만 띄운다. DB 장비의 5432 포트가 이 장비에서 접속 가능해야 한다. 마이그레이션은 수집기가 시작할 때 자동 적용한다 |

## 사용법

```bash
scripts/collector.sh once -d      # 첫 적재 (백그라운드, 로그는 logs/collector_*.log)
tail -f logs/collector_*.log      # 진행 확인. 끝나면 "catalog: done" 줄이 남는다
scripts/collector.sh status       # 컨테이너 상태와 언어별 세트·카드 수
scripts/collector.sh start        # 상시 실행 시작 (매일 05:30)
scripts/collector.sh logs         # 상시 실행 로그
scripts/collector.sh stop         # 중지
```

| 명령 | 설명 |
|------|------|
| `setup` | Docker 확인, `Docker/.env` 생성, 이미지 빌드 |
| `start` | 수집기 상시 실행. 장비가 재부팅되어도 Docker가 다시 띄운다 (`restart: unless-stopped`) |
| `stop` / `restart` | 상시 실행 수집기 중지 / 재시작 |
| `once [-d]` | 1회 실행. `-d`면 백그라운드로 실행해 SSH 접속이 끊겨도 계속된다 |
| `full [-d]` | 전체 갱신으로 1회 실행 |
| `status` | 컨테이너 상태와 카탈로그 행 수, 소스별 최근 시세 기준일·행 수, 환율 (이 장비 DB일 때만) |
| `logs` | 상시 실행 로그 (Ctrl+C로 빠져나옴) |

- `once`·`full`은 상시 실행 수집기가 떠 있으면 실행하지 않는다 (같은 테이블에 동시에 쓰지 않도록). `stop` 후 실행하고, 끝나면 `start`한다.
- 범위를 줄여 시험할 때: `CATALOG_LANGUAGES=en CATALOG_SETS=sv03.5 PRICES_TCGCSV_GROUPS=23237 scripts/collector.sh once`
  - 시세 범위는 카탈로그와 같은 언어·세트를 쓴다. TCGCSV 영어판 그룹은 세트 ID로 고를 수 없어 `PRICES_TCGCSV_GROUPS`로 지정한다 (예: `sv03.5` → `23237`).
  - 카탈로그만 돌리려면 `PRICES_SOURCES=none`, 시세 작업 일부만 돌리려면 `PRICES_SOURCES=fx,tcgcsv`.
- 상시 실행(`start`)의 설정은 `Docker/.env`에 적는다. 실행 시 지정한 환경 변수는 `once`·`full`에만 적용된다.

## 업데이트

```bash
git pull
scripts/collector.sh restart      # 컨테이너가 시작할 때 소스를 다시 빌드한다
```

## 문제 해결

| 증상 | 확인 |
|------|------|
| `Docker 데몬에 접속할 수 없습니다` | `sudo systemctl start docker`, 사용자를 `docker` 그룹에 추가 (`sudo usermod -aG docker $USER` 후 재로그인) |
| `scripts/collector.sh: Permission denied` | `chmod +x scripts/collector.sh` 또는 `bash scripts/collector.sh ...` |
| `/usr/bin/env: 'bash\r'` | 줄바꿈이 CRLF로 바뀐 경우. `git config core.autocrlf input` 후 `git checkout -- scripts/collector.sh` |
| 로그에 `catalog: set failed` | 해당 세트만 실패. 다음 실행에서 다시 시도된다. 계속 실패하면 TCGdex에서 그 세트 응답 확인 |
| 로그에 `tcgdex request retry`가 많음 | TCGdex 일시 장애 또는 호출 제한. 자동 재시도(최대 3회)한다 |
