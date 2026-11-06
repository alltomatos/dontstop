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
