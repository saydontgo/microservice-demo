#!/usr/bin/env bash
set -euo pipefail

REDIS_HOST="${REDIS_HOST:-127.0.0.1}"
REDIS_PORT="${REDIS_PORT:-6379}"
REDIS_PASSWORD="${REDIS_PASSWORD:-}"
REDIS_DB="${REDIS_DB:-0}"
PREFIX="microservice_demo_smoke_$(date +%s)_$$"

redis_cmd() {
  local args=(redis-cli --raw -h "$REDIS_HOST" -p "$REDIS_PORT" -n "$REDIS_DB")
  if [[ -n "$REDIS_PASSWORD" ]]; then
    args+=(-a "$REDIS_PASSWORD" --no-auth-warning)
  fi
  "${args[@]}" "$@"
}

cleanup() {
  redis_cmd DEL \
    "auth:token:${PREFIX}" \
    "product:search:smoke:1" \
    "idem:10001:${PREFIX}" \
    "seller:trend:10001:2026-05-18:2026-05-24" \
    "rate:login:${PREFIX}" >/dev/null || true
}
trap cleanup EXIT

pong="$(redis_cmd PING)"
if [[ "$pong" != "PONG" ]]; then
  echo "FAIL: Redis PING failed, got: $pong" >&2
  exit 1
fi

redis_cmd SET "auth:token:${PREFIX}" '{"userId":10001,"role":"SELLER"}' EX 7200 >/dev/null
token_ttl="$(redis_cmd TTL "auth:token:${PREFIX}")"
if (( token_ttl <= 0 )); then
  echo "FAIL: auth token TTL not set" >&2
  exit 1
fi

redis_cmd SET "product:search:smoke:1" '[{"productId":30001,"status":1}]' EX 60 >/dev/null
product_cache="$(redis_cmd GET "product:search:smoke:1")"
if [[ "$product_cache" != *'"status":1'* ]]; then
  echo "FAIL: product search cache should use numeric status" >&2
  exit 1
fi

set_nx_result="$(redis_cmd SET "idem:10001:${PREFIX}" PROCESSING NX EX 86400)"
if [[ "$set_nx_result" != "OK" ]]; then
  echo "FAIL: first idempotency SET NX should succeed" >&2
  exit 1
fi

set_nx_duplicate="$(redis_cmd SET "idem:10001:${PREFIX}" PROCESSING NX EX 86400 || true)"
if [[ -n "$set_nx_duplicate" ]]; then
  echo "FAIL: duplicate idempotency SET NX should be blocked" >&2
  exit 1
fi

redis_cmd SET "idem:10001:${PREFIX}" '{"status":2,"responseCode":200}' XX EX 86400 >/dev/null
idem_value="$(redis_cmd GET "idem:10001:${PREFIX}")"
if [[ "$idem_value" != *'"status":2'* ]]; then
  echo "FAIL: idempotency result should be updated to numeric success status" >&2
  exit 1
fi

redis_cmd SET "seller:trend:10001:2026-05-18:2026-05-24" '{"points":[]}' EX 300 >/dev/null
trend_ttl="$(redis_cmd TTL "seller:trend:10001:2026-05-18:2026-05-24")"
if (( trend_ttl <= 0 )); then
  echo "FAIL: seller trend TTL not set" >&2
  exit 1
fi

redis_cmd INCR "rate:login:${PREFIX}" >/dev/null
redis_cmd EXPIRE "rate:login:${PREFIX}" 600 >/dev/null
rate_ttl="$(redis_cmd TTL "rate:login:${PREFIX}")"
if (( rate_ttl <= 0 )); then
  echo "FAIL: login rate limit TTL not set" >&2
  exit 1
fi

echo "PASS: Redis smoke checks passed"
