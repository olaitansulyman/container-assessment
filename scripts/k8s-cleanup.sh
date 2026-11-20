#!/usr/bin/env bash
set -euo pipefail

CLUSTER_NAME="muchtodo"
echo "This deletes the muchtodo namespace and the local Kind cluster '$CLUSTER_NAME'."
read -r -p "Type DELETE to continue: " confirmation
if [[ "$confirmation" != "DELETE" ]]; then
  echo "Cleanup cancelled."
  exit 0
fi

kubectl delete namespace muchtodo --ignore-not-found=true
kind delete cluster --name "$CLUSTER_NAME"
