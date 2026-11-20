#!/usr/bin/env bash
set -euo pipefail

if [[ ! -f .env ]]; then
  echo "Missing .env. Copy .env.example to .env and set local development values." >&2
  exit 1
fi

docker compose up --build -d
docker compose ps
echo "Health endpoint: http://localhost:8080/health"
