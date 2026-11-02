#!/bin/sh
# APP 환경 변수로 실행할 cmd/ 하위 프로그램을 고른다 (기본: server).
# 해당 디렉터리가 없으면 초기화 명령(exec)을 받을 수 있게 대기한다.
set -e

APP="${APP:-server}"

if [ -f /app/go.mod ] && [ -d "/app/cmd/$APP" ]; then
  # go run 은 SIGTERM 을 프로그램에 전달하지 않으므로, 빌드 후 exec 로 PID 1 을 넘긴다.
  go build -o "/tmp/$APP" "./cmd/$APP"
  exec "/tmp/$APP" "$@"
fi

echo "[backend] /app/cmd/$APP 가 없어 대기합니다. 'docker compose exec backend sh' 로 접속하세요."
exec tail -f /dev/null
