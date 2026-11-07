# Planejamento do dontstop — índice

> **Status:** documentação de arquitetura e produto. Nenhum código foi implementado nesta etapa.
> **Data:** 2026-11-06
> **Repositório:** `D:\dev\dontstop` (greenfield: 0 arquivos de código; apenas os mapeamentos e este planejamento).

## 1. O que é o dontstop

Gateway de IA corporativo auto-hospedado para centralizar o consumo de LLMs em nuvem e locais, com foco em:

- alta concorrência (milhares de conexões simultâneas, incluindo SSE);
- otimização financeira com cache exato e semântico;
- resiliência com failover automático entre múltiplas contas de provedores;
- pools de proxy de rede (Webshare);
- governança de dados e DLP.

A arquitetura inicial é **single-tenant (instância única)**, mas o modelo de dados, autenticação, cache, quotas e políticas já nascem com `tenant_id` e isolamento lógico para permitir multi-tenancy sem reescrita.

## 2. Documentos

| Ordem | Documento | Conteúdo |
|---|---|---|
| 00 | [Sumário executivo](00-SUMARIO-EXECUTIVO.md) | Decisões principais, metas, riscos críticos e o que ainda está em aberto |
| 01 | [PRD](01-PRD.md) | Escopo, personas, requisitos funcionais/não funcionais, SLAs, limites e critérios de aceite |
| 02 | [DER](02-DER.md) | Entidades, relacionamentos, cardinalidades, chaves, índices, RLS e retenção |
| 03 | [UML](03-UML.md) | Componentes, sequências (request, cache, failover, proxy, DLP), classes e implantação |
| 04 | [Análise de stack](04-ANALISE-STACK.md) | Comparação de alternativas, cálculos de capacidade e recomendação única |
| 05 | [ADRs](05-ADRS.md) | 18 Architecture Decision Records com contexto, alternativas e consequências |
| 06 | [Roadmap e riscos](06-ROADMAP-RISCOS.md) | Fases P0/P1/P2, critérios de saída, riscos, mitigação e decisões pendentes |
| 07 | [i18n e guia do usuário](07-I18N-E-USER-GUIDE.md) | Idiomas, preferência, fallback, links contextuais e estrutura do guia versionado |
| — | [ADD](architecture/ADD.md) | Documento consolidado de arquitetura (C4, NFRs, stack, segurança, deploy Swarm) |
| — | [Backlog](BACKLOG.md) | Slices verticais prontos para Issues, com dependências e critérios de aceite |
| — | [DEVELOPER-ROADMAP](../DEVELOPER-ROADMAP.md) | Epic E1 e slices P0-α com links para as Issues no GitHub |
| — | [Contrato OpenAPI v1](contracts/openapi-v1.yaml) | Contrato público de `/v1/models` e `/v1/chat/completions` (JSON + SSE) |
| — | [QA P0-α E1.1–E1.4](qa/p0-alpha-e1.1-e1.4.md) | Plano, casos executados, auditoria de segurança e riscos residuais |`r`n| — | [QA P0-α E1.6](qa/p0-alpha-e1.6.md) | Roteamento, failover, circuit breaker, isolamento e riscos residuais |`r`n| — | [QA P0-α E1.12](qa/p0-alpha-e1.12.md) | Credenciais, criptografia, endpoint e riscos residuais |`r`n| — | [QA P0-α E1.5](qa/p0-alpha-e1.5.md) | Chat JSON/SSE, idempotência, validação e riscos residuais |`r`n| — | [QA P0-α E1.7](qa/p0-alpha-e1.7.md) | Cache exato, TTL, invalidação, isolamento e riscos residuais |`r`n| — | [QA P0-α E1.8](qa/p0-alpha-e1.8.md) | Usage/custo, outbox, NATS, idempotência e riscos residuais |`r`n| — | [QA P0-α E1.9](qa/p0-alpha-e1.9.md) | OTel, spans, SLOs, carga 100 RPS e riscos residuais |`r`n| — | [SLOs P0-α](observability/P0-ALPHA-SLOS.md) | Sinais OTel, metas, consultas Datadog/Azure e exportação |
| — | [Design System](../design-system/MASTER.md) | Tokens, componentes e regras visuais (corporate navy/slate) |
| — | [Mockup do console](../design-system/mockups/console.html) | Mockup navegável: Dashboard, Providers, Endpoints, API Manager, MCP, Cache, DLP, Observabilidade, Custo, Configurações e onboarding |
| — | [Guia do usuário](user-guide/README.md) | Guias por página/modal em 6 idiomas, com imagens, alt e fallback |
| — | [Referência: Endpoints/API Manager](reference/ENDPOINTS-API-MANAGER-REFERENCIAS.md) | Como 9Router e OmniRoute tratam endpoints e API keys; o que adotar/evitar |
| — | [Referência: MCP](reference/MCP-REFERENCIAS.md) | Como 9Router e OmniRoute expõem MCP; decisões do dontstop |

