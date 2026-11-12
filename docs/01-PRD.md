# dontstop — Product Requirements Document (PRD)

> Fontes: `[REPO]`, `[MAP-9R]`, `[MAP-OMNI]`, `[MAP-ALLCODE]` (ver [README](README.md)).
> Versão: 1.0 — 2026-11-06. **Nenhum código foi implementado.**

## 1. Contexto e problema

A organização consome LLMs por múltiplos times, ferramentas e chaves espalhadas. Isso gera:

- custo sem visibilidade e sem padronização de cache;
- fragilidade quando uma conta/provider atinge limite ou cai;
- exposição de dados sensíveis em prompts e respostas;
- dificuldade de aplicar egress controlado (proxy) e auditoria.

Os gateways auditados mostram que o padrão funciona: um endpoint OpenAI-compatible, catálogo de provedores, tradução de protocolos, fallback em camadas e governança. Também mostram as dívidas a evitar: mutex global, segredos em texto plano, criptografia opt-in, rate limit apenas em memória, monólitos e ausência de CI de qualidade `[MAP-9R]` `[MAP-OMNI]` `[MAP-ALLCODE]`.

## 2. Objetivos e métricas de sucesso

| Objetivo | Métrica | Meta |
|---|---|---|
| Centralizar consumo | % de chamadas de LLM passando pelo gateway | ≥ 80% em 90 dias após GA |
| Reduzir custo | custo por 1M tokens vs. baseline sem cache | −30% |
| Melhorar disponibilidade | sucesso de requisições com fallback disponível | ≥ 99,5% |
| Resiliência | failover de conta/provider | RTO ≤ 2 s, sem erro ao cliente quando há alternativa |
| Governança | requisições com decisão DLP auditada | 100% das requisições com política ativa |
| Performance | overhead do gateway | p95 ≤ 30 ms, p99 ≤ 80 ms |
| Adoção | chaves ativas / times integrados | ≥ 50 chaves e 10 times no P1 |

## 3. Não-objetivos (P0/P1)

- Treinar ou hospedar modelos.
- Substituir o provedor de identidade corporativo (SSO completo fica no P1/P2; P0 usa API keys + admin local).
- Faturamento automatizado com cobrança real (P2).
- Multi-região ativo-ativo (P2).
- Marketplace de plugins de terceiros (P2).
- Suporte a todos os protocolos proprietários de todos os provedores no P0 (cobertura inicial: OpenAI, Anthropic, Gemini, Azure, Bedrock e OpenAI-compatible).
- **Túneis de rede** (Cloudflare Tunnel, Tailscale Funnel, ngrok): fora de escopo em todas as fases.

## 4. Personas

| Persona | Objetivo | Dores | Necessidades |
|---|---|---|---|
| **Admin de Plataforma** | Operar o gateway e os provedores | Configuração fragmentada, falta de visão de saúde | Catálogo, contas, políticas, health, logs |
| **FinOps / Gestão** | Controlar custo e orçamento | Sem visão por time/modelo, surpresas de fatura | Quotas, custo por token, cache hit, relatórios |
| **Segurança / DLP** | Evitar vazamento de dados | Prompts com PII/segredos, falta de auditoria | Regras, redaction, bloqueio, trilha imutável |
| **Desenvolvedor / Tech Lead** | Usar LLM sem atrito | Chaves espalhadas, limites, indisponibilidade | Endpoint único, API key, fallback, compatibilidade OpenAI |
| **Auditor** | Verificar conformidade | Falta de evidência | Retenção, exportação, integridade de logs |
| **Cliente interno (tenant futuro)** | Isolar seu time/empresa | Falta de isolamento e quotas | `tenant_id`, chaves, políticas e limites próprios |

## 5. Jornadas principais

1. **Integrar uma ferramenta**: admin cria API key → dev aponta base URL e key → primeira chamada funciona em < 5 min.
2. **Cadastrar um provedor**: admin informa provider/conta/credencial → validação de saúde → modelos disponíveis no catálogo.
3. **Sobrecarga de um provedor**: gateway detecta 429/erro → abre circuit breaker → tenta próxima conta → próximo provider/combo → registra evento.
4. **Cache**: requisição semelhante a uma anterior elegível → exact cache hit ou semantic hit → resposta em < 50 ms e custo zero de upstream.
5. **Proxy**: conta configurada com pool Webshare → gateway escolhe proxy saudável → registra latência/sucesso → quarentena em caso de falha.
6. **DLP**: prompt contém CPF/segredo → política decide redigir ou bloquear → resposta/auditoria registra a decisão sem expor o dado.
7. **Governança**: gestor define orçamento por chave/time → gateway bloqueia ao atingir 100% → alerta e relatório.

