#!/bin/sh
# package.json 이 있으면 의존성 설치 후 dev 서버를 실행하고, 없으면 초기화 명령(exec)을 받을 수 있게 대기한다.
set -e

if [ -f /app/package.json ]; then
  if [ ! -d /app/node_modules/.bin ]; then
    npm install
  fi
  exec npm run dev -- --host 0.0.0.0 --port 5173
fi

echo "[frontend] /app/package.json 이 없어 대기합니다. 'docker compose exec frontend sh' 로 접속하세요."
exec tail -f /dev/null
