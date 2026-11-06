# dontstop — Roadmap, riscos e decisões pendentes

> Fontes: `[REPO]`, `[MAP-9R]`, `[MAP-OMNI]`, `[MAP-ALLCODE]` (ver [README](README.md)).
> Prazos são indicativos e dependem de time, infraestrutura e validação de capacidade.

## 1. Visão de fases

| Fase | Janela | Tema | Critério de saída |
|---|---|---|---|
| **P0** | 0–3 meses | MVP single-tenant, caminho quente confiável | 5k streams e 500 RPS validados; failover ≤ 2 s; cache exato; DLP regex; Webshare; SLOs no ar |
| **P1** | 3–6 meses | Escala, controle e governança | 20k streams e 1k RPS; 99,9% em 30 dias; cache total ≥ 40%; DLP NER; quotas/orçamento; RLS |
| **P2** | 6–12 meses | Multi-tenancy GA e evolução | 1.000 tenants; HA multi-região; WebSocket; SDK de plugins; DLP avançado; billing |

## 2. P0 — MVP (0–3 meses)

### 2.1 Entregáveis

| Área | Entregável | Dependência |
|---|---|---|
| Data plane | `/v1/chat/completions` + `/v1/models`, SSE, cancelamento e backpressure | ADR-001, ADR-012 |
| Providers | 5 provedores (OpenAI, Anthropic, Gemini, Azure, OpenAI-compatible) | Catálogo e credenciais |
| Contas | Múltiplas contas por provider, prioridade/peso, health check | ADR-002/011 |
| Failover | Circuit breaker, cooldown exponencial, lockout por modelo | Redis (ADR-003) |
| Cache | L1 in-process + exact cache L2 com TTL, escopo por tenant/chave, invalidação | Redis (ADR-003/013) |
| Rate limit | Token bucket por tenant/chave/modelo, 429 com `Retry-After` | Redis |
| Proxy | Import/sync Webshare, health, rotação, quarentena, fallback direto | ADR-008 |
| DLP | Regex/validadores (CPF/CNPJ, Luhn, e-mail, JWT, chaves), allow/redact/block/audit | ADR-009 |
| Identidade | API keys Argon2id, tenant padrão, RBAC básico de admin | ADR-007 |
| Observabilidade | OTel + backend gerenciado (Datadog/Azure Monitor), SLOs e logs sem PII | ADR-006 |
| Admin | CRUD de providers/contas/rotas/chaves/políticas; visão de saúde | Control plane |
| Lazy loading | Code splitting no console + init sob demanda no backend | ADR-013 |
| UI/UX | `design-system/MASTER.md` + mockup validado antes do código | `/ui-ux-pro-max` |
| Deploy | Docker Compose (dev) + Docker Swarm single-region (prod) | ADR-010 |

### 2.2 Critérios de saída (go/no-go)

- Teste de carga: **5.000 streams** simultâneos e **500 RPS** por 30 min, overhead p95 ≤ 30 ms.
- Chaos: derrubar conta, modelo e provider; sucesso com alternativa em ≤ 2 s.
- Cache exato: hit ≥ 20% em carga sintética; isolamento por tenant comprovado.
- DLP: 100% dos padrões cobertos; zero PII em log; falso positivo < 5% na amostra.
- Segurança: `gosec`/`govulncheck` limpos, gitleaks, Trivy, teste de IDOR e SSRF.
- SLO: dashboards e alertas de burn rate ativos no backend gerenciado.
- UI: LCP ≤ 2,5 s, INP ≤ 200 ms, CLS ≤ 0,1 e axe sem violações críticas.
- Lazy: nenhum chunk opcional no primeiro paint; cold start do caminho quente ≤ 50 ms de overhead.

### 2.3 Fora do P0

Semantic cache, OAuth de providers, `/v1/responses`, WebSocket, multi-região, RLS, billing.

## 3. P1 — Escala e controle (3–6 meses)

| Área | Entregável | Critério |
|---|---|---|
| Cache | Semantic cache (Qdrant), embeddings locais, threshold e TTL | hit total ≥ 40%; erro < 0,5% |
| Providers | 15+ provedores, OAuth/refresh, catálogo de modelos | validação e health em ≤ 5 s |
| Governança | Quotas/orçamento por tenant/chave/modelo, alertas | bloqueio em 100%; relatório diário |
| DLP | Presidio sidecar (NER), janelas de streaming, modo audit→block | p95 ≤ 60 ms; FP < 3% |
| Console | Provider/route/cache/DLP/usage completos, exportação | usabilidade validada com 5 usuários |
| Escala | autoscaling via controlador externo (`docker service scale`), PgBouncer, Redis HA, NATS cluster, carga 20k/1k | SLO 99,9% por 30 dias |
| Multi-tenancy | RLS, escopos, testes negativos | zero vazamento em teste |
| Identidade | OIDC/SAML + MFA; papéis e auditoria de sessão | login SSO e IDOR cobertos |
| Corporativo | Exportação SIEM, chargeback e retenção por tenant | relatório reconciliado |
| Analytics | Agregação diária; avaliar ClickHouse | consultas p95 ≤ 2 s |

**Critério de saída:** todos os NFRs do PRD §7 medidos em produção por 30 dias; custo por 1M tokens −30% vs. baseline.

## 4. P2 — Multi-tenancy e evolução (6–12 meses)

- Multi-tenancy GA com RLS, chaves e retenção por tenant.
- HA multi-região (RTO ≤ 15 min, RPO ≤ 5 min para metadados).
- WebSocket/Responses API completo; SDK de plugins de provider.
- DLP avançado com tokenização reversível e classificação.
- Billing/faturamento e marketplace interno.
- ClickHouse para analytics detalhado se > 50M linhas/mês.
- Certificações e auditoria externa conforme compliance.