## 6. Requisitos funcionais

### Épico E1 — API compatível

| ID | Requisito | Pri. | Critério de aceite |
|---|---|---|---|
| FR-01 | Expor `POST /v1/chat/completions` compatível com OpenAI, streaming e não-streaming | P0 | SDK oficial OpenAI funciona sem alteração; SSE com `data:` e `[DONE]` |
| FR-02 | Expor `GET /v1/models` com catálogo filtrado por tenant/política | P0 | Lista apenas modelos permitidos e disponíveis |
| FR-03 | Expor `POST /v1/embeddings` | P1 | Compatível com OpenAI; usado também pelo cache semântico |
| FR-04 | Expor `POST /v1/responses` (Responses API) | P1 | Conversão Chat Completions ↔ Responses testada por contrato |
| FR-05 | Suportar `stream=true` com backpressure e cancelamento do cliente | P0 | Cliente desconecta → upstream é abortado em ≤ 1 s; sem goroutine órfã |
| FR-06 | Limitar corpo e validar schema (Zod/JSON Schema equivalente) | P0 | Corpo acima do limite retorna 413; schema inválido retorna 400 sem stack trace |

### Épico E2 — Provedores e credenciais

| ID | Requisito | Pri. | Critério de aceite |
|---|---|---|---|
| FR-07 | CRUD de provedores e modelos com visão em **grade ou lista**, filtros por auth/capability e ações em lote | P0 | Alternância sem perder filtros; cards/linhas com contadores; mudança reflete em ≤ 5 s |
| FR-08 | Múltiplas contas por provedor com prioridade/peso | P0 | Rotação e escolha por prioridade/peso conforme política |
| FR-09 | Credenciais via API key e OAuth (quando aplicável) | P0/P1 | API key no P0; OAuth no P1; refresh automático |
| FR-10 | Validação de credencial e health check (individual e em lote) | P0 | Cadastro só é “ativo” após teste bem-sucedido ou opt-in explícito; “Testar todos” reporta por card |
| FR-11 | Criptografia de credenciais em repouso | P0 | AES-256-GCM com chave externa; nunca logada; rotação suportada |
| FR-12 | Catálogo de modelos com capabilities (visão, tools, contexto) | P0 | `GET /v1/models` expõe capabilities normalizadas |

### Épico E3 — Roteamento, failover e resiliência

| ID | Requisito | Pri. | Critério de aceite |
|---|---|---|---|
| FR-13 | Rotas/combos com ordem e estratégias (priority, weighted, least-latency) | P0 | Mudança de estratégia sem restart; teste de failover automatizado |
| FR-14 | Fallback em três níveis: conta → modelo → provider | P0 | Falha injetada em conta/modelo/provider resulta em sucesso quando há alternativa, em ≤ 2 s |
| FR-15 | Circuit breaker por provider (closed/open/half-open) | P0 | 5 falhas ou 50% de erro em 30 s abre; half-open após 15 s |
| FR-16 | Cooldown exponencial por conta e lockout por modelo | P0 | 429/401/5xx aplicam cooldown; modelo ruim não derruba a conta inteira |
| FR-16.1 | Failover padrão entre contas do mesmo provedor | P0 | 429/quota/5xx/timeout tenta a próxima conta do provedor antes de outro provedor; 400 do cliente não consome outra conta |
| FR-17 | Idempotência de requisições (header `Idempotency-Key`) | P1 | Repetição devolve a mesma resposta dentro da janela configurada |
| FR-18 | Timeout, retry e abort budget por requisição | P0 | Nenhuma requisição excede o budget total configurado |

### Épico E4 — Cache

