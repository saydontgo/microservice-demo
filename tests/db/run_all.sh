#!/usr/bin/env bash
set -euo pipefail

MODE="${1:-all}"
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

MYSQL_HOST="${MYSQL_HOST:-127.0.0.1}"
MYSQL_PORT="${MYSQL_PORT:-3306}"
MYSQL_USER="${MYSQL_USER:-demo}"
MYSQL_PASSWORD="${MYSQL_PASSWORD:-}"
MYSQL_DATABASE="${MYSQL_DATABASE:-microservice_demo}"

run_mysql() {
  echo "[MySQL] running smoke test on ${MYSQL_HOST}:${MYSQL_PORT}/${MYSQL_DATABASE}"
  local args=(-h "$MYSQL_HOST" -P "$MYSQL_PORT" -u "$MYSQL_USER" "$MYSQL_DATABASE")
  if [[ -n "$MYSQL_PASSWORD" ]]; then
    args=(-h "$MYSQL_HOST" -P "$MYSQL_PORT" -u "$MYSQL_USER" -p"$MYSQL_PASSWORD" "$MYSQL_DATABASE")
  fi
  mysql "${args[@]}" < "$ROOT_DIR/tests/db/mysql_smoke_test.sql"
}

run_redis() {
  echo "[Redis] running smoke test on ${REDIS_HOST:-127.0.0.1}:${REDIS_PORT:-6379}/${REDIS_DB:-0}"
  bash "$ROOT_DIR/tests/db/redis_smoke_test.sh"
}

case "$MODE" in
  mysql)
    run_mysql
    ;;
  redis)
    run_redis
    ;;
  all)
    run_mysql
    run_redis
    ;;
  *)
    echo "Usage: bash tests/db/run_all.sh [all|mysql|redis]" >&2
    exit 2
    ;;
esac

echo "PASS: selected database tests completed"
