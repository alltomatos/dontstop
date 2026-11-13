# DEVELOPER-ROADMAP — DonStop IA

> Roadmap de execução para o `/developer`. IDs de Epic (`E##`) e de slice (`E##.#`) são estáveis e não podem ser reutilizados.
> Decisões: [ADR-018](docs/05-ADRS.md), [ADD](docs/architecture/ADD.md), [PRD](docs/01-PRD.md), [DER](docs/02-DER.md) e [contrato OpenAPI](docs/contracts/openapi-v1.yaml).

## Epics

- [**[E1] P0-α — Gateway vertical slice (chat + models)**](https://github.com/alltomatos/dontstop/issues/1) - `in_progress`
- **[E2] Isolamento de proxy por provedor (ADR-024)** - `in_progress`

## E1 — P0-α Gateway vertical slice

**Objetivo:** entregar o primeiro fluxo ponta a ponta do gateway: autenticação por API key, catálogo de modelos, `/v1/chat/completions` com JSON/SSE, failover multi-conta, cache exato Redis e usage/custo no Postgres, com deploy Swarm e observabilidade mínima.

**Escopo:** OpenAI e Anthropic, single-tenant, 50 RPS sustentado/150 burst, 500 SSE, 3 réplicas, 1 região. DLP, cache semântico, Qdrant, proxies Webshare e multi-tenancy ficam fora do P0-α.

**Critérios de sucesso:**
- [ ] `/v1/models` e `/v1/chat/completions` (JSON + SSE) passam no contrato e no SDK OpenAI.
- [ ] API key inválida, escopo/modelo não permitido e tenant divergente falham fechado.
- [ ] Failover de conta/provider ocorre em ≤ 2 s antes do primeiro token.
- [ ] Cache exato tem hit/miss mensurável e isolamento por tenant.
- [ ] Usage/custo fecha por requisição e reconcilia com a tabela de preços em ≤ 1%.
- [ ] Carga de 2× o envelope (100 RPS/300 burst) com p95 de overhead ≤ 30 ms e erros ≤ 0,5%.
- [ ] 3 réplicas em Swarm com `/healthz`, `/readyz`, traces e dashboards de golden signals.
- [ ] OpenAPI validado no CI e sem divergência de contrato.

## Slices

| ID | Slice | Estado | Issue | Blocked by |
|---|---|---|---|---|
| E1.1 | Bootstrap Go, CI e Dockerfile | `todo` | [#2](https://github.com/alltomatos/dontstop/issues/2) | — |
| E1.2 | Schema Postgres, migrações e tenant bootstrap | `todo` | [#3](https://github.com/alltomatos/dontstop/issues/3) | E1.1 |
| E1.3 | Auth por API key + tenant/escopos | `todo` | [#4](https://github.com/alltomatos/dontstop/issues/4) | E1.2 |
| E1.4 | Registry de providers, adapters OpenAI/Anthropic e `/v1/models` | `todo` | [#5](https://github.com/alltomatos/dontstop/issues/5) | E1.2 |
| E1.5 | `/v1/chat/completions` JSON + SSE | `done` | [#6](https://github.com/alltomatos/dontstop/issues/6) | E1.3, E1.4, E1.12 |
| E1.6 | Roteamento multi-conta, failover e circuit breaker | `done` | [#7](https://github.com/alltomatos/dontstop/issues/7) | E1.4 |
| E1.7 | Cache exato Redis e invalidação | `done` | [#8](https://github.com/alltomatos/dontstop/issues/8) | E1.3, E1.5 |
| E1.8 | Usage/custo com outbox e NATS | `done` | [#9](https://github.com/alltomatos/dontstop/issues/9) | E1.5 |
| E1.9 | OTel, health/readiness e carga 2× | `todo` | [#10](https://github.com/alltomatos/dontstop/issues/10) | E1.5, E1.7, E1.8 |
| E1.10 | OpenAPI lint, geração de tipos e contract tests | `todo` | [#11](https://github.com/alltomatos/dontstop/issues/11) | E1.5 |
| E1.11 | Swarm/Traefik com 3 réplicas | `todo` | [#12](https://github.com/alltomatos/dontstop/issues/12) | E1.1, E1.9 |
| E1.12 | Credenciais de provider, base URL e resolução segura | `done` | [#19](https://github.com/alltomatos/dontstop/issues/19) | E1.1, E1.2 |

## Definição de pronto do P0-α

- Código real, sem pseudocódigo, com DI explícita no bootstrap.
- Testes unitários, integração (Postgres/Redis), contrato e segurança negativa.
- Sem PII/segredo em logs; redaction testada.
- Métricas/traces do caminho quente; SLOs no backend gerenciado.
- Migrações idempotentes e índices revisados.
- Documentação e contrato atualizados.
- `go test ./...`, `go vet`, `golangci-lint`, `govulncheck`, lint/test/build do admin verdes.

## E2 — Isolamento de proxy por provedor (ADR-024)

**Objetivo:** isolar o IP de cada conta do provedor (1 conta/IP por provedor), reutilizar IPs entre provedores diferentes, introduzir o modo `auto` gerenciado pelo Webshare e rotacionar com poison persistente + re-sync, sem banimento cruzado.

**Escopo:** schema/DB, data plane Go (modo efetivo, alocação, classificação de falha, poison, re-sync), Admin API, console (detalhe do provedor e `/proxies`), testes de segurança e observabilidade. Fora de escopo: túneis e proxies fora do Webshare/manual.

**Critérios de sucesso:**
- [ ] Duas contas do mesmo provedor nunca compartilham IP; provedores diferentes reutilizam.
- [ ] Modo `auto` distribui IPs distintos por conta e rotaciona na falha.
- [ ] Falha de IP envenena só o provedor afetado e dispara re-sync do Webshare.
- [ ] Rotação transparente antes do primeiro chunk; sem fallback direto silencioso.
- [ ] Credenciais nunca em resposta/log; isolamento por tenant comprovado.

## E2 — Slices

| ID | Slice | Estado | Issue | Blocked by |
|---|---|---|---|---|
| E2.1 | Schema de isolamento (0020), `proxy_poison`, ban columns | `done` | [#53](https://github.com/alltomatos/dontstop/issues/53) | — |
| E2.2 | Modo efetivo e alocação 1 conta/IP por provedor | `done` | [#54](https://github.com/alltomatos/dontstop/issues/54) | E2.1 |
| E2.3 | Classificação de falha e poison persistente | `done` | [#55](https://github.com/alltomatos/dontstop/issues/55) | E2.1 |
| E2.4 | Re-sync Webshare automático (debounce) | `done` | [#56](https://github.com/alltomatos/dontstop/issues/56) | E2.3 |
| E2.5 | Admin API: `proxy_auto` e modos `inherit/off/manual/auto` | `done` | [#57](https://github.com/alltomatos/dontstop/issues/57) | E2.2, E2.3 |
| E2.6 | Console: proxy por conta e modo auto | `done` | [#58](https://github.com/alltomatos/dontstop/issues/58) | E2.5 |
| E2.7 | Console: rede de proxy 1:1 + poison | `done` | [#59](https://github.com/alltomatos/dontstop/issues/59) | E2.5 |
| E2.8 | Segurança e testes negativos | `done` | [#60](https://github.com/alltomatos/dontstop/issues/60) | E2.2, E2.3, E2.4 |
| E2.9 | Observabilidade, guia e runbook | `done` | [#61](https://github.com/alltomatos/dontstop/issues/61) | E2.4 |

> E2.1 implementado no working tree (migração `0020`, `provider_id` + unique por provedor, dedup, `proxy_poison`, `Candidates` provider-aware). Pendente de commit/PR por decisão explícita.
