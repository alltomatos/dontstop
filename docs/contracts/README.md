# Contracts — OpenAPI

Contrato público P0-α: `openapi-v1.yaml`.

## Validar

```bash
npx --yes @redocly/cli@latest lint docs/contracts/openapi-v1.yaml
```

## Breaking changes

```bash
git show origin/main:docs/contracts/openapi-v1.yaml > /tmp/base.yaml
go run github.com/oasdiff/oasdiff@latest breaking /tmp/base.yaml docs/contracts/openapi-v1.yaml
```

## Gerar tipos TypeScript

```bash
npx --yes openapi-typescript@latest docs/contracts/openapi-v1.yaml -o /tmp/openapi.d.ts
```

## Teste de contrato Go

```bash
cd gateway
go test ./internal/api -run TestOpenAPIContract -v
```

O CI `contract.yml` executa lint, breaking-change, geração de tipos e o teste de contrato em PRs que alterem o contrato ou `internal/api`.
