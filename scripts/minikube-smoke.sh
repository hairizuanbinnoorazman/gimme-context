#!/usr/bin/env bash
set -euo pipefail

NAMESPACE="${NAMESPACE:-gimme-context}"
RELEASE="${RELEASE:-gimme-context}"
PROFILE="${MINIKUBE_PROFILE:-minikube}"
PORT="${SMOKE_PORT:-18080}"
BASE_URL="http://127.0.0.1:${PORT}"

for command in minikube curl jq; do
  if ! command -v "$command" >/dev/null 2>&1; then
    echo "missing required command: $command" >&2
    exit 1
  fi
done

kubectl=(minikube --profile "$PROFILE" kubectl --)
"${kubectl[@]}" --namespace "$NAMESPACE" rollout status "deployment/${RELEASE}-api" --timeout=120s
"${kubectl[@]}" --namespace "$NAMESPACE" rollout status "deployment/${RELEASE}-worker" --timeout=120s
"${kubectl[@]}" --namespace "$NAMESPACE" rollout status "deployment/${RELEASE}-web" --timeout=120s

"${kubectl[@]}" --namespace "$NAMESPACE" port-forward "service/${RELEASE}-web" "${PORT}:8080" >/tmp/gimme-context-port-forward.log 2>&1 &
forward_pid=$!
trap 'kill "$forward_pid" 2>/dev/null || true' EXIT

for _ in $(seq 1 30); do
  if curl --fail --silent "$BASE_URL/health/live" >/dev/null; then
    break
  fi
  sleep 1
done

curl --fail --silent "$BASE_URL/" | grep -q "Gimme Context"
curl --fail --silent "$BASE_URL/api/v1" | jq -e '.service == "gimme-context-api"' >/dev/null

# The development identity boundary treats this principal header as login.
incident="$({ curl --fail --silent \
  -H 'Content-Type: application/json' \
  -H 'X-Principal-ID: minikube-user' \
  -d '{"title":"Minikube smoke incident","description":"Deployment validation","severity":"SEV-3","scope":["minikube"]}' \
  "$BASE_URL/api/v1/workspaces/minikube-smoke/incidents"; })"
incident_id="$(printf '%s' "$incident" | jq -er '.id')"

post="$({ curl --fail --silent \
  -H 'Content-Type: application/json' \
  -H 'X-Principal-ID: minikube-user' \
  -d '{"blocks":[{"type":"markdown","payload":{"text":"Minikube deployment is usable."}}]}' \
  "$BASE_URL/api/v1/workspaces/minikube-smoke/incidents/$incident_id/posts"; })"

printf '%s' "$post" | jq -e '.authorId == "minikube-user" and .blocks[0].payload.text == "Minikube deployment is usable."' >/dev/null
curl --fail --silent "$BASE_URL/api/v1/workspaces/minikube-smoke/incidents/$incident_id/posts" \
  | jq -e '.items | length == 1' >/dev/null

echo "Smoke test passed: web UI, API proxy, development login, incident creation, and posting."