## 5. Riscos

| # | Risco | Prob. | Impacto | Mitigação | Gatilho de alerta |
|---|---|---|---|---|---|
| R1 | Time sem Go atrasa o P0 | Média | Alto | ADR-001 com fallback TS; spike de 2 semanas; pareamento | Velocity < 70% por 2 sprints |
| R2 | Cache semântico responde errado | Média | Alto | Threshold 0,95, TTL curto, escopo por tenant, feature flag | Taxa de reclamação > 0,1% |
| R3 | Banda Webshare estoura o plano | Alta | Alto | Proxy só quando necessário; medir TB/mês; fallback direto | > 80% da franquia |
| R4 | DLP gera falso positivo e bloqueia trabalho | Média | Médio | Modo audit antes de block; allowlist; feedback loop | FP > 5% |
| R5 | Postgres vira gargalo de escrita | Média | Alto | Batch via NATS, PgBouncer, partições, amostragem; ClickHouse P2 | Write latency p95 > 50 ms |
| R6 | Redis indisponível derruba rate limit/circuit | Média | Alto | HA, fallback local degradado, timeout curto | Erros Redis > 0,1% |
| R7 | RLS degrada performance | Baixa | Médio | Índices com tenant; teste de carga; schema-per-tenant P2 | p95 de queries +30% |
| R8 | Vazamento de credenciais | Baixa | Crítico | AES-GCM + KMS, Argon2id, redaction, rotação, auditoria | Alerta de secret scanning |
| R9 | Provider muda contrato e quebra adapter | Alta | Médio | Testes de contrato gravados; feature flags; versão de protocolo | Aumento de 4xx/5xx por provider |
| R10 | Custo de infraestrutura acima do previsto | Média | Médio | Orçamento por componente; right-sizing; spot para workers | > 120% do previsto |
| R11 | Swarm sem autoscaling nativo | Alta | Médio | Scale manual com alertas no P0; controlador externo (métricas do backend + Docker API) no P1; capacidade reservada | Conexões/CPU > 70% por 10 min |
| R12 | Estado em volumes locais no Swarm | Média | Alto | Nós `tier=data` dedicados, backups testados e serviços gerenciados quando possível | Falha de backup ou de volume |
| R13 | Lock-in/custo do backend de observabilidade | Média | Médio | OTel como contrato; orçamento de cardinalidade; decisão D11 | Custo > 120% do previsto |
| R14 | Lazy loading degradar o primeiro uso | Média | Médio | Eager só do crítico; timeout/fallback; teste de cold start | TTFT p95 > meta |

## 6. Decisões pendentes

| # | Decisão | Impacto | Prazo sugerido |
|---|---|---|---|
| D1 | Cloud vs. on-premise e região | Arquitetura de deploy e latência | Antes do P0 |
| D2 | Plano Webshare (proxies/banda) | Custo e capacidade de egress | Antes do P0 |
| D3 | Stack do time (Go vs. TS) | Cronograma e contratação | Semana 1 |
| D4 | Provedores iniciais | Escopo do catálogo | Semana 1 |
| D5 | Modelo de embeddings (local vs. provedor) | Custo/latência do cache semântico | P1 |
| D6 | Retenção de payload e base legal LGPD | Compliance e storage | P0 |
| D7 | Estratégia de HA (multi-AZ vs. multi-região) | RTO/RPO e custo | P1 |
| D8 | ClickHouse vs. Postgres para analytics | Custo e complexidade | P1 (avaliar) |
| D9 | Autoscaling no Swarm: manual vs. controlador externo | Velocidade de resposta a picos | P1 |
| D10 | Serviços gerenciados vs. self-hosted no Swarm | Custo, RTO e operação | Antes do P0 |
| D11 | Backend de observabilidade (Datadog vs. Azure Monitor) e estratégia air-gapped | Custo, egress e compliance | P0 |
| D12 | Biblioteca de gráficos e padrão de visualização do console | Consistência e performance | P0 |

## 7. Dependências críticas

1. **Identidade corporativa** para OIDC/SAML no P1/P2.
2. **KMS/Vault** para envelope encryption.
3. **Contrato/plano Webshare** e teste de reputação de IPs.
4. **Acesso aos provedores** e chaves de teste para validação.
5. **Time de SRE** para Docker Swarm, observabilidade e game days.
6. **Jurídico/DPO** para DPA, retenção e base legal.
7. **IdP corporativo** (OIDC/SAML) e definição de grupos/papéis no P1.
8. **Contrato do backend de observabilidade** (Datadog/Azure Monitor) e política de retenção/cardinalidade.

## 8. KPIs de acompanhamento

| KPI | Meta P1 |
|---|---|
| Disponibilidade | ≥ 99,9% |
| Overhead p95 | ≤ 30 ms |
| Cache hit total | ≥ 40% |
| Custo por 1M tokens | −30% vs. baseline |
| Failover com sucesso | ≥ 99,5% |
| Incidentes P1 | ≤ 1/mês |
| Cobertura de testes do data plane | ≥ 70% nas linhas críticas |
| MTTR | ≤ 30 min |

## 9. Fontes consultadas

- `[MAP-9R]` — riscos de mutex global, segredos em texto plano, cache e fallback.
- `[MAP-OMNI]` — escala, circuit breaker/cooldown, policies, Redis opcional e dívidas de i18n/god-files.
- `[MAP-ALLCODE]` — ponte v1/v2, rotação sem feedback, runtime npm e relay Python.
- `[REPO]` — confirmação de greenfield; roadmap é proposta.
