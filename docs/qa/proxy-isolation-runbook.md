# Runbook — Isolamento de proxy por provedor

> ADR-024. Complementa [`proxy-isolation.md`](proxy-isolation.md) e
> [`reference/WEBSHARE-PROXY-PLAN.md`](../reference/WEBSHARE-PROXY-PLAN.md).

## Sinais de alerta

| Métrica | Sintoma | Leitura |
|---|---|---|
| `proxy.candidate_miss` | subindo | pool sem IP elegível para o provedor |
| `proxy.poison` | pico | IPs sendo banidos/bloqueados |
| `proxy.refresh{status="failed"}` | > 0 | re-sync Webshare falhando |
| `proxy.rotate` | pico | rotações frequentes (instabilidade de IP) |

Contas em `manual`/`auto` falham **fechado** quando não há IP saudável; isso aparece como erro de
proxy no `chat`, não como conexão direta.

## Diagnóstico

1. `GET /admin/v1/proxies/webshare` → `proxy_count`, `healthy_count`.
2. `GET /admin/v1/proxies` → contas por IP e `banned_until`/`ban_reason`.
3. Banco (somente leitura):
   ```sql
   SELECT tenant_id, provider_id, proxy_endpoint_id, reason, banned_until
   FROM proxy_poison WHERE banned_until > now() ORDER BY banned_until DESC;
   ```
4. Confirmar se o provedor afetado tem IPs livres: contas do mesmo provedor não podem compartilhar IP.

## Ações

- **Repor o pool**: `POST /admin/v1/proxies/webshare/sync` (manual) ou
  `POST /admin/v1/proxies/webshare/refresh` (com debounce; re-sync orientado a falha).
- **Pool esgotado**: reduzir o número de contas do provedor, liberar atribuições órfãs ou ampliar o
  plano Webshare. Não relaxar a regra 1:1 por provedor.
- **Ban suspeito**: validar com o provedor; o poison expira sozinho (15 min / 1 h / 24 h). Não
  reatribuir manualmente o mesmo IP à mesma conta/provedor.
- **Reuso cross-provider**: permitido apenas para IPs não envenenados para aquele provedor.

## Emergência (quebra temporária do isolamento)

- `PATCH /admin/v1/providers/{id}/accounts/{accountId}` com `{"proxy_mode":"off"}` faz a conta sair do
  proxy (conexão direta). Use só como último recurso: expõe o IP real do gateway e pode agravar bans.
- Desabilitar o provedor interrompe o tráfego sem alterar o pool.

## Escalonamento

1. Verificar plano/limites do Webshare (`/admin/v1/proxies/webshare`).
2. Coletar `proxy.poison` + `banned_until` e o provedor afetado.
3. Se o pool não se recuperar após dois `sync` (com intervalo ≥ 60 s), abrir incidente de rede.
