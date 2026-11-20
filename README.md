# MuchTodo API — Go, MongoDB, Docker & Kubernetes

A learning/portfolio project demonstrating a Go REST API backed by MongoDB, containerized with Docker and deployable locally through Docker Compose or Kubernetes (Kind).

## Features
- Create, list, retrieve, update and delete todo items with MongoDB persistence
- JSON validation, bounded request bodies and MongoDB ObjectID validation
- `/health` process health and `/ready` database readiness endpoints
- Request-scoped database timeouts and graceful HTTP shutdown
- Multi-stage non-root distroless Docker image
- Docker Compose and Kind deployment configurations
- Automated API tests using an in-memory test store

## API endpoints
| Method | Endpoint | Purpose |
|---|---|---|
| GET | `/health` | Process health |
| GET | `/ready` | Database readiness |
| POST | `/todos` | Create todo |
| GET | `/todos` | List todos |
| GET | `/todos/{id}` | Retrieve todo |
| PATCH | `/todos/{id}` | Update title and/or completion |
| DELETE | `/todos/{id}` | Delete todo |

Example:
```bash
curl -X POST http://localhost:8080/todos -H 'Content-Type: application/json' -d '{"title":"Document the API"}'
```

## Run with Docker Compose
Copy `.env.example` to `.env`, replace every placeholder, then build and start:
```bash
./scripts/docker-build.sh
./scripts/docker-run.sh
curl --fail http://localhost:8080/health
curl --fail http://localhost:8080/ready
```

## Deploy to Kind
Copy `.env.k8s.mongodb.example` to `.env.k8s.mongodb` and `.env.k8s.backend.example` to `.env.k8s.backend`. Set matching MongoDB credentials and a backend `MONGO_URI` using the in-cluster service hostname `mongodb`. Then run:
```bash
./scripts/k8s-deploy.sh
kubectl get pods,services,pvc -n muchtodo
curl --fail http://localhost:30080/ready
```

Do not commit populated env files, real credentials, generated Secret manifests or kubeconfig files.

## Run tests
```bash
cd application-code
go test ./...
```

Tests cover request validation and create/list/get/update/delete behaviour using an in-memory store. A live MongoDB integration test remains a useful next enhancement.

## Security and limitations
- `/health` indicates process liveness; `/ready` checks database readiness and is intended for Kubernetes readiness probes.
- The Compose API binds to loopback and MongoDB is not published on the host.
- The runtime image uses a non-root user; Kubernetes drops capabilities, disables privilege escalation and uses a read-only root filesystem.
- MongoDB is a single instance in this local lab, not a highly available production topology.
- No authentication/authorization, pagination, rate limiting, tracing or metrics endpoint is implemented yet. Do not expose this API publicly.
- Prometheus/Grafana, Loki, Docker Swarm and Traefik are not implemented in this repository; demonstrate them separately.