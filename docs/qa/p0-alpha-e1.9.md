# QA — P0-α E1.9

> Gate de observabilidade e carga.
> Data: 2026-11-07. Issue: #10.

## Escopo testado

- OTLP/HTTP traces + metrics via `OTEL_EXPORTER_OTLP_*`; noop quando endpoint ausente.
- `otelhttp` para golden signals HTTP.
- Spans `auth.authenticate`, `cache.exact.get`, `routing.select`, `provider.chat`, `provider.stream`, `usage.record`.
- Runtime metrics (`go.goroutine.count`, memória) via OTel runtime.
- Readiness com Postgres/Redis.
- Carga JSON a 100 RPS por 30 s (2× envelope P0-α de 50 RPS).

## Fora de escopo / pendente

- Teste SSE de 500 streams: scripts `tests/load/chat-sse.js` prontos, mas o harness local de provider fake não sustentou 500 conexões; executar em staging antes do GA.
- Backend gerenciado real (Datadog/Azure) não configurado neste ambiente.
- Alertas/SLOs publicados no backend dependem das credenciais do operador.

## Casos executados

| ID | Caso | Resultado |
|---|---|---|
| QA-81 | `telemetry.Setup` sem endpoint é noop | PASS |
| QA-82 | Spans de auth/cache/routing/provider/usage existem no código | PASS |
| QA-83 | `otelhttp` envolve o router | PASS |
| QA-84 | Runtime metrics iniciados com OTLP | PASS |
| QA-85 | Readiness continua `postgres`/`redis` | PASS |
| QA-86 | Carga 100 RPS/30 s | 2.982 req, 0 falhas, p95 152 ms (inclui provider fake) |
| QA-87 | `go test -race -p 1 ./...` com Postgres/Redis | PASS |
| QA-88 | `golangci-lint` v2.15, `go vet`, `go build` | PASS |
| QA-89 | `govulncheck` com Go 1.26.10 | PASS, 0 vulnerabilidades chamadas |

## Auditoria de segurança

- Spans não incluem prompt, resposta, API key ou credencial; apenas modelo/provider/status/tokens.
- Exportador OTLP usa endpoint via ambiente; sem endpoint, nenhum dado sai.
- `otelhttp` não altera autenticação/escopo; é apenas instrumentação.
- Logs continuam sem PII/segredos.

## Riscos residuais

- 500 SSE ainda não validado no ambiente local; gate de staging obrigatório.
- Sem backend gerenciado configurado, dashboards/SLOs não foram materializados.
- Amostragem fixa de 10%; erros podem exigir sampler dedicado.
- Go foi atualizado para 1.26.10 para corrigir vulnerabilidades de OTel/net/http; CI/lint usam v2.15.0.