| ID | Requisito | Pri. | Critério de aceite |
|---|---|---|---|
| FR-19 | Cache exato por hash normalizado (tenant, modelo, mensagens, parâmetros) | P0 | Hit devolve resposta em ≤ 50 ms; TTL configurável |
| FR-20 | Cache semântico por embedding + ANN | P1 | Threshold configurável; hit ≥ 0,95 de similaridade; isolamento por tenant/modelo |
| FR-21 | Invalidação por modelo, provider, tenant e versão de prompt | P0 | Invalidação propaga em ≤ 5 s |
| FR-22 | Cache scope: global do tenant, por API key ou desligado | P0 | Configuração por chave/tenant; sem vazamento entre escopos |
| FR-23 | Estatísticas de hit/miss e economia estimada | P0 | Dashboard e API com números por período |

### Épico E5 — Proxy Webshare

| ID | Requisito | Pri. | Critério de aceite |
|---|---|---|---|
| FR-24 | Importar e sincronizar lista de proxies Webshare | P0 | Sync manual e agendado; falha não derruba o serviço |
| FR-25 | Health check e score por proxy | P0 | Latência, taxa de sucesso e banimento por proxy; quarentena automática |
| FR-26 | Rotação: sticky por conta, round-robin e least-latency | P0 | Estratégia por pool/provider; teste de rotação determinístico |
| FR-27 | Limite de concorrência e isolamento de egress por proxy | P0 | Proxy saturado não recebe novas conexões; sem cabeça-de-fila global |
| FR-28 | Fallback para conexão direta quando o pool estiver indisponível | P0 | Configurável por política; alerta emitido |

### Épico E6 — DLP e governança de dados

| ID | Requisito | Pri. | Critério de aceite |
|---|---|---|---|
| FR-29 | Políticas DLP por tenant/rota: allow, redact, block, audit | P0 | Decisão registrada com regra, detector e hash; sem PII bruta no log |
| FR-30 | Detecção de PII/segredos no prompt e na resposta | P0 | CPF/CNPJ (com dígito), e-mail, cartão (Luhn), chaves AWS/JWT e telefone |
| FR-31 | DLP em streaming com janelas sobrepostas | P1 | Não vaza padrão que cruza chunk; latência adicional ≤ 60 ms p95 |
| FR-32 | Redaction reversível opcional (tokenização) | P2 | Rehidratação apenas por ator autorizado e auditada |
| FR-33 | Auditoria imutável de decisões e configurações | P0 | Append-only, retenção 1 ano, exportável |
| FR-34 | Prompt injection guard | P1 | Bloqueio/flag conforme política; falso positivo mensurável |

### Épico E7 — Tenancy, autenticação e quotas

| ID | Requisito | Pri. | Critério de aceite |
|---|---|---|---|
| FR-35 | API keys com hash Argon2id, escopos e expiração | P0 | Chave só é exibida na criação; revogação em ≤ 1 s |
| FR-36 | `tenant_id` em todas as entidades e namespaces | P0 | Consultas e caches sempre filtram tenant; teste negativo de vazamento |
| FR-37 | Quotas por tenant/chave/modelo (RPS, tokens, custo) | P0 | 429 com `Retry-After`; bloqueio em 100% do orçamento |
| FR-38 | Papéis: admin, operador, auditor, cliente | P1 | RBAC aplicado em todas as rotas administrativas; teste IDOR |
| FR-39 | JWT de sessão para console + OIDC/SAML (P2) | P1/P2 | Cookie httpOnly, CSRF, expiração e rotação |

### Épico E8 — Observabilidade e operação

| ID | Requisito | Pri. | Critério de aceite |
|---|---|---|---|
| FR-40 | Métricas golden signals por provider/modelo/tenant | P0 | Backend gerenciado (Datadog/Azure Monitor) com latência, tráfego, erros e saturação |
| FR-41 | Tracing distribuído por requisição e tentativa | P0 | Span do gateway + tentativa de provider + cache + DLP |
| FR-42 | Logs estruturados sem segredos/PII | P0 | Redaction validada por teste automatizado |
| FR-43 | Explorador de requisições com retenção configurável | P1 | Busca por request-id/tenant/modelo; payload off por padrão |
| FR-44 | Health/readiness/liveness e drain gracioso | P0 | Réplica sai do balanceador (drain) sem derrubar SSE em andamento |

### Épico E9 — Custo e relatórios

