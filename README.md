# go-shorter

A production-ready URL shortener built with Go (v1.24), PostgreSQL, and Redis following Clean Architecture. Includes high‑performance redirect edge, REST API with JWT auth, OpenAPI spec, Helm chart for Kubernetes, and a static Next.js UI.

- Owner: Kai
- Email: nhatnghiatyper@gmail.com
- Website: https://kkloudtarus.net

## Features
- Clean Architecture with clear layering (`internal/domain|usecase|repository|service|delivery|infra`)
- Binaries
  - `shortlink-api`: CRUD links, auth (JWT), stats, health/metrics
  - `redirector`: ultra-thin edge (GET /{slug}) with Redis cache-first, PG fallback, async-friendly design, OG bot rendering
  - `click-collector` (optional): future event aggregation
- Database schema with migrations (PostgreSQL 16+)
- Redis cache for slug → target_url and optional meta
- Slug providers: Base62 (from PG sequence), pad min length 4
- Rate limit (token bucket) for write-path
- OpenAPI 3.1 spec (`api/openapi.yaml`)
- Observability: Prometheus metrics and optional OTLP tracing
- Helm chart (`deploy/helm/shortlink`) with API, redirector, UI deployments and seed Job
- Static UI (Next.js + Tailwind, exported and served via nginx)

## Architecture
- `internal/domain`: entities, value objects
- `internal/repository`: interfaces and PG implementations
- `internal/service`: auth (JWT, bcrypt), slug, rate limit
- `internal/delivery/http`: chi handlers, middleware, validation
- `internal/infra`: postgres (pgx), redis, migrations
- `pkg/shared`: config, logger, telemetry

## Quickstart (Local)
Prereqs: Go 1.24, Docker, docker-compose, Node (for UI dev optional)

1) Start dependencies and services
```bash
docker compose up -d postgres redis api redirector ui
```

2) Apply migrations (if not applied yet)
```bash
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/shortlink?sslmode=disable"
psql "$DATABASE_URL" -f migrations/0001_init.up.sql
psql "$DATABASE_URL" -f migrations/0002_redirect_perf_indexes.up.sql
```

3) Seed admin
```bash
go run ./cmd/shortlink-api seed \
  --admin-email admin@example.com \
  --password Admin@123 \
  --tenant default
```

4) Verify
- API: http://localhost:8080/healthz
- Redirector: http://localhost:8082/healthz
- UI: http://localhost:3000/app/

## Usage
- Login to get JWT
```bash
curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@example.com","password":"Admin@123"}'
```
- Create a link
```bash
TOKEN=<jwt>
curl -s -X POST http://localhost:8080/api/v1/links \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"target_url":"https://golang.org","custom_slug":"go"}'
```
- Open short URL (redirect)
```bash
curl -I http://localhost:8082/go
```

## UI (Static)
- Next.js configured with `output: export` and `basePath: /app`
- Served via nginx container at http://localhost:3000/app/
- Env: `NEXT_PUBLIC_API_BASE` (default http://localhost:8080), `NEXT_PUBLIC_REDIRECT_BASE` (default http://localhost:8082)

## Helm (Kubernetes)
```bash
helm upgrade --install shortlink ./deploy/helm/shortlink \
  --set admin.email=admin@example.com \
  --set admin.tenant=default \
  --set seedJob.enabled=true
```
- Chart includes Deployments (api, redirector, ui), Services, Ingress, Secret for admin password, and a seed Job hook.

## Configuration
- Config via env (preferred) or YAML
- Env examples:
  - `APP_DB_DSN=postgres://...`
  - `APP_REDIS_ADDR=redis:6379`
  - `APP_JWT_SECRET=changeme`
  - `APP_SERVER_PORT=8080` (via YAML mapping server.port)
- Example YAML: `config/config.example.yaml`

## Observability
- Prometheus:
  - API: `GET /metrics`
  - Redirector: `GET /metrics`
- Tracing (optional): set OTLP endpoint envs in `telemetry` section

## Development
- Run API dev: `make dev`
- Run redirector dev: `make run`
- Build binaries: `make build`
- Tests: `make test`

## Testing
- Unit tests: `go test ./...`
- Integration (manual via compose): CRUD with API, redirect hit/miss
- E2E: compose up (postgres, redis, api, redirector, ui) and test via UI

## Security Notes
- Validate `target_url` schemes (http/https)
- JWT (HS256) with refresh (extend as needed)
- Password hashing with bcrypt
- Rate limit write-path per tenant

## Roadmap
- Background OG crawler and Redis meta cache
- Click collector and aggregations
- Swagger UI hosting and more usecases
- Multi-tenant domains and custom rules

## Contributing
Issues and PRs are welcome. Please run `go fmt` and `go test ./...` before submitting.

## License
MIT (consider adding a LICENSE file).

## Contact
- Owner: Kai
- Email: nhatnghiatyper@gmail.com
- Website: https://kkloudtarus.net
