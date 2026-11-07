# P0-α Load Tests

Pré-requisitos: gateway com Postgres, Redis, MASTER_KEY e provider de teste; k6 instalado.

## JSON a 100 RPS (2× envelope P0-α)

```bash
k6 run \
  -e BASE_URL=http://127.0.0.1:8080 \
  -e API_KEY=sk-ds-e2e000000000001 \
  -e MODEL=gpt-4.1 \
  -e RPS=100 \
  -e DURATION=30s \
  tests/load/chat.js
```

## SSE a 500 streams

```bash
k6 run \
  -e BASE_URL=http://127.0.0.1:8080 \
  -e API_KEY=sk-ds-e2e000000000001 \
  -e MODEL=gpt-4.1 \
  -e VUS=500 \
  -e DURATION=30s \
  tests/load/chat-sse.js
```

## Critérios

- `http_req_failed < 0,5%`.
- JSON p95 < 2 s (inclui tempo do provider fake; overhead do gateway deve ficar ≤ 30 ms).
- SSE sem erros e com `[DONE]` em todas as respostas.
- Comparar com `gateway.requests`, `http.server.request.duration`, cache hit/miss e failover no backend OTel.

> O provider de teste deve responder rápido e de forma determinística; não use provedor real para o gate de overhead.
