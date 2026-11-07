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

## Provider credentials

Provider credentials are encrypted with AES-256-GCM. `MASTER_KEY` must be a base64-encoded 32-byte key.

```bash
export MASTER_KEY='<base64-32-byte-key>'
printf '%s' 'sk-provider-secret' | go run ./cmd/encrypt-secret
```

Store the base64 ciphertext in `provider_accounts.credential_cipher` and the provider endpoint in `provider_accounts.base_url`. The gateway never logs the plaintext credential.

## Chat completions

`POST /v1/chat/completions` requires a Bearer key with `chat:write`.

```bash
curl -s http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer $DONTSTOP_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"model":"gpt-4.1","messages":[{"role":"user","content":"Olá"}],"stream":false}'
```

Use `"stream": true` for SSE; non-streaming requests accept `Idempotency-Key`.

## Exact cache

Non-streaming responses are cached in Redis for 1 h. The response header `X-Donstop-Cache` reports `hit`, `miss` or `bypass`. Streaming requests bypass the exact cache. If Redis is unavailable, the gateway bypasses the cache without failing the request.

## Usage and cost

Non-streaming and streaming requests record usage in `requests`, `usage_daily` and `usage_outbox`. Cost uses `provider_models.input_price_per_million` and `output_price_per_million`. With `NATS_URL` set, a worker publishes outbox events to JetStream with the outbox `event_id` as `Msg-Id`; failed publishes are retried from Postgres.

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
