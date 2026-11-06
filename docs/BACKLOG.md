# dontstop — Backlog de implementação (slices verticais)

> **Status:** P0-α fatiado e publicado. O commit inicial está na `main`; a Epic [E1](https://github.com/alltomatos/dontstop/issues/1) e as slices E1.1–E1.11 foram criadas no GitHub. O restante do backlog permanece como candidato para os próximos ciclos.
> Fontes: [PRD](01-PRD.md), [DER](02-DER.md), [UML](03-UML.md), [Análise de stack](04-ANALISE-STACK.md), [ADRs](05-ADRS.md), [Roadmap](06-ROADMAP-RISCOS.md), [DEVELOPER-ROADMAP](../DEVELOPER-ROADMAP.md) e [OpenAPI](contracts/openapi-v1.yaml).

## Convenções

- **ID:** `S<fase>.<n>`; **Blocked by** lista dependências.
- **Aceite:** referencia requisitos do PRD (`FR-xx`, `NFR-xx`) e artefatos.
- **Definição de pronto:** código + testes + observabilidade + documentação + revisão de segurança.

## P0 — MVP single-tenant

| ID | Slice | Entrega | Blocked by | Aceite |
|---|---|---|---|---|
| S0.1 | Bootstrap do monorepo | `gateway/` Go, `admin/` Next.js, `deploy/`, CI, lint, Dockerfiles | — | build + lint + testes vazios verdes |
| S0.2 | Stack Swarm | `stack.yml`, Traefik, redes overlay, secrets, healthchecks | S0.1 | `docker stack deploy` sobe serviços e `/healthz` responde |
| S0.3 | Schema e migrações | Postgres + tenants/keys/providers/accounts/requests/audit | S0.1 | migrações idempotentes; tenant padrão; testes |
| S0.4 | Auth e tenant context | API key Argon2id, `tenant_id`, middleware | S0.3 | FR-35, FR-36; teste negativo de tenant |
| S0.5 | Catálogo de provedores | 2 provedores (OpenAI, Anthropic), modelos e capabilities | S0.3 | FR-07, FR-12; `GET /v1/models` |
| S0.6 | Adaptadores e tradução | Interface de provider, tradução request/response, SSE | S0.5 | FR-01, FR-05; teste de contrato OpenAI |
| S0.7 | Contas e rotação | Múltiplas contas, prioridade/peso, health check | S0.5 | FR-08, FR-10; teste de seleção |
| S0.8 | Circuit breaker e cooldown | Redis para estado, half-open, cooldown exponencial | S0.7 | FR-15, FR-16; chaos de conta/provider |
| S0.9 | Cache exato | Hash normalizado, TTL, invalidação, escopo | S0.4, S0.6 | FR-19, FR-21, FR-22; hit ≥ 20% em teste |
| S0.10 | Rate limit | Token bucket Lua por tenant/chave/modelo | S0.4 | FR-37, NFR-09; 429 com `Retry-After` |
| S0.11 | Webshare | Import/sync, health, score, rotação, quarentena | S0.7 | FR-24..FR-28; falha do pool não derruba tráfego |
| S0.12 | DLP regex | CPF/CNPJ, Luhn, e-mail, JWT, chaves; allow/redact/block/audit | S0.4 | FR-29, FR-30, FR-33; zero PII em log |
| S0.13 | Uso e custo | NATS + batch para `requests`/`usage_daily`; custo por modelo | S0.6 | FR-45; custo reconciliado ≤ 1% |
| S0.14 | Observabilidade | OTel + backend gerenciado (Datadog/Azure Monitor), SLOs e alertas; sem Prometheus/Grafana | S0.2, S0.6 | FR-40, FR-41, FR-42; trace completo |
| S0.15 | Admin básico | CRUD de providers/contas/rotas/chaves/políticas | S0.3 | FR-07; usabilidade mínima |
| S0.16 | Carga e caos | k6/ghz 5k streams/500 RPS; derrubar conta/modelo/provider | S0.8..S0.14 | NFR-01..06; RTO ≤ 2 s |
| S0.17 | Contrato OpenAPI | Especificação `/v1` + admin, geração de tipos e validação no CI | S0.4 | contrato versionado; SDK OpenAI sem patch |
| S0.18 | Design system e wireframes | Validar `design-system/MASTER.md` + mockup; implementar telas | S0.15 | revisão de UX e acessibilidade |
| S0.19 | Lazy loading do console | Code splitting por rota, charts/Monaco/logs sob demanda, virtualização e skeletons | S0.15 | LCP ≤ 2,5 s; INP ≤ 200 ms; bundle ≤ 250 KB |
| S0.20 | Lazy init do backend | Adaptadores, catálogo, embeddings e proxy sob demanda com timeout/fail-open | S0.6, S0.11 | init ≤ 50 ms; cold start testado |
| S0.21 | Cache em camadas | L1 in-process + invalidação pub/sub sobre o exact cache L2 | S0.9 | hit L1 ≥ 5%; Redis ops −10% |
| S0.22 | Endpoint Registry | Catálogo versionado + políticas por endpoint (auth, rate, modelos, cache, DLP, proxy, limites) | S0.4, S0.6 | FR-55, FR-56; mudança em ≤ 5 s |
| S0.23 | API Manager | Chaves com escopos de endpoint/modelo ativo de provider conectado, quota, RPM/TPM/USD, expiração ou sem expiração, IP allowlist, max sessions, rotação e uso | S0.4, S0.22 | FR-57..FR-59; segredo só na criação; `expires_at` nulo permitido |
| S0.24 | Endpoint final | URLs pública/local, servidor, catálogo por categoria, aliases para CLIs e system prompt opcional | S0.4, S0.22 | FR-64; copiar URL; sem túneis |
| S0.25 | Providers UI | Grade/lista, cards agrupados por auth, filtros por capability/estado, busca por modelo e ações em lote | S0.5, S0.7 | FR-07/FR-10; drawer de detalhes; toggle persistido |
| S0.26 | MCP | Servidor desligado por padrão, toggle global, transportes stdio/SSE/Streamable HTTP, status/atividade, ferramentas/escopos e endpoint copiável | S0.4, S0.22 | FR-65..FR-69; sem túnel; ativação auditada |
| S0.27 | i18n + guia | Seis locales, onboarding, Settings, fallback, guia por página/modal e link contextual | S0.1, S0.25 | FR-70..FR-73; testes de locale/fallback/links; imagens 6 idiomas |

## P1 — Escala e controle

| ID | Slice | Entrega | Blocked by | Aceite |
|---|---|---|---|---|
| S1.1 | Cache semântico | Embeddings locais + Qdrant + threshold/TTL | S0.9, S0.13 | FR-20; hit total ≥ 40%; erro < 0,5% |
| S1.2 | Provedores adicionais | Gemini, Azure, Bedrock, OpenAI-compatible (15+) | S0.6 | FR-07; health e modelos |
| S1.3 | OAuth de providers | Fluxo OAuth, refresh, rotação | S0.7 | FR-09; refresh sem re-login |
| S1.4 | Quotas e orçamento | Limites por tenant/chave/modelo + alertas | S0.10, S0.13 | FR-37; bloqueio em 100% |
| S1.5 | RLS e RBAC | `app.tenant_id`, políticas, papéis e testes IDOR | S0.4 | FR-36, FR-38; zero vazamento |
| S1.6 | DLP NER | Presidio sidecar + janelas de streaming | S0.12 | FR-31; p95 ≤ 60 ms |
| S1.7 | Console completo | Provider/route/cache/DLP/usage/export | S0.15 | usabilidade com 5 usuários |
| S1.8 | Autoscaling Swarm | Controlador externo (métricas do backend + Docker API) | S0.2, S0.14 | escala 20k/1k; 99,9% em 30 d |
| S1.9 | Analytics | Agregação diária; avaliar ClickHouse | S0.13 | consultas p95 ≤ 2 s |
| S1.10 | Hardening | Pen test, fuzz, SBOM, assinatura de imagem | S0.16 | achados críticos = 0 |
| S1.11 | SSO corporativo | OIDC/SAML + MFA + mapeamento de papéis | S1.5 | login SSO, logout e auditoria |
| S1.12 | SIEM e chargeback | Exportação assinada por período/centro de custo | S0.13, S1.4 | exportação reconciliada |
| S1.13 | Air-gapped | Registry privado, backend de observabilidade alternativo e instalação offline | S0.2, S0.14 | instalação sem SaaS externo |
| S1.14 | Aprovação de chaves | Workflow de aprovação/auditoria de criação, rotação e revogação | S0.23, S1.11 | trilha imutável e aprovador |
| S1.15 | OpenAPI por endpoint | Spec versionada, importação e docs geradas | S0.17, S0.22 | contrato validado no CI |
| S1.16 | Grupos de chaves | Allow/deny por modelo/provider com deny precedence | S0.23 | FR-63; teste de precedência |
| S1.17 | Access schedule e sessões | Janela de dias/horário/timezone e max sessions | S0.23 | FR-57; bloqueio fora da janela |

## P2 — Multi-tenancy e evolução

| ID | Slice | Entrega | Blocked by | Aceite |
|---|---|---|---|---|
| S2.1 | Multi-tenancy GA | Isolamento físico/lógico, chaves por tenant e SCIM | S1.5, S1.11 | 1.000 tenants; zero vazamento |
| S2.2 | HA multi-região | Clusters Swarm + replicação + LB global | S1.8 | RTO ≤ 15 min; RPO ≤ 5 min |
| S2.3 | WebSocket/Responses | Suporte completo a Responses API | S1.2 | contrato e testes gravados |
| S2.4 | SDK de plugins | Interface de provider plugável | S1.2 | plugin de exemplo + testes |
| S2.5 | DLP avançado | Tokenização reversível e classificação | S1.6 | rehidratação auditada |
| S2.6 | Billing | Faturamento e relatórios por tenant | S1.4 | reconciliação mensal |
| S2.7 | ClickHouse | Analytics detalhado | S1.9 | > 50M linhas/mês |

## Dependências externas

- Commit inicial no remote + GitHub Issues (bloqueia `/to-issues`).
- KMS/Vault para envelope encryption.
- Plano Webshare e teste de reputação.
- Credenciais de teste dos provedores.
- DPO/jurídico para LGPD.

## Como fatiar em Issues após o commit inicial

1. Criar labels `p0`, `p1`, `p2`, `security`, `infra`, `provider`, `cache`, `dlp`.
2. Uma Issue por linha da tabela, com corpo: contexto, escopo, fora de escopo, critérios de aceite (PRD), artefatos (links), dependências (`Blocked by #ID`).
3. Milestone por fase; projeto Kanban com colunas Backlog/Ready/In Progress/Review/Done.
4. PR obrigatório com testes, observabilidade e atualização deste backlog.

## Fontes consultadas

- [PRD](01-PRD.md), [DER](02-DER.md), [UML](03-UML.md), [Análise de stack](04-ANALISE-STACK.md), [ADRs](05-ADRS.md) e [Roadmap](06-ROADMAP-RISCOS.md).
- [Referência de Endpoints/API Manager](reference/ENDPOINTS-API-MANAGER-REFERENCIAS.md) — lições de 9Router e OmniRoute.
- `[REPO]` — estado greenfield de `D:\dev\dontstop` (sem commits; remote `origin` configurado).
