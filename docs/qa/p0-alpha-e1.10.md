# QA — P0-α E1.10

> Gate do contrato OpenAPI.
> Data: 2026-11-07. Issue: #11.

## Escopo testado

- OpenAPI 3.1.0 em `docs/contracts/openapi-v1.yaml`.
- Rotas `/models` GET e `/chat/completions` POST.
- Request obrigatório `model` + `messages`.
- Respostas 200 JSON + SSE e erros 401/403/429.
- `bearerAuth` presente.
- Lint Redocly, breaking-change com oasdiff e geração TypeScript.
- Teste de contrato Go estrutural.

## Fora de escopo

- SDK oficial OpenAI executando contra o gateway real; o smoke E2E foi feito em E1.5.
- Publicação de SDK tipado para clientes.
- Mock server do contrato.

## Casos executados

| ID | Caso | Resultado |
|---|---|---|
| QA-90 | Redocly lint sem warnings | PASS |
| QA-91 | oasdiff contra `main` sem breaking changes | PASS |
| QA-92 | openapi-typescript gera arquivo não vazio | PASS (14 KB) |
| QA-93 | Teste Go exige paths, operação, request, responses e bearer | PASS |
| QA-94 | `go test ./internal/api -run TestOpenAPIContract` | PASS |
| QA-95 | Workflow `contract.yml` cobre lint/diff/tipos/teste | PASS (config) |
| QA-96 | `golangci-lint`, `go vet`, `go build` | PASS |
| QA-97 | `govulncheck` | PASS, 0 vulnerabilidades chamadas |

## Riscos residuais

- Não há SDK cliente gerado/versionado no repo.
- Mock server/Prism não está configurado.
- O smoke com SDK oficial depende de ambiente com provider; coberto pelo teste manual de E1.5.
