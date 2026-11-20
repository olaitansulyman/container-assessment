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
1. Copy `.env.example` to `.env` and replace every placeholder.
2. Build and start using `./scripts/docker-build.sh` and `./scripts/docker-run.sh`.
3. Verify with `curl --fail http://localhost:8080/health` and `curl --fail http://localhost:8080/ready`.
4. Try the CRUD endpoints above. Stop with `docker compose down`.

## Deploy to Kind
Copy `.env.k8s.mongodb.example` to `.env.k8s.mongodb` and `.env.k8s.backend.example` to `.env.k8s.backend`. Set matching MongoDB credentials and a backend `MONGO_URI` that uses the in-cluster service hostname `mongodb`. Then run:
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

Tests cover request validation, create/list/get/update/delete behaviour and invalid identifiers using an in-memory store. A live MongoDB integration test remains a useful next enhancement.

## Architecture and security notes
- The backend connects to MongoDB using the official Go driver and runtime `MONGO_URI` configuration.
- `/health` indicates process liveness; `/ready` checks database readiness and is used for Kubernetes readiness probes.
- The Compose API binds to loopback and MongoDB is not published on the host.
- The runtime image uses a non-root user; Kubernetes drops capabilities, disables privilege escalation and uses a read-only root filesystem.
- MongoDB is a single instance in this local lab, not a highly available production topology.

## Known limitations
- No authentication/authorization, pagination, rate limiting, tracing or metrics endpoint yet. Do not expose this API publicly.
- Kind MongoDB is for local learning, not production resilience.
- Prometheus/Grafana, Loki, Docker Swarm and Traefik are not implemented in this repository; they should be demonstrated separately.