# Observabilidade — P0-α

## Sinais e nomes

| Sinal | Fonte | Onde consultar |
|---|---|---|
| Tráfego/latência/erros HTTP | `otelhttp` (`http.server.request.duration`, `http.server.request.body.size`) | Datadog APM / Azure Monitor |
| Auth | span `auth.authenticate` | Traces |
| Cache | span `cache.exact.get`, atributo `cache.hit` | Traces |
| Roteamento/failover | span `routing.select`, atributo do provider | Traces |
| Provider upstream | spans `provider.chat`, `provider.stream` | Traces |
| Usage/custo | span `usage.record` + tabelas `usage_daily`/`requests` | Postgres/console |
| Saturação | runtime OTel (`go.goroutine.count`, memória) + réplicas | Datadog/Azure Monitor |

## SLOs P0-α

| SLO | Meta | Janela |
|---|---|---|
| Disponibilidade | 99,9% | 30 d |
| Sucesso de chat | ≥ 99,5% | 30 d |
| Overhead p95 | ≤ 30 ms | 7 d |
| p95 fim-a-fim (provider fake) | ≤ 800 ms | 7 d |
| Cache hit (quando habilitado) | ≥ 25% | 7 d |

## Consultas de exemplo (Datadog)

```text
avg:http.server.request.duration{service:dontstop-gateway}.as_count()
sum:http.server.request.duration{service:dontstop-gateway,http.status_code:5xx}.as_count()
p95:http.server.request.duration{service:dontstop-gateway}
```

## Consultas de exemplo (Azure Monitor)

```kusto
requests
| where cloud_RoleName == "dontstop-gateway"
| summarize p95 = percentile(duration, 95), failures = countif(success == false) by bin(timestamp, 5m)
```

## Exportação

- OTel Collector recebe OTLP/HTTP e exporta para Datadog ou Azure Monitor.
- **Sem** Prometheus, Grafana, Loki ou Tempo.
- Amostragem de traces: 10%; erros devem ser amostrados integralmente pela política do backend.
- Logs/traces não contêm prompt, resposta, PII ou credenciais.
