# Página — Observabilidade (backend gerenciado)

> Override do [MASTER](../MASTER.md). **Sem Grafana/Prometheus/Loki/Tempo**: o console consome um backend gerenciado (Datadog ou Azure Monitor) via OpenTelemetry.

## Objetivo

Dar visão de SLOs, latência, erros e saturação sem operar stack de observabilidade própria, com deep links para o backend corporativo.

## Wireframe (desktop ≥ 1024px)

```
┌──────────┬──────────────────────────────────────────────────────────────────────┐
│ Sidebar  │ Título: Observabilidade                  [Abrir no Datadog ↗]         │
│          ├──────────────────────────────────────────────────────────────────────┤
│ Dashboard│ ┌───────────────┬───────────────┬───────────────┬───────────────┐      │
│ Providers│ │ SLO 99,9%     │ p95 28ms      │ Erros 0,4%    │ Saturação 62% │      │
│ Cache    │ │ 99,94% ✓      │ meta 30ms     │ meta 1%       │ 8 réplicas    │      │
│ DLP      │ └───────────────┴───────────────┴───────────────┴───────────────┘      │
│ Custo    │ ┌────────────────────────────────┬──────────────────────────────┐     │
│ Observ.  │ │ Latência p50/p95/p99 (24h)     │ Erros por provider           │     │
│          │ │ [série temporal — lazy]        │ [barras empilhadas]          │     │
│          │ └────────────────────────────────┴──────────────────────────────┘     │
│          │ ┌──────────────────────────────────────────────────────────────┐      │
│          │ │ SLOs e burn rate                                             │      │
│          │ │ Disponibilidade  99,9%    orçamento 43m    burn 0,4x  ● OK   │      │
│          │ │ Latência p95     30ms     orçamento 5%     burn 0,8x  ● OK   │      │
│          │ └──────────────────────────────────────────────────────────────┘      │
│          │ ┌──────────────────────────────────────────────────────────────┐      │
│          │ │ Alertas ativos                                               │      │
│          │ │ ● Anthropic 429 elevado (12 min)  [ver trace] [silenciar]    │      │
│          │ └──────────────────────────────────────────────────────────────┘      │
└──────────┴──────────────────────────────────────────────────────────────────────┘
```

## Componentes

1. **SLO cards** com valor atual, meta, orçamento e burn rate.
2. **Gráficos** de latência e erros por provider; carregados sob demanda.
3. **Tabela de SLOs** com janela, meta, consumo e status.
4. **Alertas ativos** com severidade, desde quando e deep link.
5. **Deep links** para trace/log no backend gerenciado (Datadog/Azure Monitor).
6. **Configuração** de amostragem e retenção (sem PII).

## Dados e APIs

- `GET /admin/v1/observability/slo`
- `GET /admin/v1/observability/latency?range=24h`
- `GET /admin/v1/observability/errors?group=provider`
- `GET /admin/v1/observability/alerts`
- `GET /admin/v1/observability/links` (URLs assinadas para o backend)

## Estados

- **Loading:** skeleton nos cards e gráficos.
- **Vazio:** “Sem dados no período”.
- **Erro:** “Backend de observabilidade indisponível” + retry; não bloquear o restante do console.
- **Degradado:** se o exporter OTel estiver off-line, mostrar último snapshot e banner.

## Lazy loading

- Gráficos só ao entrar na aba; dados agregados no servidor.
- Deep links gerados sob demanda.
- Consultas cacheadas por 30–60 s; refresh manual.

## Segurança e a11y

- Sem PII em labels/traces; `tenant_id` não é label de alta cardinalidade.
- Gráficos com tabela alternativa e resumo textual.
- Alertas com ícone + texto; foco e teclado.
