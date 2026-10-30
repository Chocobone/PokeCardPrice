#!/bin/sh
# go.mod 가 있으면 서버를 실행하고, 없으면 초기화 명령(exec)을 받을 수 있게 대기한다.
set -e

if [ -f /app/go.mod ] && [ -d /app/cmd/server ]; then
  # go run 은 SIGTERM 을 서버 프로세스에 전달하지 않으므로, 빌드 후 exec 로 PID 1 을 넘긴다.
  go build -o /tmp/server ./cmd/server
  exec /tmp/server
fi

echo "[backend] /app/cmd/server 가 없어 대기합니다. 'docker compose exec backend sh' 로 접속하세요."
exec tail -f /dev/null
