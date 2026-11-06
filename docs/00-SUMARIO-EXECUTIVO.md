# dontstop — Sumário executivo

> Fontes: `[REPO]`, `[MAP-9R]`, `[MAP-OMNI]`, `[MAP-ALLCODE]` (definidas no [README](README.md)).
> Natureza: planejamento. Não há código de aplicação no repositório.

## 1. Visão

O dontstop será o **gateway corporativo de IA** da organização: um único endpoint compatível com OpenAI para consumir LLMs em nuvem e locais, com controle de custo, resiliência, proxy de saída e governança de dados. O P0 é **single-tenant**, mas o desenho já carrega `tenant_id`, namespaces de cache e políticas por tenant para permitir multi-tenancy sem migração destrutiva.

## 2. Decisões principais

| # | Decisão | Escolha | Justificativa resumida | ADR |
|---|---|---|---|---|
| 1 | Data plane | **Go 1.24+** (`net/http`, goroutines, streaming SSE) | Milhares de conexões com memória previsível, pprof e baixo overhead por conexão; melhor relação performance/velocidade de entrega que Rust | ADR-001 |
| 2 | Control plane/UI | **TypeScript + Next.js 16 + React 19** | Administração, catálogo e observabilidade; reaproveita o ecossistema dos projetos auditados | ADR-001 |
| 3 | Banco de verdade | **PostgreSQL 17 + PgBouncer + RLS** | Transações, JSONB, partições, RLS e maturidade operacional | ADR-002 |
| 4 | Cache exato e coordenação | **Redis 8 (ou Valkey 8)** | Sub-milissegundo, TTL, Lua para rate limit/locks, estado de circuit breaker | ADR-003 |
| 5 | Cache semântico | **Qdrant** (HNSW + filtros por payload) | ANN de baixa latência com isolamento por tenant/modelo; alternativa pgvector no P1 se ops precisar ser mínima | ADR-004 |
| 6 | Eventos/filas | **NATS JetStream** | Leve, Go-native, at-least-once para auditoria, webhooks e agregação | ADR-005 |
| 7 | Observabilidade | **OpenTelemetry + backend gerenciado (Datadog; alternativa Azure Monitor)** | Golden signals e tracing sem operar stack própria; **sem Grafana/Prometheus/Loki/Tempo** | ADR-006 |
| 8 | Proxy de saída | **Webshare** com pool gerenciado (import/sync/health/rotação) | Requisito explícito; isolamento de egress por conta e métricas por proxy | ADR-008 |
| 9 | DLP | **Regex + validadores in-process (P0) e Presidio sidecar (P1)** | Latência baixa no caminho quente e NER especializado quando necessário | ADR-009 |
| 10 | Deploy | **Docker Swarm** (serviços, réplicas e autoscaling via controlador externo); Compose no dev | Escala horizontal, rolling update e isolamento de tarefas; single-node no P0 | ADR-010 |
| 11 | Lazy loading e cache em camadas | Frontend com code splitting/Suspense; backend com init sob demanda; L1+L2+semântico | Reduz TTI, memória e custo sem sacrificar o caminho quente | ADR-013 |
| 12 | Identidade corporativa | OIDC/SAML + RBAC + MFA (P1/P2); API keys no P0 | SSO, auditoria e governança de acesso | ADR-014 |
| 13 | Endpoints/API Manager | Registry versionado + políticas por endpoint + API keys com escopos | Governança da superfície pública e menor privilégio | ADR-015 |
| 14 | MCP | Desligado por padrão; stdio/SSE/Streamable HTTP; ferramentas com escopo; local-only e sem túneis | Expor ferramentas sem ampliar a superfície de ataque | ADR-016 |
| 15 | i18n e guia | Seis locales com fallback en, onboarding e Settings; guia versionado por página/modal | Interface acessível e ajuda contextual em qualquer idioma | ADR-017 |

## 3. Metas numéricas (alvos a validar)

