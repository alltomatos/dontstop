# Deploy — Docker Swarm (P0-α)

Pré-requisitos: Swarm ativo (3 managers + workers), label `tier=app` nos nós de aplicação e imagens publicadas.
**Kubernetes não é suportado.**

## 1. Secrets

Crie os secrets fora do repositório; nunca use `environment` para credenciais.

```bash
printf '%s' 'postgres://user:pass@pgbouncer:6432/dontstop?sslmode=require' | docker secret create database_url -
printf '%s' 'redis://redis:6379/0' | docker secret create redis_url -
printf '%s' '<base64-32-byte-master-key>' | docker secret create master_key -
printf '%s' 'nats://nats:4222' | docker secret create nats_url -
```

O gateway lê `DATABASE_URL_FILE`, `REDIS_URL_FILE`, `MASTER_KEY_FILE` e `NATS_URL_FILE`.

## 2. Labels

```bash
docker node update --label-add tier=app <node>
```

## 3. Deploy

```bash
export GATEWAY_IMAGE=registry.example/dontstop-gateway:<tag>
export GATEWAY_HOST=gateway.dontstop.corp
docker stack config -c deploy/stack.yml
docker stack deploy -c deploy/stack.yml dontstop
```

Serviços: `traefik` (edge) e `gateway` (3 réplicas, `start-first`, rollback, `stop-grace-period 90s`).

## 4. Operação

```bash
docker service ls
docker service ps dontstop_gateway
docker service logs -f dontstop_gateway
docker service scale dontstop_gateway=6
docker service update --image registry.example/dontstop-gateway:<new-tag> dontstop_gateway
docker service rollback dontstop_gateway
```

## 5. Verificação

```bash
curl -fsS https://gateway.dontstop.corp/healthz
curl -fsS https://gateway.dontstop.corp/readyz
curl -fsS https://gateway.dontstop.corp/v1/models -H "Authorization: Bearer $DONTSTOP_API_KEY"
```

Para SSE, use `curl -N` e confirme `data: ...` + `[DONE]` sem buffering.

## 6. Observações

- Postgres/Redis/NATS/Qdrant devem rodar em nós `tier=data` ou como serviços gerenciados; não estão neste stack.
- Traefik faz TLS no entrypoint `websecure`; em produção configure certificado/ACME conforme o ambiente.
- O healthcheck do gateway usa `/app/gateway healthcheck` → `/readyz`.
- Não use `docker-compose` em produção.
