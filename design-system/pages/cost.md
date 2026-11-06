# Página — Custo e Quotas

> Override do [MASTER](../MASTER.md). Foco: FinOps, orçamento, chargeback e economia de cache.

## Objetivo

Mostrar custo por tenant/provider/modelo/chave, consumo de quotas, projeção e ações de controle.

## Wireframe (desktop ≥ 1024px)

```
┌──────────┬──────────────────────────────────────────────────────────────────────┐
│ Sidebar  │ Título: Custo e Quotas                    [Exportar ▾] [Nova quota]  │
│          ├──────────────────────────────────────────────────────────────────────┤
│ Dashboard│ ┌───────────────┬───────────────┬───────────────┬───────────────┐      │
│ Providers│ │ Custo mês     │ Economia cache│ Orçamento     │ Projeção      │      │
│ Cache    │ │ US$ 8.420     │ US$ 3.910     │ 62% usado     │ US$ 13.580    │      │
│ DLP      │ │ +18% vs mês   │ -32%          │ US$ 13.500    │ +8% acima     │      │
│ Custo    │ └───────────────┴───────────────┴───────────────┴───────────────┘      │
│ Observ.  │ ┌────────────────────────────────┬──────────────────────────────┐     │
│          │ │ Custo por dia (30d)            │ Top modelos por custo        │     │
│          │ │ [área empilhada — lazy]        │ [tabela com % e tokens]      │     │
│          │ └────────────────────────────────┴──────────────────────────────┘     │
│          │ ┌──────────────────────────────────────────────────────────────┐      │
│          │ │ Quotas e orçamentos                                          │      │
│          │ │ Escopo        Métrica   Limite     Usado    Status           │      │
│          │ │ tenant        custo     US$ 13.5k  62%      ● OK             │      │
│          │ │ key: prod-1   tokens    50M       48,2M    ◐ 96%            │      │
│          │ └──────────────────────────────────────────────────────────────┘      │
└──────────┴──────────────────────────────────────────────────────────────────────┘
```

## Componentes

1. **KPI cards**: custo do mês, economia de cache, orçamento usado, projeção.
2. **Gráfico de custo diário** por provider/modelo (área empilhada), lazy.
3. **Tabela Top modelos** com tokens, custo, % do total e cache hit.
4. **Quotas/orçamentos** por tenant, chave, modelo e provider; barra de progresso + status.
5. **Chargeback**: exportação por centro de custo e período.
6. **Ações**: criar quota, ajustar orçamento, exportar CSV/JSON, simular economia.

## Dados e APIs

- `GET /admin/v1/costs/summary?range=30d`
- `GET /admin/v1/costs/by-model?range=30d`
- `GET /admin/v1/quotas`
- `POST/PATCH /admin/v1/quotas`
- `GET /admin/v1/costs/export?format=csv`

## Estados

- **Loading:** skeleton nos KPIs e tabela.
- **Vazio:** “Sem custo no período”.
- **Erro:** fallback para agregados diários; banner de dados parciais.
- **Alerta:** quota ≥ 80% (warning) e ≥ 100% (destructive) com texto.

## Lazy loading

- Gráficos e tabela detalhada sob demanda.
- Exportação assíncrona (job + download).
- Agregações cacheadas por 60 s.

## Segurança e a11y

- Números com locale e unidade; `tabular-nums`.
- Barras com valor textual e status, não apenas cor.
- Exportação exige permissão de FinOps e é auditada.
- Teclado e leitor de tela na tabela.
