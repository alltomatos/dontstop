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

## 6. Stack standalone (sem Traefik)

Para um host unico ou laboratorio, use `deploy/stack-standalone.yml`. Ele nao sobe Traefik, nao tem labels de roteamento e mantem **1 replica** do gateway publicada diretamente em `:8080`.

```bash
export GATEWAY_IMAGE=registry.example/dontstop-gateway:<tag>
docker stack config -c deploy/stack-standalone.yml
docker stack deploy -c deploy/stack-standalone.yml dontstop-standalone
curl -fsS http://127.0.0.1:8080/healthz
```

O `restart_policy` usa `condition: any`, `delay: 0s` e tentativas ilimitadas; se o container cair, o Swarm cria uma nova task imediatamente. O `update_config` usa `start-first` e rollback em falha.

> Nao escale essa stack para mais de 1 replica: com `ports` publicadas, o Swarm passa a distribuir via routing mesh, o que caracteriza balanceamento. Para HA com mais replicas, use `deploy/stack.yml` com Traefik.

## 7. Observações

- Postgres/Redis/NATS/Qdrant devem rodar em nós `tier=data` ou como serviços gerenciados; não estão neste stack.
- Traefik faz TLS no entrypoint `websecure`; em produção configure certificado/ACME conforme o ambiente.
- O healthcheck do gateway usa `/app/gateway healthcheck` → `/readyz`.
- Não use `docker-compose` em produção.

## 8. AgentRouter relay (ADR-025)

O AgentRouter exige o SDK Python `anthropic` síncrono para passar o WAF de fingerprint. O stack roda
um sidecar **vendorizado** (`vendor/agentrouter-relay`, MIT) na rede interna `app`, **sem porta
publicada** e sem segredo em `environment`; a chave é repassada por requisição via `x-api-key` pelo
gateway (cifrada em repouso no Postgres).

```bash
docker build -t registry.example/dontstop-agentrouter-relay:<tag> vendor/agentrouter-relay
docker push registry.example/dontstop-agentrouter-relay:<tag>
export AGENTROUTER_RELAY_IMAGE=registry.example/dontstop-agentrouter-relay:<tag>
```

O gateway usa `AGENTROUTER_RELAY_URL=http://agentrouter-relay:7187` (já no `stack.yml`). Teste local:

```bash
docker run --rm -p 7187:7187 registry.example/dontstop-agentrouter-relay:<tag>
curl -fsS http://127.0.0.1:7187/health
```