| ID | Requisito | Pri. | Critério de aceite |
|---|---|---|---|
| FR-45 | Custo por requisição, modelo, provider, chave e tenant | P0 | Tabela de preços configurável; custo estimado e reconciliado |
| FR-46 | Relatórios diários/mensais e exportação CSV/JSON | P1 | Fechamento diário com ≤ 5 min de atraso |
| FR-47 | Simulador de economia por cache | P2 | Comparação “com/sem cache” por período |

### Épico E10 — Corporativo, performance e lazy loading

| ID | Requisito | Pri. | Critério de aceite |
|---|---|---|---|
| FR-48 | Lazy loading de rotas, gráficos, editores e tabelas grandes no console | P0 | Nenhum chunk opcional no primeiro paint; skeleton e fallback de erro |
| FR-49 | Inicialização sob demanda de adaptadores, catálogo de modelos, embeddings e pools de proxy | P0 | Caminho quente inicializa só o essencial; init lazy ≤ 50 ms |
| FR-50 | Cache em camadas: L1 in-process, L2 Redis e cache HTTP para estáticos | P0 | Hit L1 ≥ 5%; L2 ≥ 25%; invalidação por pub/sub |
| FR-51 | SSO corporativo OIDC/SAML e MFA | P1/P2 | Login SSO, logout, mapeamento de papéis e auditoria |
| FR-52 | SCIM/sync de diretório e RBAC granular | P2 | Provisionamento/desprovisionamento automático; teste de IDOR |
| FR-53 | Exportação para SIEM e chargeback por centro de custo | P1 | Exportação assinada, agendada e auditada |
| FR-54 | Suporte a on-premise/air-gapped com registry privado e imagens assinadas | P1 | Instalação sem dependência de SaaS externo (exceto providers) |

### Épico E11 — Endpoints e API Manager

| ID | Requisito | Pri. | Critério de aceite |
|---|---|---|---|
| FR-55 | Catálogo de endpoints (`/v1/chat/completions`, `/v1/models`, `/v1/embeddings`, `/v1/responses`) com status, versão e deprecação | P0 | Endpoint desligado retorna 404/403 sem vazar detalhes; mudança auditada |
| FR-56 | Política por endpoint: categoria, auth, escopos, rate limit, modelos, cache, DLP, proxy, body/timeout, CORS e streaming | P0 | Configuração aplicada em ≤ 5 s; simulação “pode chamar?” |
| FR-57 | API keys com escopos de endpoint/modelo (apenas modelos ativos de provedores conectados), quota, rate limits (RPM/TPM/USD), expiração **ou “sem expiração”**, IP allowlist, max sessions, access schedule e rotação | P0 | Segredo exibido só na criação; rotação/revogação em ≤ 1 s; `expires_at` nulo permitido pela política |
| FR-58 | Centro de custo e uso por chave, **sem dono/tenant na UI do P0** | P0 | Uso e custo atribuídos ao centro de custo |
| FR-59 | Uso por endpoint e por chave (requests, tokens, custo, erros) | P0 | Relatório por período e exportação |
| FR-60 | Aprovação e auditoria de criação/rotação/revogação de chave e mudança de política | P1 | Workflow de aprovação e trilha imutável |
| FR-61 | Deprecação/versionamento de endpoint com aviso a consumidores | P1 | Header de deprecação e data publicada |
| FR-62 | OpenAPI/docs por endpoint e importação de especificação | P1 | Spec versionada e docs geradas |
| FR-63 | Grupos de chaves com regras allow/deny por modelo/provider | P1 | Deny tem precedência; chave sem grupo é irrestrita |
| FR-64 | Página “Endpoint final”: URLs pública/local, servidor, catálogo de endpoints disponíveis, aliases para CLIs e system prompt opcional | P0 | Copiar URL funciona; catálogo agrupado; sem túneis |

### Épico E12 - MCP

| ID | Requisito | Pri. | Critério de aceite |
|---|---|---|---|
| FR-65 | Servidor MCP **desligado por padrão** com ativação/desativação global auditada | P0 | Estado refletido no status; ativar/desativar em ≤ 1 s |
| FR-66 | Transportes `stdio`, `SSE` e `Streamable HTTP`, com comando/URL copiável por transporte | P0 | Seleção persiste; endpoint/comando correto; conexão testada |
| FR-67 | Catálogo de ferramentas com escopo e switch individual; ferramenta desativada não é anunciada | P0 | Chamada a ferramenta off retorna erro de escopo |
| FR-68 | Status do processo (PID, uptime, heartbeat), atividade 24h e detalhes de runtime | P0 | Dados atualizados ao abrir; restart disponível |
| FR-69 | MCP **local-only** (stdio/loopback) ou autenticado por Bearer/escopo; **sem túneis** | P0 | Acesso externo sem credencial retorna 401; nenhuma opção de túnel |

