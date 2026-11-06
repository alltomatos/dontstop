# Página — Dashboard

> Override do [MASTER](../MASTER.md). Foco: visão executiva em < 5 s, densidade alta, lazy loading de gráficos.

## Objetivo

Responder “o gateway está saudável, rápido, barato e seguro?” com KPIs, saúde de providers e incidentes recentes.

## Wireframe (desktop ≥ 1024px)

```
┌──────────┬──────────────────────────────────────────────────────────────────────┐
│ Sidebar  │ Topbar: [busca global ⌘K]        [tenant] [ambiente] [avatar]         │
│          ├──────────────────────────────────────────────────────────────────────┤
│ Dashboard│ Título: Visão geral                     [últimas 24h ▾] [atualizar]   │
│ Providers│ ┌─────────┬─────────┬─────────┬─────────┬─────────┬─────────┐          │
│ Cache    │ │ RPS     │ Streams │ p95     │ Erros   │ Cache   │ Custo   │          │
│ DLP      │ │ 842     │ 3.1k    │ 28 ms   │ 0,4%    │ 41%     │ US$ 312 │          │
│ Custo    │ │ +12%    │ +8%     │ -3 ms   │ -0,1pp  │ +5pp    │ -18%    │          │
│ Observ.  │ └─────────┴─────────┴─────────┴─────────┴─────────┴─────────┘          │
│          │ ┌──────────────────────────────┬──────────────────────────────┐        │
│          │ │ Tráfego e latência (24h)     │ Custo por provider (24h)     │        │
│          │ │ [gráfico lazy — skeleton]    │ [bullet chart / top 5]       │        │
│          │ └──────────────────────────────┴──────────────────────────────┘        │
│          │ ┌──────────────────────────────────────────────────────────────┐       │
│          │ │ Saúde dos providers                                          │       │
│          │ │ Provider     Contas  Status      p95     Erros   Cache      │       │
│          │ │ OpenAI       3/3     ● Saudável  210ms   0,2%    38%        │       │
│          │ │ Anthropic    2/3     ◐ Cooldown 240ms   1,1%    44%        │       │
│          │ └──────────────────────────────────────────────────────────────┘       │
│          │ ┌──────────────────────────────┬──────────────────────────────┐        │
│          │ │ Incidentes recentes          │ Ações rápidas                │        │
│          │ │ 10:42 Anthropic 429 → fallback│ [Testar provider] [Limpar cache]│     │
│          │ └──────────────────────────────┴──────────────────────────────┘        │
└──────────┴──────────────────────────────────────────────────────────────────────┘
```

## Componentes

1. **Filtro de período** (`24h`, `7d`, `30d`, custom) persistido na URL.
2. **KPI cards** com valor, variação e sparkline; números tabulares.
3. **Gráfico de tráfego/latência**: linha dupla (RPS + p95), lazy, skeleton, legenda e tabela alternativa.
4. **Custo por provider**: bullet chart com valor visível e meta.
5. **Tabela de saúde**: provider, contas ativas/total, status (ícone+texto), p95, erros, cache hit.
6. **Incidentes**: últimos 5 eventos de failover/DLP/erro com link para detalhe.
7. **Ações rápidas**: testar provider, invalidar cache, abrir command palette.

## Dados e APIs

- `GET /admin/v1/metrics/overview?range=24h`
- `GET /admin/v1/providers/health`
- `GET /admin/v1/incidents?limit=5`
- `GET /admin/v1/costs/summary?range=24h`

## Estados

- **Loading:** skeleton nos KPIs e gráficos; nunca tela em branco.
- **Vazio:** “Nenhuma requisição no período” + ação “Abrir playground”.
- **Erro:** card com causa, retry e link para observabilidade.
- **Parcial:** se o backend de métricas estiver indisponível, mostrar KPIs de cache/erro locais e banner “métricas externas indisponíveis”.

## Lazy loading

- Gráficos e tabelas abaixo da dobra via `dynamic import`.
- Prefetch do chunk de “Providers” no idle.
- KPIs renderizados no servidor; detalhes no cliente.
- Dados com `stale-while-revalidate` de 15 s; botão “atualizar” força refresh.

## Acessibilidade

- Cada KPI tem `aria-label` com valor e unidade.
- Gráficos com resumo textual (`aria-describedby`) e tabela oculta para leitor de tela.
- Status nunca só por cor: ícone + rótulo.
- Foco no título após navegação; atalhos documentados.
