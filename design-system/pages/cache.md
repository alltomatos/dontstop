# Página — Cache

> Override do [MASTER](../MASTER.md). Foco: economia, hit rate, camadas e invalidação segura.

## Objetivo

Explicar quanto o cache economiza, onde ele acerta/erra e permitir invalidar com segurança por tenant/modelo/versão.

## Wireframe (desktop ≥ 1024px)

```
┌──────────┬──────────────────────────────────────────────────────────────────────┐
│ Sidebar  │ Título: Cache                                    [Invalidar ▾]       │
│          ├──────────────────────────────────────────────────────────────────────┤
│ Dashboard│ ┌───────────────┬───────────────┬───────────────┬───────────────┐      │
│ Providers│ │ Hit total     │ Exact hit     │ Semantic hit  │ Economia 24h  │      │
│ Cache    │ │ 41,2%         │ 26,4%         │ 14,8%         │ US$ 1.284     │      │
│ DLP      │ │ +5,1pp        │ +2,0pp        │ +3,1pp        │ -32% custo    │      │
│ Custo    │ └───────────────┴───────────────┴───────────────┴───────────────┘      │
│ Observ.  │ ┌────────────────────────────────┬──────────────────────────────┐     │
│          │ │ Hit rate por dia (lazy)        │ Top modelos por economia     │     │
│          │ │ [linha exact/semantic]         │ [tabela compacta]            │     │
│          │ └────────────────────────────────┴──────────────────────────────┘     │
│          │ ┌──────────────────────────────────────────────────────────────┐      │
│          │ │ Camadas e configuração                                       │      │
│          │ │ L1 in-process: 10s   L2 Redis: 1h   Semântico: 24h (0,95)   │      │
│          │ │ [Editar TTL] [Limpar L1] [Reindexar semântico]              │      │
│          │ └──────────────────────────────────────────────────────────────┘      │
│          │ ┌──────────────────────────────────────────────────────────────┐      │
│          │ │ Eventos recentes                                             │      │
│          │ │ 10:44 exact hit  gpt-4.1  tenant A  12ms   economizou 1.2k t │      │
│          │ │ 10:43 semantic   claude   tenant A  0,97   economizou 2.4k t │      │
│          │ └──────────────────────────────────────────────────────────────┘      │
└──────────┴──────────────────────────────────────────────────────────────────────┘
```

## Componentes

1. **KPIs**: hit total, exact, semantic, economia estimada.
2. **Gráfico de hit rate** por dia, com linhas separadas e tabela alternativa.
3. **Top modelos por economia** (tokens e US$).
4. **Configuração de camadas**: L1 (TTL curto), L2 (TTL), semântico (threshold/TTL), feature flag.
5. **Invalidação**: por tenant, modelo, provider, versão de prompt e chave; confirmação e escopo visível.
6. **Eventos**: hit/miss, similaridade, latência, tokens economizados; nunca exibir prompt completo.

## Dados e APIs

- `GET /admin/v1/cache/stats?range=24h`
- `GET /admin/v1/cache/events?limit=50`
- `POST /admin/v1/cache/invalidate` `{scope, id, model, prompt_version}`
- `GET/PATCH /admin/v1/cache/settings`

## Estados

- **Loading:** skeleton nos KPIs e gráfico.
- **Vazio:** “Sem tráfego de cache no período”.
- **Erro:** banner com retry; se Redis off-line, mostrar “cache degradado”.
- **Perigo:** invalidação global exige confirmação com digitação.

## Lazy loading

- Gráficos e tabela de eventos carregados ao entrar na aba.
- Estatísticas agregadas no servidor; eventos paginados.
- Reindexação semântica sempre assíncrona com barra de progresso.

## Segurança e a11y

- Nunca exibir payload de cache; apenas hash/ID.
- Escopo de invalidação sempre com `tenant_id` visível.
- Cores de hit/miss acompanhadas de texto e ícone.
- Teclado e leitor de tela no gráfico via tabela oculta.