### Épico E13 - i18n e guia do usuário

| ID | Requisito | Pri. | Critério de aceite |
|---|---|---|---|
| FR-70 | Interface em `en`, `pt-PT`, `pt-BR`, `es`, `fr` e `de`, com fallback obrigatório para inglês | P0 | Troca de locale sem reload; chave ausente não quebra a UI |
| FR-71 | Onboarding no primeiro acesso com idioma detectado; preferência em `dontstop.locale` e Settings para troca | P0 | Escolha persiste; onboarding não reaparece; `<html lang>` atualizado |
| FR-72 | Guia versionado por página/modal, mesmo slug em seis idiomas, com imagem, `alt`, passos, campos e erros | P0 | 100% do inventário coberto; índices e manifest atualizados |
| FR-73 | Link **Ajuda** contextual com URL por locale/slug e fallback inglês → índice | P0 | Link abre em nova aba; href muda ao trocar idioma |

## 7. Requisitos não funcionais

| ID | Categoria | Meta mensurável | Como medir |
|---|---|---|---|
| NFR-01 | Concorrência | 20.000 streams SSE simultâneos no cluster; 5.000 por réplica | k6/ghz com SSE; métricas de conexões ativas |
| NFR-02 | Throughput | 1.000 req/s sustentado; 3.000 req/s por 60 s | Teste de carga com mix 70% streaming |
| NFR-03 | Latência de overhead | p95 ≤ 30 ms; p99 ≤ 80 ms | Histograma OTel no backend gerenciado |
| NFR-04 | TTFB/TTFT | TTFB p95 ≤ 800 ms; TTFT p95 ≤ 1,5 s (upstream-dependente) | Medição ponta a ponta |
| NFR-05 | Disponibilidade | 99,9% mensal; erro orçamento 43,2 min | SLO + alerta de burn rate |
| NFR-06 | Failover | RTO ≤ 2 s; RPO 0 (config) e ≤ 60 s (logs) | Chaos test mensal |
| NFR-07 | Cache | hit exato ≥ 25%, semântico +15%, total ≥ 40% | Métricas de cache por tenant/modelo |
| NFR-08 | Economia | −30% de custo vs. baseline | Relatório de custo com/sem cache |
| NFR-09 | Rate limit | p99 ≤ 3 ms; precisão ±1% | Teste de concorrência no Redis |
| NFR-10 | DLP | p95 ≤ 20 ms (regex), ≤ 60 ms (NER); bloqueio ≤ 50 ms | Histograma por estágio |
| NFR-11 | Banco | pool ≤ 50 conexões por réplica; PgBouncer; sem N+1 | `pg_stat_activity`, tracing |
| NFR-12 | Redis | pool ≤ 50 por réplica; operações p99 ≤ 2 ms | Latência do cliente |
| NFR-13 | Escalabilidade | ganho ~linear até 4× réplicas; autoscaling em 70% do alvo | Teste de escala |
| NFR-14 | Recuperação | réplica reinicia em ≤ 30 s; região em ≤ 15 min (P2) | Game day |
| NFR-15 | Segurança | TLS 1.3; AES-256-GCM; Argon2id; zero segredo em log | Scan + teste de redaction |
| NFR-16 | Retenção | metadados 90 d/1 ano; auditoria 1 ano; payload off | Política por tabela |
| NFR-17 | Observabilidade | 100% das requisições com trace-id; amostragem 10%; backend gerenciado (Datadog/Azure Monitor) | OTel exporter + backend |
| NFR-18 | Compatibilidade | SDK OpenAI oficial funciona sem patch | Teste de contrato |
| NFR-19 | Qualidade | cobertura ≥ 70% nas linhas críticas do gateway | Relatório de cobertura no CI |
| NFR-20 | Performance de UI | LCP ≤ 2,5 s; INP ≤ 200 ms; CLS ≤ 0,1; bundle inicial ≤ 250 KB gzip | Lighthouse/Web Vitals + bundle analyzer |
| NFR-21 | Lazy loading | nenhum módulo opcional no primeiro paint; init sob demanda ≤ 50 ms no caminho quente | Teste de cold start + tracing |
| NFR-22 | Observabilidade corporativa | sem PII em labels; cardinalidade controlada; retenção configurável | Auditoria de labels + política do backend |