## 3. Fontes consultadas

Os documentos citam explicitamente as seguintes fontes:

| Chave | Fonte | Uso |
|---|---|---|
| `[REPO]` | `D:\dev\dontstop` | Estado real do repositório: greenfield, sem código, sem commits |
| `[MAP-9R]` | [`../MAPEAMENTO-ARQUITETURAL-9ROUTER.md`](../MAPEAMENTO-ARQUITETURAL-9ROUTER.md) | Pontos fortes/fracos de um gateway OpenAI-compatible com fallback, RTK, SQLite e multitenancy |
| `[MAP-OMNI]` | [`../MAPEAMENTO-ARQUITETURAL-OMNIROUTE.md`](../MAPEAMENTO-ARQUITETURAL-OMNIROUTE.md) | Escala, resiliência, 380 provedores, 222 migrações, Redis opcional, governança e dívidas |
| `[MAP-ALLCODE]` | [`../MAPEAMENTO-PROVEDORES-ALLCODE.md`](../MAPEAMENTO-PROVEDORES-ALLCODE.md) | Como um runtime real lida com provedores: catálogo, adaptadores, protocolos, OAuth, rotação e lacunas |

> **Regra de honestidade:** tudo que não pôde ser verificado no repositório está marcado como **suposição**, **lacuna** ou **decisão pendente**. Números de capacidade são alvos de engenharia a validar com carga real, não medições do ambiente atual.

## 4. Premissas estruturais (explícitas)

1. **Não existe código de aplicação no `dontstop`.** O planejamento parte dos três mapeamentos e de requisitos de negócio; qualquer menção a módulos internos do dontstop é proposta, não código existente.
2. **Volumetria é alvo, não medição.** Os números de RPS/concorrência são premissas de capacidade a validar com teste de carga.
3. **Plano Webshare é desconhecido.** A análise assume um pool de 500–2.000 proxies e franquia de banda a confirmar; não há números de plano contratado no repositório.
4. **Nuvem/on-prem ainda não decididos.** A recomendação assume **Docker Swarm** em nuvem ou cluster próprio; Docker Compose é o caminho de desenvolvimento/P0. Kubernetes está fora de escopo por decisão do operador.
5. **Time e stack atuais desconhecidos.** A recomendação considera o histórico dos projetos analisados (TypeScript/Node), mas escolhe Go para o data plane por requisitos de concorrência; há um ADR com o fallback TypeScript caso o time seja 100% TS.
6. **Compliance assumida:** LGPD como baseline (dados de usuários no Brasil), com opção de retenção configurável e DLP opt-in por política. Outros frameworks (PCI, HIPAA) não foram declarados.
7. **Provedores iniciais:** OpenAI, Anthropic, Google Gemini, Azure OpenAI e Bedrock, além de endpoints OpenAI-compatible. O catálogo final é decisão de produto.

## 5. Convenções

- **Metas numéricas** aparecem como alvo (`≤`, `≥`, `p95`, `p99`).
- **Capacidade** é expressa por cluster e por réplica/task quando aplicável.
- **Prioridade** segue P0 (MVP), P1 (escala/controle) e P2 (evolução).
- **Multi-tenancy:** `tenant_id` é obrigatório em todas as entidades de dados; o tenant padrão é `00000000-0000-0000-0000-000000000000`.
- **Segredos:** nunca em texto plano; chaves de API com hash Argon2id; credenciais de provedores com AES-256-GCM e envelope de chave.

## 6. UI/UX e mockups

- **Design system:** [`design-system/MASTER.md`](../design-system/MASTER.md) — paleta navy/slate, Fira Sans/Fira Code, densidade de dashboard e WCAG 2.2 AA.
- **Specs por tela:** `design-system/pages/{dashboard,providers,endpoints,api-manager,mcp,cache,dlp,observability,cost}.md`.
- **Mockup navegável:** [`design-system/mockups/console.html`](../design-system/mockups/console.html) — abra no navegador e alterne as abas.
- **Regra:** nenhuma tela vai para código antes de o mockup/spec ser validado.

## 7. Glossário rápido

| Termo | Definição |
|---|---|
| Provider | Fornecedor de LLM (OpenAI, Anthropic, Gemini, Bedrock, endpoint compatível) |
| Account/Connection | Credencial individual de um provider (API key ou OAuth) |
| Route/Combo | Política de encadeamento de modelos/provedores com fallback |
| Exact cache | Cache por hash normalizado da requisição |
| Semantic cache | Cache por similaridade vetorial de embeddings |
| Circuit breaker | Estado de saúde por provider (closed/open/half-open) |
| Cooldown | Suspensão temporária de uma conta após erro |
| DLP | Data Loss Prevention: inspeção, redação ou bloqueio de dados sensíveis |
| Tenant | Unidade de isolamento lógico (empresa/organização); single-tenant no P0 |
| RLS | Row-Level Security do PostgreSQL |
