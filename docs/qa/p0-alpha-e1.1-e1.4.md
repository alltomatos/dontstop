# QA — P0-α E1.1–E1.4

> Gate de qualidade e segurança antes do merge do stack `#14 → #15 → #16 → #17`.
> Data: 2026-11-06. Escopo: bootstrap, schema/migrações, auth por API key, catálogo e adapters.

## Escopo testado

- `/healthz` e `/readyz`, incluindo check do Postgres.
- Migrações, índices, tenant padrão e reexecução idempotente.
- Hash/verificação Argon2id, Bearer auth, revogação, expiração e escopos.
- `/v1/models` com autenticação, escopo `models:read`, catálogo e isolamento por tenant.
- Tradução OpenAI/Anthropic (request, response, usage e erros).
- Build, Dockerfile não-root e CI.

## Fora de escopo (issues posteriores)

- `/v1/chat/completions` e streaming SSE (E1.5).
- Roteamento multi-conta, failover e circuit breaker (E1.6).
- Cache exato (E1.7), usage/custo (E1.8), OTel/carga (E1.9).
- DLP, cache semântico, Webshare, rate limiting e deploy Swarm.
- Rate limiting, body size limits e auditoria persistida ainda não existem no P0-α atual.

## Casos executados

| ID | Caso | Resultado |
|---|---|---|
| QA-01 | `go test -race ./...` com Postgres 17 | PASS |
| QA-02 | Migrações aplicam e reexecutam sem erro | PASS |
| QA-03 | Tenant padrão semeado | PASS |
| QA-04 | Índices de API key, roteamento e requests existem | PASS |
| QA-05 | `/healthz` = 200; `/readyz` inclui `postgres: ok` | PASS |
| QA-06 | API key válida autentica e carrega tenant/escopos | PASS |
| QA-07 | Header ausente/malformado retorna 401 | PASS |
| QA-08 | Chave revogada/expirada não autentica | PASS |
| QA-09 | Escopo ausente em `/v1/models` retorna 403 | PASS |
| QA-10 | Tenant B não vê modelos do tenant A | PASS |
| QA-11 | Prefixo com payload de injeção não autentica | PASS |
| QA-12 | Catálogo exclui modelo inativo/provider desabilitado | PASS |
| QA-13 | Modelo em cooldown aparece com status `cooldown` | PASS |
| QA-14 | Adapter OpenAI traduz request/response/usage | PASS |
| QA-15 | Adapter Anthropic traduz system/max_tokens/usage | PASS |
| QA-16 | Erro 401 do provider vira `ErrProviderAuth` | PASS |
| QA-17 | `golangci-lint`, `go vet`, `go build` | PASS |
| QA-18 | `govulncheck` com Go 1.26.10 | PASS, 0 vulnerabilidades chamadas |
| QA-19 | Imagem Docker non-root responde `/healthz` | PASS |

## Auditoria de segurança

- **AuthN:** Bearer obrigatório; verificação Argon2id em tempo constante; chave nunca entra em erro/log.
- **AuthZ:** `/v1/models` exige `models:read`; tenant vem do contexto autenticado, não do cliente.
- **SQLi:** queries usam placeholders pgx; teste com prefixo malicioso rejeitado.
- **Tenant isolation:** catálogo e lookup de chave filtram por `tenant_id`; teste negativo com dois tenants.
- **Segredos:** fixtures usam chaves falsas; credenciais de provider não são renderizadas nem logadas.
- **SSRF (residual):** `BaseURL` do adapter é configuração administrativa, não input do chamador; allowlist de upstream entra com o hardening/proxy.
- **Rate limiting (residual):** ainda não implementado; será endereçado no slice de rate limit/produção.
- **Transporte:** TLS/HSTS ficam no Traefik/Swarm (E1.11); HTTP local é apenas desenvolvimento.

## Achados

Nenhum defeito bloqueante ou de segurança explorável encontrado no escopo E1.1–E1.4.

## Próximos gates

- E1.5 deve adicionar validação de body, `Idempotency-Key`, limites de payload e testes de streaming/cancelamento.
- E1.6 deve cobrir failover e cooldown com chaos test.
- E1.9/E1.11 devem validar carga, TLS e headers de segurança no ambiente Swarm.