## 8. SLAs e SLOs

| Item | Compromisso |
|---|---|
| Disponibilidade mensal | 99,9% |
| Latência de controle (admin) | p95 ≤ 300 ms |
| Failover quando há alternativa saudável | ≤ 2 s, sem erro 5xx ao cliente |
| Suporte a incidentes | P1: 15 min; P2: 2 h; P3: 1 dia útil |
| Janela de manutenção | comunicada com 48 h; fora de horário comercial |
| RPO de configuração | 0 (Postgres com replicação síncrona) |
| RPO de uso/auditoria | ≤ 60 s (buffer NATS/Redis) |

## 9. Limites de capacidade (P0)

| Recurso | Limite inicial | Observação |
|---|---|---|
| Streams SSE por réplica | 5.000 | 2 GiB de memória por réplica |
| Streams SSE por cluster | 20.000 | 4 réplicas mínimas |
| RPS por cluster | 1.000 (3.000 burst) | autoscaling por conexões/RPS |
| Corpo de requisição | 2 MB padrão / 16 MB máximo | Configurável por rota |
| Providers | 50 no P0, 500 no P1 | Catálogo versionado |
| Contas por provider | 50 | Rotação e health |
| API keys | 10.000 | Hash + escopo |
| Tenants | 1 (P0), 1.000 (P2) | `tenant_id` desde o P0 |
| Cache exato | 1M entradas / 16 GB Redis | TTL 1 h padrão |
| Cache semântico | 2M vetores / 16 GB Qdrant | 384 dimensões |
| Retenção de metadados | 90 d quente / 1 ano frio | Partições mensais |
| Payload de request/response | desligado por padrão | Opt-in por tenant |

## 10. Segurança e compliance

- **Autenticação:** API keys (Argon2id, prefixo identificável, últimos 4 caracteres visíveis), JWT httpOnly para console; OIDC/SAML + MFA no P1/P2.
- **Autorização:** RBAC + política por provider/modelo; toda rota administrativa exige papel; testes de IDOR obrigatórios.
- **Segredos:** AES-256-GCM com envelope key (KMS/Vault); nunca em logs, traces ou cache.
- **Rede:** SSRF guard (IP privado/metadata), allowlist de domínios upstream, egress via Webshare, TLS 1.3, HSTS.
- **DLP:** decisão auditada; fail-closed para segredos de alta confiança; mascaramento por padrão.
- **Auditoria:** append-only, hash encadeado, exportável; acesso restrito ao papel auditor.
- **LGPD:** base legal por tenant, retenção configurável, direito de exclusão por `tenant_id`/`subject_id` (P1), DPA com provedores.
- **Supply chain:** lockfile, SBOM, `govulncheck`/`npm audit`, gitleaks, Trivy, CodeQL, imagens assinadas (cosign).

### 10.1 Requisitos corporativos

| ID | Requisito | Pri. | Critério de aceite |
|---|---|---|---|
| CR-01 | SSO corporativo (OIDC/SAML) e MFA para o console | P1/P2 | Login/logout, mapeamento de grupos e sessão auditada |
| CR-02 | RBAC granular (admin, operador, FinOps, auditor, cliente) | P1 | Permissões aplicadas por rota; teste IDOR |
| CR-03 | Trilha de auditoria imutável e exportável para SIEM | P0/P1 | Hash encadeado; exportação assinada |
| CR-04 | Data residency e retenção configurável por tenant | P1 | Dados em região definida; TTL aplicado |
| CR-05 | Suporte a on-premise/air-gapped | P1 | Instalação sem SaaS externo (exceto providers) |
| CR-06 | Registry privado, imagens assinadas e SBOM | P0 | `cosign verify` + SBOM no pipeline |
| CR-07 | Chargeback por centro de custo/time | P1 | Relatório mensal reconciliado |
| CR-08 | Segregação de funções e aprovação para ações críticas | P1 | Aprovação em mudanças de política/credencial |
| CR-09 | DLP e classificação de dados alinhados à LGPD | P0 | Regras, auditoria e retenção |
| CR-10 | SLA de suporte e disponibilidade corporativa | P1 | 99,9%; P1 em 15 min |
| CR-11 | Observabilidade sem operar stack própria | P0 | OTel → backend gerenciado; sem Grafana/Prometheus |
| CR-12 | Acessibilidade WCAG 2.2 AA no console | P0 | Auditoria axe + testes de teclado |

