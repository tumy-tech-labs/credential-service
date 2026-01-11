#!/usr/bin/env bash
set -euo pipefail

# Five-minute killer demo: human -> agent -> gateway authorize -> synthetic JWT -> sample API -> deny outside scope.

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "Missing required command: $1" >&2
    exit 1
  fi
}

require_cmd docker
require_cmd jq
require_cmd curl
require_cmd python3

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

echo "[1/8] Starting local stack (issuer, verifier/gateway, Postgres, sample API)..."
docker compose up -d postgres issuer verifier s2s-api >/dev/null

echo "[2/8] Waiting for services to be ready..."
wait_for() {
  local url=$1
  for _ in {1..30}; do
    if curl -sSf "$url" >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done
  echo "Service at $url did not become ready" >&2
  exit 1
}
wait_for http://localhost:8080/healthz
wait_for http://localhost:8081/healthz
wait_for http://localhost:8082/orders || true # API may 401 when unauthenticated

extract_claim() {
  local token=$1
  local claim=$2
  python3 - "$token" "$claim" <<'PY'
import base64, json, sys

raw = sys.argv[1]
claim = sys.argv[2]
try:
    payload_b64 = raw.split('.')[1]
    # Pad base64 safely
    padding = '=' * (-len(payload_b64) % 4)
    payload = base64.urlsafe_b64decode(payload_b64 + padding)
    data = json.loads(payload)
    print(data.get(claim, ""))
except Exception as exc:
    print(f"", file=sys.stderr)
    raise SystemExit(f"failed to decode claim {claim}: {exc}")
PY
}

human_did="did:example:human-demo"
audience="sample-api"

echo "[3/8] Issuing parent (human) credential..."
parent_vc=$(curl -sSf -X POST http://localhost:8080/v1/credentials/issue \
  -H "Content-Type: application/json" \
  -d '{"subject_did":"'"$human_did"'","ttl_seconds":600,"claims":{"aud":"'"$audience"'","scope":["orders:read","orders:write"]}}' | jq -r .credential)

echo "[4/8] Registering issuer in the trust registry..."
issuer_did=$(extract_claim "$parent_vc" iss)
curl -sSf -X POST http://localhost:8081/v1/trust/issuers \
  -H "Content-Type: application/json" \
  -d '{"issuer_did":"'"$issuer_did"'"}' >/dev/null

echo "[5/8] Spawning agent credential (short TTL, narrowed scope)..."
agent_did="did:example:agent-demo"
agent_vc=$(curl -sSf -X POST http://localhost:8080/v1/credentials/delegate \
  -H "Content-Type: application/json" \
  -d '{"parent_credential":"'"$parent_vc"'","delegate_did":"'"$agent_did"'","scope":["orders:read"],"ttl_seconds":120,"claims":{"aud":"'"$audience"'"}}' | jq -r .credential)

echo "[6/8] Calling gateway authorize for the agent chain (expect allow + synthetic JWT)..."
authz_json=$(curl -sSf -X POST http://localhost:8081/v1/gateway/authorize \
  -H "Content-Type: application/json" \
  -d '{"credentials":["'"$parent_vc"'","'"$agent_vc"'"],"expected_audience":"'"$audience"'","resource":"orders","action":"read","want_synthetic_jwt":true}')

echo "$authz_json" | jq
synthetic_jwt=$(echo "$authz_json" | jq -r .synthetic_jwt)

if [ "$synthetic_jwt" = "null" ] || [ -z "$synthetic_jwt" ]; then
  echo "Synthetic JWT was not minted; aborting." >&2
  exit 1
fi

echo "[7/8] Calling sample API with synthetic JWT (expect success)..."
curl -sSf http://localhost:8082/orders -H "Authorization: Bearer $synthetic_jwt" | jq

echo "[8/8] Attempting out-of-scope action (expect policy_denied)..."
deny_json=$(curl -sSf -X POST http://localhost:8081/v1/gateway/authorize \
  -H "Content-Type: application/json" \
  -d '{"credentials":["'"$parent_vc"'","'"$agent_vc"'"],"expected_audience":"'"$audience"'","resource":"orders","action":"write","want_synthetic_jwt":true}')
echo "$deny_json" | jq

echo "\nDone. You now have a full human → agent → gateway → synthetic JWT → API → deny walkthrough."
