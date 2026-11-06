# QA — P0-α E1.7

> Gate do cache exato Redis.
> Data: 2026-11-06. Issue: #8.

## Escopo testado

- Chave estável `sha256` com tenant, escopo, modelo, mensagens, parâmetros e prompt version.
- Prefixo `t:{tenant}:cache:exact:` e isolamento por tenant.
- Roundtrip de resposta e TTL.
- Invalidação por modelo, provider, prompt version e tenant.
- Hit/miss/`X-Donstop-Cache` no chat não streaming.
- Bypass automático quando Redis falha.
- SSE marcado como `X-Donstop-Cache: bypass`.

## Fora de escopo

- Cache semântico/Qdrant (P1).
- L1 in-process (S0.21).
- Replay de SSE a partir do cache.
- Cache negativo de erros de provider.

## Casos executados

| ID | Caso | Resultado |
|---|---|---|
| QA-56 | Mesma entrada gera a mesma chave | PASS |
| QA-57 | Tenant/model/params/prompt/mensagens alteram a chave | PASS |
| QA-58 | Roundtrip de resposta no Redis | PASS |
| QA-59 | Tenant B não lê entrada do tenant A | PASS |
| QA-60 | Invalidação por provider remove só o alvo | PASS |
| QA-61 | Invalidação por tenant limpa o restante | PASS |
| QA-62 | Hit retorna resposta cacheada e `hit` | PASS |
| QA-63 | Miss chama provider, grava e retorna `miss` | PASS |
| QA-64 | Header `X-Donstop-Cache` é exposto | PASS |
| QA-65 | Redis indisponível faz bypass sem quebrar o chat | PASS |
| QA-66 | Streaming responde `X-Donstop-Cache: bypass` | PASS |
| QA-67 | `go test -race -p 1 ./...` com Postgres/Redis | PASS |
| QA-68 | `golangci-lint`, `go vet`, `go build` | PASS |
| QA-69 | `govulncheck` com Go 1.26.10 | PASS, 0 vulnerabilidades |

## Auditoria de segurança

- Chaves sempre prefixadas por tenant; teste negativo de leitura cruzada.
- TTL padrão de 1 h; índices de invalidação expiram junto.
- Cache armazena apenas resposta/usage; não armazena prompt bruto nem credencial.
- Falha do Redis é fail-open para disponibilidade, sem afetar autenticação/roteamento.
- Invalidação não expõe payload em logs.

## Riscos residuais

- Streaming não é cacheado (bypass explícito).
- Cache negativo existe na estrutura de entrada, mas não é ativado para erros de provider para não mascarar failover.
- Invalidação ainda é interna (sem endpoint Admin).
- L1 e cache semântico ficam para S0.21/P1.
