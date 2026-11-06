# QA — P0-α E1.6

> Gate do roteamento multi-conta, failover e circuit breaker.
> Data: 2026-11-06. Issue: #7.

## Escopo testado

- Seleção por prioridade e peso; failover entre contas/providers.
- Parada imediata depois do primeiro token.
- Circuit breaker Redis `closed → open → half_open → closed`.
- Cooldown exponencial por auth, rate limit, 5xx e erro desconhecido.
- Exclusão de contas em cooldown e providers desabilitados.
- Isolamento de estado do breaker por tenant.
- Readiness com Postgres e Redis.

## Fora de escopo

- `/v1/chat/completions` e streaming (E1.5).
- Cache, usage/custo, OTel e deploy Swarm.
- Rate limiting de entrada e body limits.

## Casos executados

| ID | Caso | Resultado |
|---|---|---|
| QA-20 | Falha na conta A faz failover para B antes do primeiro token | PASS |
| QA-21 | Falha depois do primeiro token não faz retry | PASS |
| QA-22 | Conta open é ignorada | PASS |
| QA-23 | Breaker abre, vira half-open e fecha após sucesso | PASS |
| QA-24 | 429/5xx/auth são classificados para cooldown | PASS |
| QA-25 | Candidatos excluem conta em cooldown/provider desabilitado | PASS |
| QA-26 | Estado do breaker não cruza tenants | PASS |
| QA-27 | `/readyz` inclui `postgres` e `redis` | PASS |
| QA-28 | `go test -race -p 1 ./...` com Postgres/Redis | PASS |
| QA-29 | `golangci-lint`, `go vet`, `go build` | PASS |
| QA-30 | `govulncheck` com Go 1.26.10 | PASS, 0 vulnerabilidades |

## Auditoria de segurança

- **Isolamento:** chaves Redis usam `t:{tenant}:cb:{account}:...`; teste negativo com o mesmo `account_id` em dois tenants.
- **Feedback de falha:** erros de auth/rate/server não expõem payload do provider e alimentam cooldown.
- **Fail-open controlado:** erro de leitura do breaker não bloqueia tráfego; a indisponibilidade do Redis é tratada como degradação, não como bypass de autenticação.
- **Bounded retries:** máximo de 3 tentativas; jamais após o primeiro token.
- **Sem segredos:** o router carrega `credential_cipher`, mas não descriptografa nem loga credenciais nesta slice.

## Riscos residuais

- Backoff sem jitter; pode gerar sincronização de retries em incidentes grandes.
- Half-open usa um único probe por conta, sem limite global por provider.
- Rate limiting de entrada e body limits continuam pendentes para o slice de produção.
- Chaos automatizado de provider completo será ampliado em E1.9.
