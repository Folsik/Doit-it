#!/usr/bin/env bash
# Explicit, local-development-only fixture. Never run against production.
set -euo pipefail

: "${DEV_TEST_EMAIL:?Set DEV_TEST_EMAIL to a non-production test email}"
: "${DEV_TEST_PASSWORD:?Set DEV_TEST_PASSWORD to a unique 12-72 character test password}"
if [[ "${DEV_TEST_PASSWORD}" == *$'\n'* || ${#DEV_TEST_PASSWORD} -lt 12 || ${#DEV_TEST_PASSWORD} -gt 72 ]]; then
  echo "Invalid development password length/format" >&2
  exit 1
fi
if [[ "${DEV_TEST_EMAIL}" != *@* ]]; then
  echo "Invalid email" >&2
  exit 1
fi
API_URL="${DEV_API_URL:-http://127.0.0.1:8080}"
case "$API_URL" in
  http://127.0.0.1:8080|http://localhost:8080) ;;
  *) echo "Refusing to seed non-local API: $API_URL" >&2; exit 1 ;;
esac
if [[ "${APP_ENV:-development}" != development ]]; then
  echo "Refusing to seed when APP_ENV is not development" >&2
  exit 1
fi
payload=$(DEV_TEST_EMAIL="$DEV_TEST_EMAIL" DEV_TEST_PASSWORD="$DEV_TEST_PASSWORD" python3 - <<'PY'
import json, os
print(json.dumps({"email":os.environ["DEV_TEST_EMAIL"],"password":os.environ["DEV_TEST_PASSWORD"],"full_name":"Development Test User","country_code":"DE"}))
PY
)
response_file=$(mktemp)
trap 'rm -f "$response_file"' EXIT
status=$(curl --silent --show-error --output "$response_file" --write-out '%{http_code}' \
  --header 'Content-Type: application/json' --data "$payload" "$API_URL/v1/auth/register")
case "$status" in
  201) echo "Development-only test user created; account initialized with zero balance." ;;
  409) echo "Development test user already exists; no changes made." ;;
  *) echo "Seed failed (HTTP $status)" >&2; exit 1 ;;
esac
