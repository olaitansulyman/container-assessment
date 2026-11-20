#!/usr/bin/env bash
set -euo pipefail

CLUSTER_NAME="muchtodo"
IMAGE_NAME="muchtodo-api:local"

for file in .env.k8s.mongodb .env.k8s.backend; do
  if [[ ! -f "$file" ]]; then
    echo "Missing $file. Copy the matching .example file and set local values." >&2
    exit 1
  fi
done

if ! kind get clusters | grep -qx "$CLUSTER_NAME"; then
  cat <<'EOF' | kind create cluster --name "$CLUSTER_NAME" --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  extraPortMappings:
  - containerPort: 30080
    hostPort: 30080
    protocol: TCP
EOF
fi

kubectl config use-context "kind-$CLUSTER_NAME"

docker build -t "$IMAGE_NAME" .
kind load docker-image "$IMAGE_NAME" --name "$CLUSTER_NAME"

kubectl apply -f kubernetes/namespace.yaml

kubectl -n muchtodo create secret generic mongodb-secret \
  --from-env-file=.env.k8s.mongodb \
  --dry-run=client -o yaml | kubectl apply -f -

kubectl -n muchtodo create secret generic backend-secret \
  --from-env-file=.env.k8s.backend \
  --dry-run=client -o yaml | kubectl apply -f -

kubectl apply -f kubernetes/mongodb/
kubectl apply -f kubernetes/backend/

kubectl -n muchtodo rollout status deployment/mongodb --timeout=180s
kubectl -n muchtodo rollout status deployment/backend --timeout=180s

echo "Deployment complete. Test: curl http://localhost:30080/health"