## 11. Multi-tenancy — ganchos estruturais no P0

| Gancho | Implementação no P0 | Evolução P1/P2 |
|---|---|---|
| Identidade | `tenant_id` no contexto e em todas as tabelas | RLS por sessão (P1) + claims OIDC (P2) |
| API keys | Chave vinculada a um tenant | Escopos e cotas por tenant |
| Cache | Prefixo `t:{tenant}:` no Redis; payload `tenant_id` no Qdrant | Coleção por tenant para isolamento físico |
| Quotas | Limites por chave e por tenant | Orçamento e faturamento por tenant |
| Políticas | Regras com `tenant_id` e escopo global | Herança organização → tenant → time |
| Auditoria | Evento sempre com `tenant_id` | Exportação e retenção por tenant |
| Banco | Coluna + índice em todas as tabelas raiz | RLS (P1), partições por tenant se necessário (P2) |
| Segredos | Uma chave de envelope por instância | Chave por tenant (KMS) |

## 12. Critérios de aceite por épico

| Épico | Critério de aceite (gate) |
|---|---|
| E1 API | SDK OpenAI oficial passa em teste de contrato; SSE com cancelamento sem vazamento |
| E2 Providers | Cadastro + validação + criptografia + catálogo em ≤ 5 s; teste de credencial inválida |
| E3 Resiliência | Chaos test: derrubar conta/modelo/provider e obter sucesso com alternativa em ≤ 2 s |
| E4 Cache | Teste de hit exato e semântico com isolamento de tenant; invalidação em ≤ 5 s |
| E5 Proxy | Pool Webshare com health, rotação e quarentena; falha do pool não derruba o tráfego |
| E6 DLP | Testes negativos para PII/segredos; zero PII em log; decisão auditada |
| E7 Tenancy | Teste negativo de vazamento entre tenants em API, cache e banco |
| E8 Observabilidade | Trace completo de uma requisição com spans de cache, DLP e tentativa |
| E9 Custo | Custo por requisição reconciliado com a tabela de preços em ≤ 1% |
| E10 Corporativo | LCP ≤ 2,5 s, INP ≤ 200 ms, CLS ≤ 0,1; nenhum chunk opcional no primeiro paint |
| E11 Endpoints/API Manager | Escopo mínimo por endpoint/modelo ativo; falha fechada antes do upstream; sem dono na UI do P0 |
| E12 MCP | Desligado por padrão; transporte/ferramentas persistem; sem túnel e sem segredo em log |
| E13 i18n/guia | Seis locales com fallback; onboarding e Settings persistem; guia completo com links contextuais |

## 13. Premissas, lacunas e decisões pendentes

- **Premissa:** não há código no `dontstop`; todos os módulos são propostas.
- **Premissa:** volumes são alvos; a primeira campanha de carga deve validar ou corrigir as metas.
- **Lacuna:** plano Webshare, banda e quantidade de proxies.
- **Lacuna:** stack e tamanho do time; ADR-001 prevê fallback TypeScript.
- **Lacuna:** cloud/on-prem, região e orçamento de infraestrutura.
- **Lacuna:** compliance além da LGPD.
- **Pendente:** provedores iniciais, política de retenção de payload, modelo de embeddings do cache semântico.

## 14. Fontes consultadas

- `[REPO]` — estado greenfield de `D:\dev\dontstop`.
- `[MAP-9R]` — fallback em dois níveis, combos, tradução, quota multitenant, mutex global e segredos em texto plano.
- `[MAP-OMNI]` — 380 provedores, circuit breaker/cooldown/model lockout, `provider.use`, route guard, Redis opcional, volume de testes e dívidas de escala.
- `[MAP-ALLCODE]` — catálogo/plugins, protocolos, adaptadores proprietários, rotação sem feedback e instalação em runtime.
