#!/usr/bin/env bash
# PokeCardPrice 수집기 실행 스크립트.
# 개발 PC 가 아닌 별도 장비(상시 켜 두는 서버, 미니 PC 등)에서 수집기만 돌릴 때 쓴다.
#
# 필요한 것: 이 저장소 clone, Docker(Compose v2 포함), bash.
# 자세한 사용법: mydocs/manual/collector_remote.md
#
# 사용법: scripts/collector.sh <명령> [옵션]
#   setup            Docker 확인, Docker/.env 준비, 이미지 빌드
#   start            수집기 상시 실행 (매일 05:30 KST collect-daily, 매월 1일 전체 갱신)
#   stop             수집기 중지 (로컬 DB 는 그대로 둔다)
#   restart          수집기 재시작 (소스를 git pull 한 뒤 반영)
#   once [-d]        수집 작업 1회 실행. 빈 DB 첫 적재는 수 시간 걸린다
#   full [-d]        전체 갱신으로 1회 실행 (모든 카드 상세 재조회)
#   status           컨테이너 상태와 DB 카탈로그·시세 행 수
#   logs             상시 실행 수집기 로그 보기 (Ctrl+C 로 빠져나옴)
#
#   -d               백그라운드로 실행하고 logs/ 에 로그를 남긴다 (SSH 접속이 끊겨도 계속 실행)
#
# 환경 변수 (Docker/.env 또는 실행 시 지정):
#   COLLECTOR_DATABASE_URL  다른 장비의 DB 에 적재할 때 지정. 비우면 이 장비의 compose db 를 띄워 쓴다
#   CATALOG_LANGUAGES       수집 언어 (예: en,ja). 비우면 en,ja,ko
#   CATALOG_SETS            수집 세트 (예: sv03.5,SV5K). 비우면 전체
#   CATALOG_FULL            true 면 매 실행 전체 갱신
#   PRICES_SOURCES          시세 작업 (fx,tcgcsv,tcgdex_cardmarket). 비우면 전체, none 이면 끔
#   PRICES_TCGCSV_GROUPS    TCGCSV 그룹 ID (예: 23237, 개발용)
#   COLLECTOR_INTERVAL      일정 대신 이 간격으로 반복 (예: 30m, 개발용)
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DOCKER_DIR="$ROOT/Docker"
LOG_DIR="$ROOT/logs"
ENV_FILE="$DOCKER_DIR/.env"

die() { echo "오류: $*" >&2; exit 1; }
info() { echo "[collector.sh] $*"; }

compose() { docker compose --project-directory "$DOCKER_DIR" -f "$DOCKER_DIR/docker-compose.yml" "$@"; }

# .env 의 값을 읽는다 (실행 시 지정한 환경 변수가 우선).
env_value() {
  local key="$1" val="${!1:-}"
  if [[ -z "$val" && -f "$ENV_FILE" ]]; then
    val="$(grep -E "^${key}=" "$ENV_FILE" | tail -n1 | cut -d= -f2- | tr -d '\r' || true)"
  fi
  printf '%s' "$val"
}

# 외부 DB 를 쓰면 compose 의 db 를 띄우지 않는다.
external_db() { [[ -n "$(env_value COLLECTOR_DATABASE_URL)" ]]; }

require_docker() {
  command -v docker >/dev/null 2>&1 || die "docker 가 없습니다. Docker Engine(Compose v2 포함)을 설치하세요"
  docker compose version >/dev/null 2>&1 || die "docker compose(v2) 를 쓸 수 없습니다"
  docker info >/dev/null 2>&1 || die "Docker 데몬에 접속할 수 없습니다 (데몬 실행 여부, docker 그룹 권한 확인)"
}

require_env() {
  [[ -f "$ENV_FILE" ]] || die "Docker/.env 가 없습니다. 먼저 'scripts/collector.sh setup' 을 실행하세요"
}

# 실행 시 지정한 CATALOG_*·PRICES_*·COLLECTOR_INTERVAL 을 컨테이너에 넘긴다 (.env 값보다 우선).
pass_env_args() {
  local k
  for k in CATALOG_LANGUAGES CATALOG_SETS CATALOG_FULL PRICES_SOURCES PRICES_TCGCSV_GROUPS COLLECTOR_INTERVAL; do
    if [[ -n "${!k:-}" ]]; then printf -- '-e\n%s=%s\n' "$k" "${!k}"; fi
  done
}

cmd_setup() {
  require_docker
  if [[ ! -f "$ENV_FILE" ]]; then
    cp "$DOCKER_DIR/.env.example" "$ENV_FILE"
    info "Docker/.env 를 만들었습니다. POSTGRES_PASSWORD 를 바꾸고, 외부 DB 를 쓰면 COLLECTOR_DATABASE_URL 을 지정하세요"
  else
    info "Docker/.env 가 이미 있습니다"
  fi
  if grep -qE '^POSTGRES_PASSWORD=change-me' "$ENV_FILE" && ! external_db; then
    info "경고: POSTGRES_PASSWORD 가 기본값(change-me)입니다"
  fi
  compose build collector
  info "준비 완료. 'scripts/collector.sh once -d' 로 첫 적재, 'scripts/collector.sh start' 로 상시 실행을 시작하세요"
}

