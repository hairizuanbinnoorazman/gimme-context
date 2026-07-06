#!/usr/bin/env bash
set -euo pipefail

PROFILE="${MINIKUBE_PROFILE:-minikube}"
NAMESPACE="${NAMESPACE:-gimme-context}"
RELEASE="${RELEASE:-gimme-context}"
IMAGE_TAG="${MINIKUBE_IMAGE_TAG:-minikube-$(date +%Y%m%d%H%M%S)}"
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

for command in minikube docker helm; do
  if ! command -v "$command" >/dev/null 2>&1; then
    echo "missing required command: $command" >&2
    exit 1
  fi
done

if ! minikube status --profile "$PROFILE" >/dev/null 2>&1; then
  minikube start --profile "$PROFILE"
fi

docker build --target api -t "gimme-context:$IMAGE_TAG" "$ROOT"
docker build --target worker -t "gimme-context-worker:$IMAGE_TAG" "$ROOT"
docker build -t "gimme-context-web:$IMAGE_TAG" "$ROOT/web"

minikube image load --profile "$PROFILE" "gimme-context:$IMAGE_TAG"
minikube image load --profile "$PROFILE" "gimme-context-worker:$IMAGE_TAG"
minikube image load --profile "$PROFILE" "gimme-context-web:$IMAGE_TAG"

helm upgrade --install "$RELEASE" "$ROOT/deploy/helm/gimme-context" \
  --namespace "$NAMESPACE" \
  --create-namespace \
  --values "$ROOT/deploy/helm/gimme-context/values-minikube.yaml" \
  --set "image.tag=$IMAGE_TAG" \
  --set "workerImage.tag=$IMAGE_TAG" \
  --set "webImage.tag=$IMAGE_TAG" \
  --wait \
  --timeout 5m

echo "Deployment is ready."
echo "Run: minikube service --profile $PROFILE --namespace $NAMESPACE ${RELEASE}-web"
echo "Validate it with: make minikube-smoke"
