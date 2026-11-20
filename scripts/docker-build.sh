#!/usr/bin/env bash
set -euo pipefail

IMAGE_NAME="${IMAGE_NAME:-muchtodo-api:local}"
PLATFORM="${PLATFORM:-}"

if [[ -n "$PLATFORM" ]]; then
  docker buildx build --platform "$PLATFORM" --load -t "$IMAGE_NAME" .
else
  docker build -t "$IMAGE_NAME" .
fi

echo "Built image: $IMAGE_NAME"