| Dimensão | Meta |
|---|---|
| RPS sustentado no cluster | 1.000 req/s |
| Burst | 3.000 req/s por 60 s |
| Streams SSE simultâneos | 20.000 no cluster (5.000 por réplica) |
| Overhead do gateway | p95 ≤ 30 ms; p99 ≤ 80 ms |
| Time-to-first-token (ponta a ponta) | p95 ≤ 1,5 s (dominado pelo upstream) |
| Disponibilidade mensal | 99,9% (43,2 min de indisponibilidade) |
| Failover de conta/provider | RTO ≤ 2 s; sem erro visível ao cliente quando há alternativa saudável |
| RPO | 0 para configuração/políticas; ≤ 60 s para logs/uso |
| Cache exato | hit rate ≥ 25% |
| Cache semântico | +15% de hit rate; total ≥ 40% |
| Redução de custo | ≥ 30% vs. baseline sem cache |
| Rate limit | decisão p99 ≤ 3 ms |
| DLP | p95 ≤ 20 ms (regex) e ≤ 60 ms (NER); bloqueio ≤ 50 ms |
| Retenção | metadados 90 d quente/1 ano frio; payload desligado por padrão; auditoria 1 ano |
| UI (console) | LCP ≤ 2,5 s; INP ≤ 200 ms; CLS ≤ 0,1; bundle inicial ≤ 250 KB gzip |
| Lazy loading | nenhum módulo opcional no primeiro paint; init sob demanda ≤ 50 ms no caminho quente |

## 4. Riscos críticos

| Risco | Impacto | Mitigação |
|---|---|---|
| **Complexidade de dois planos** (gateway Go + console TS) | Alto custo inicial e duas pipelines | Contratos OpenAPI versionados, CI separada, biblioteca de tipos gerada |
| **Cache semântico gerar resposta errada** | Incidente de confiança e custo | Só cachear temperatura baixa e prompts determinísticos; threshold alto; TTL curto; invalidação por modelo/tenant |
| **Vazamento de credenciais de provedores** | Comprometimento financeiro e de dados | AES-256-GCM + envelope (KMS/Vault), Argon2id para API keys, redaction, rotação e auditoria |
| **Proxy Webshare instável ou banido** | Falhas em cascata e latência | Health checks, quarentena, score por taxa de sucesso/latência, fallback direto, limite de concorrência por proxy |
| **DLP no streaming** | Falso positivo/negativo e latência | Janelas com sobreposição, regras determinísticas no P0, bloqueio fail-closed só para segredos de alta confiança |
| **Volume de logs/uso** | Crescimento de storage e custo | Payload off por padrão, amostragem, agregação em NATS, partições e ClickHouse no P2 |
| **Dependência de observabilidade SaaS** | Custo/lock-in e indisponibilidade em ambiente air-gapped | OTel como contrato; orçamento de cardinalidade; decisão D11 para on-prem |
| **Lazy loading quebrar o caminho quente** | Latência de primeira requisição e erros | Init eager apenas do crítico; lazy com timeout e fallback; testes de cold start |
| **Dependência de um único time/stack** | Bus factor | ADRs, runbooks, testes de contrato e fallback TS documentado (ADR-001) |

## 5. Fases (resumo)

| Fase | Prazo indicativo | Entrega |
|---|---|---|
| **P0 — MVP single-tenant** | 0–3 meses | `/v1/chat/completions`, 5 provedores, contas múltiplas, failover, cache exato, Webshare, DLP regex, Postgres/Redis, OTel, deploy Compose/Swarm |
| **P1 — Escala e controle** | 3–6 meses | Cache semântico, 15+ provedores, quotas/orçamento, console completo, NATS, DLP NER, OAuth de providers, HA e RLS |
| **P2 — Multi-tenancy e evolução** | 6–12 meses | Multi-tenancy GA (isolamento por tenant), chaves por tenant, HA multi-região, WebSocket, SDK de plugins, DLP avançado |

## 6. Decisões pendentes (não inventadas)

1. **Cloud vs. on-premise** e região primária.
2. **Plano Webshare** (quantidade de proxies e franquia de banda).
3. **Volume real** de RPS, tokens e tenants.
4. **Escopo de compliance** além da LGPD.
5. **Stack do time** (Go vs. TypeScript) — ADR-001 tem fallback.
6. **Provedores iniciais** e prioridade de modelos.
7. **Política de retenção de payload** (off por padrão na proposta).
8. **Estratégia de embeddings** para cache semântico (modelo local vs. provedor).

## 7. Fontes consultadas

- `[REPO]` — `D:\dev\dontstop`: greenfield, 0 arquivos de código, sem commits.
- `[MAP-9R]` — `../MAPEAMENTO-ARQUITETURAL-9ROUTER.md`: fallback, combos, tradução, SQLite, multitenancy, dívidas de segurança e concorrência.
- `[MAP-OMNI]` — `../MAPEAMENTO-ARQUITETURAL-OMNIROUTE.md`: escala, 380 provedores, 222 migrações, Redis opcional, policies, route guard, i18n e dívidas.
- `[MAP-ALLCODE]` — `../MAPEAMENTO-PROVEDORES-ALLCODE.md`: catálogo/plugins, protocolos, adaptadores proprietários, rotação e lacunas de feedback.
