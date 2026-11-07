# QA — P0-α E1.11

> Gate de deploy Swarm.
> Data: 2026-11-07. Issue: #12.

## Escopo testado

- `deploy/stack.yml` com 3 réplicas do gateway.
- Traefik edge, TLS entrypoint, HTTP/2 e flush de SSE.
- Docker secrets para DATABASE_URL, REDIS_URL, MASTER_KEY e NATS_URL.
- Redes overlay `edge`/`app`/`data` (data encrypted).
- Rolling update `start-first`, rollback, `stop-grace-period 90s`.
- Healthcheck nativo via `/app/gateway healthcheck` → `/readyz`.
- Imagem Docker non-root com healthcheck.

## Fora de escopo / pendente

- Deploy real em cluster Swarm de staging/produção: `docker stack config` validado, mas o rollout de 3 réplicas deve ser executado no ambiente alvo.
- Certificado TLS/ACME real do Traefik.
- Postgres/Redis/NATS/Qdrant gerenciados fora do stack.

## Casos executados

| ID | Caso | Resultado |
|---|---|---|
| QA-98 | `docker stack config -c deploy/stack.yml` | PASS |
| QA-99 | Stack declara 3 réplicas e update start-first/rollback | PASS (config) |
| QA-100 | Secrets montados via arquivos (`_FILE`) | PASS |
| QA-101 | Healthcheck `/app/gateway healthcheck` | PASS |
| QA-102 | Imagem Docker inicia e fica `healthy` | PASS |
| QA-103 | Runbook de deploy/scale/rollback | PASS |
| QA-104 | `go test`, lint, vet, build, govulncheck | PASS |

## Auditoria de segurança

- Nenhum segredo no `stack.yml` ou em `environment`; apenas Docker secrets.
- Execução como `nonroot`; healthcheck sem shell.
- Rede `data` criptografada.
- TLS no entrypoint `websecure`; HTTP redireciona para HTTPS.
- Sem Kubernetes, sem túneis.

## Riscos residuais

- Rollout real e certificado TLS dependem do cluster/ambiente.
- Stateful services não estão no stack; precisam de `tier=data` + backup.
- Autoscaling é manual (`docker service scale`) no P0.
