# Architecture Decision Records (ADRs)

O log canônico de ADRs do dontstop é [`docs/05-ADRS.md`](../05-ADRS.md). Esta pasta existe para
indexar e, no futuro, hospedar ADRs autônomos; não duplique o conteúdo aqui sem atualizar a fonte.

## Índice

| ADR | Título |
|---|---|
| ADR-001 | Data plane em Go e control plane em TypeScript/Next.js |
| ADR-002 | PostgreSQL como system of record, com PgBouncer e partições |
| ADR-003 | Redis para cache exato, rate limit, locks e circuit breaker |
| ADR-004 | Qdrant para cache semântico |
| ADR-005 | NATS JetStream para eventos e jobs assíncronos |
| ADR-006 | OpenTelemetry + backend gerenciado (sem Prometheus/Grafana) |
| ADR-007 | Multi-tenancy por `tenant_id` + RLS |
| ADR-008 | Gerenciador de pool Webshare |
| ADR-009 | DLP em dois estágios |
| ADR-010 | Docker Swarm com réplicas e autoscaling externo |
| ADR-011 | Segredos com AES-256-GCM e envelope key |
| ADR-012 | Compatibilidade OpenAI como contrato primário |
| ADR-013 | Lazy loading e cache em camadas |
| ADR-014 | Identidade corporativa: OIDC/SAML, MFA e RBAC |
| ADR-015 | Endpoint Registry e escopos de API key |
| ADR-016 | MCP opt-in, local-only e sem túneis |
| ADR-017 | i18n sem dependência e guia versionado por página |
| ADR-018 | Envelope P0-α e primeiro slice vertical do gateway |
| ADR-019 | Origem nativa dos provedores e adaptação AgentRouter |
| ADR-020 | Origem configurável do guia do usuário (local → GitHub) |
| ADR-021 | Proxy Webshare dedicado e sticky por conta de provedor |
| ADR-022 | Failover padrão entre contas do mesmo provedor |
| ADR-023 | Combos nativos com failover na mesma requisição |
| ADR-024 | Isolamento de IP por provedor, modo `auto` e poison persistente |
| ADR-025 | Sidecar relay vendorizado para o AgentRouter (WAF de fingerprint) |
