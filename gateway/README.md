# DonStop IA Gateway

Go data plane for the DonStop IA gateway. The current scaffold exposes liveness and readiness probes and is ready for the P0-α slices.

## Run locally

```bash
cd gateway
go run ./cmd/gateway
```

Default port: `8080`. Override with `PORT`.

```bash
PORT=9090 go run ./cmd/gateway
curl http://localhost:9090/healthz
curl http://localhost:9090/readyz
```

## Database

The gateway uses `DATABASE_URL` and `REDIS_URL` when present and exposes both checks in `/readyz`.

```bash
export DATABASE_URL='postgres://postgres:postgres@localhost:5432/dontstop?sslmode=disable'
export REDIS_URL='redis://localhost:6379/0'
go run ./cmd/migrate
go run ./cmd/gateway
```

Migrations are embedded in `migrations/` and are applied transactionally with an advisory lock. They are idempotent on re-run.

## Validate

```bash
go test -race ./...
go vet ./...
go build ./...
```

## Build the image

```bash
docker build -t dontstop-gateway:dev .
```

The final image runs as `nonroot` and contains only the static binary.
