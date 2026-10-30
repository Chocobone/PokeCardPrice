#!/bin/sh
# go.mod 가 있으면 서버를 실행하고, 없으면 초기화 명령(exec)을 받을 수 있게 대기한다.
set -e

if [ -f /app/go.mod ] && [ -d /app/cmd/server ]; then
  exec go run ./cmd/server
fi

echo "[backend] /app/cmd/server 가 없어 대기합니다. 'docker compose exec backend sh' 로 접속하세요."
exec tail -f /dev/null