cmd_start() {
  require_docker; require_env
  local args=()
  mapfile -t args < <(pass_env_args)
  if ((${#args[@]})); then
    info "주의: start 는 실행 시 지정한 환경 변수를 넘기지 못합니다. Docker/.env 에 적으세요 (${args[*]})"
  fi
  if external_db; then
    compose up -d --build --no-deps collector
  else
    compose up -d --build db collector
  fi
  info "수집기를 시작했습니다. 다음 실행 시각은 'scripts/collector.sh logs' 의 job scheduled 줄에서 확인하세요"
}

cmd_stop() {
  require_docker; require_env
  compose stop collector
}

cmd_restart() {
  require_docker; require_env
  compose restart collector
}

# run_once 는 일회용 컨테이너에서 수집기를 -once 로 실행한다.
# 상시 실행 중인 수집기와 겹치지 않도록 실행 중이면 멈추고 알린다.
run_once() {
  local full="$1" detach="$2"
  require_docker; require_env
  if [[ -n "$(compose ps -q --status running collector 2>/dev/null)" ]]; then
    die "상시 실행 수집기가 떠 있습니다. 작업이 겹치지 않도록 'scripts/collector.sh stop' 후 다시 실행하세요"
  fi
  local args=()
  mapfile -t args < <(pass_env_args)
  if [[ "$full" == true ]]; then args+=(-e CATALOG_FULL=true); fi
  local deps=()
  if external_db; then deps=(--no-deps); else compose up -d --wait db; fi
  # -T: TTY 없이 실행 (nohup·cron 에서도 동작). --rm: 끝나면 컨테이너 삭제.
  local cmd=(docker compose --project-directory "$DOCKER_DIR" -f "$DOCKER_DIR/docker-compose.yml"
    run --rm -T "${deps[@]}" "${args[@]}" collector -once)

  if [[ "$detach" == true ]]; then
    mkdir -p "$LOG_DIR"
    local log
    log="$LOG_DIR/collector_$(date +%Y%m%d_%H%M%S).log"
    nohup "${cmd[@]}" >"$log" 2>&1 </dev/null &
    info "백그라운드로 시작했습니다 (PID $!)"
    info "진행 확인: tail -f $log"
    info "끝나면 마지막 줄에 'catalog: done' 또는 'catalog: failed' 가 남습니다"
  else
    "${cmd[@]}"
  fi
}

cmd_status() {
  require_docker; require_env
  compose ps
  if external_db; then
    info "외부 DB 를 사용 중이라 행 수는 표시하지 않습니다"
    return
  fi
  if [[ -n "$(compose ps -q --status running db 2>/dev/null)" ]]; then
    # shellcheck disable=SC2016 # $POSTGRES_* 는 db 컨테이너 안에서 확장한다
    compose exec -T db sh -c 'psql -q -U "$POSTGRES_USER" -d "$POSTGRES_DB"' <<'SQL'
SELECT s.language, count(DISTINCT s.id) AS sets, count(c.id) AS cards
FROM sets s LEFT JOIN cards c ON c.set_id = s.id GROUP BY 1 ORDER BY 1;
SELECT (SELECT count(*) FROM card_links) AS links,
       (SELECT count(*) FROM cards WHERE has_ko_edition) AS has_ko_edition,
       (SELECT count(*) FROM pokemon_species) AS species;
SELECT ps.code AS price_source, max(p.price_date) AS latest_price_date, count(p.*) AS price_rows
FROM price_sources ps JOIN price_history p ON p.source_id = ps.id GROUP BY 1 ORDER BY 1;
SELECT base_currency || '/' || quote_currency AS fx_pair, max(rate_date) AS latest_rate_date, count(*) AS rates
FROM fx_rates GROUP BY 1 ORDER BY 1;
SQL
  fi
}

cmd_logs() {
  require_docker; require_env
  compose logs -f --tail 100 collector
}

usage() { sed -n '2,27p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//'; }

main() {
  local action="${1:-}"
  [[ $# -gt 0 ]] && shift
  local detach=false
  if [[ "${1:-}" == "-d" ]]; then detach=true; shift; fi
  [[ $# -eq 0 ]] || die "알 수 없는 인자: $*"
  case "$action" in
    setup) cmd_setup ;;
    start) cmd_start ;;
    stop) cmd_stop ;;
    restart) cmd_restart ;;
    once) run_once false "$detach" ;;
    full) run_once true "$detach" ;;
    status) cmd_status ;;
    logs) cmd_logs ;;
    ""|-h|--help|help) usage ;;
    *) usage; die "알 수 없는 명령: $action" ;;
  esac
}

main "$@"
