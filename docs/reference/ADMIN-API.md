# Admin API — providers

API usada pelo console Next.js para sair dos dados mockados.

## Autentica??o

- `Authorization: Bearer <session-token>` para o console; a sess?o ? criada por `/admin/v1/auth/login` ou `/admin/v1/setup`.
- `Authorization: Bearer <api-key>` continua v?lido para automa??o com escopo `manage`.
- O cookie de sess?o do Next ? `httpOnly`, `SameSite=Strict`, com validade de 8 horas.

## Endpoints

### `GET /admin/v1/setup/status`

P?blico. Retorna `{"needs_setup": true}` quando ainda n?o existe usu?rio do console.

### `POST /admin/v1/setup`

P?blico e permitido apenas uma vez. Cria o propriet?rio com senha Argon2id e devolve a sess?o.

### `POST /admin/v1/auth/login`

P?blico. Autentica e-mail/senha e devolve a sess?o.

### `GET /admin/v1/auth/me`

Protegido. Confirma a sess?o e os escopos.

### `POST /admin/v1/auth/logout`

Protegido. Revoga a sess?o.

### `POST /admin/v1/providers`

Cria provedor, primeira conta e primeiro modelo em uma transa??o. O `origin` deve ser `native` ou `custom`.

### `GET /admin/v1/providers`

Lista provedores configurados do tenant autenticado.

```json
{
  "object": "list",
  "data": [
    {
      "id": "uuid",
      "name": "OpenAI",
      "kind": "openai",
      "origin": "native",
      "enabled": true,
      "accounts_count": 3,
      "models_count": 12
    }
  ]
}
```

`origin` é sempre `native` (catálogo embutido do dontstop) ou `custom` (cadastro do operador). Valores de gateways de referência não são aceitos.

### `GET /admin/v1/providers/{id}`

Retorna o provedor com `accounts` (sem credencial).

### `POST /admin/v1/providers/{id}/accounts`

Cria uma conta. A credencial é cifrada com AES-256-GCM antes do INSERT e nunca é retornada.

```json
{
  "name": "produção",
  "credential": "sk-...",
  "base_url": "https://api.openai.com",
  "priority": 100,
  "weight": 1
}
```

Resposta: `201` com `id`, `name`, `status`, `base_url`, `priority`, `weight` e `cooldown_until`.

## Integração com o frontend

`admin/lib/admin-api.ts` usa:

- `DONTSTOP_GATEWAY_URL`
- `DONTSTOP_ADMIN_API_KEY`

Se as variáveis não existirem ou o gateway falhar, o console usa o catálogo mock (`admin/lib/providers.ts`) para manter a UI navegável. O POST de conta passa por `admin/app/api/admin/providers/[id]/accounts/route.ts`, que também faz fallback local quando a Admin API não está configurada.

## Segurança

- `manage` é obrigatório; `models:read`/`chat:write` não autorizam administração.
- Credenciais são cifradas com o `MASTER_KEY` do gateway e nunca aparecem na resposta.
- O tenant vem do principal autenticado; o cliente não escolhe `tenant_id`.
- Erros de validação retornam 400; recurso de outro tenant retorna 404.

## API keys

### `GET /admin/v1/api-keys`

Lista chaves do tenant. O segredo nunca ? retornado; apenas `prefix`, escopos, expira??o, revoga??o e ?ltimo uso.

### `POST /admin/v1/api-keys`

```json
{"name":"prod-web","scopes":["chat:write","models:read"],"expires_at":"2027-03-01T00:00:00Z"}
```

Resposta `201` inclui `key` completo **uma ?nica vez**. O gateway persiste apenas Argon2id + prefixo.

### `POST /admin/v1/api-keys/{id}/revoke`

Marca `revoked_at` e retorna a chave sem segredo.

O console usa `admin/app/api/admin/api-keys/route.ts` e `.../[id]/revoke/route.ts` como BFF server-side. Sem Admin API configurada, a tela mant?m o cat?logo mock.

## Custos e observabilidade

### `GET /admin/v1/costs/summary?days=30`

Retorna `summary` (custo, requests, tokens), `by_provider`, `by_day` e `by_model` a partir de `usage_daily`.

### `GET /admin/v1/observability/summary?days=1`

Retorna requests, erros, taxa de erro, p50/p95/p99, custo e tokens a partir de `requests`.

O console usa esses endpoints quando a Admin API est? configurada e mant?m os mocks como fallback.

## MCP

### `GET /admin/v1/mcp/status`

Retorna `enabled`, `transport` (`stdio`/`sse`/`streamable-http`) e as 6 ferramentas com escopo e estado.

### `PATCH /admin/v1/mcp/settings`

```json
{"enabled":true,"transport":"streamable-http"}
```

### `PATCH /admin/v1/mcp/tools/{id}`

```json
{"enabled":false}
```

Persist?ncia em `mcp_settings` e `mcp_tool_settings`. Desligado por padr?o; toda altera??o exige `manage`.

## Cache

### `GET /admin/v1/cache/stats?days=1`

Retorna hits, misses, requests, hit rate e lat?ncia m?dia a partir de `requests.cache_status`.

### `GET /admin/v1/cache/events?limit=50`

Retorna eventos recentes sem payload: hor?rio, status, modelo, lat?ncia e tokens.

O console usa os endpoints quando configurado e mant?m o mock como fallback.

## DLP

### `GET /admin/v1/dlp/rules`

Lista regras com detector, a??o, severidade e estado.

### `POST /admin/v1/dlp/rules`

```json
{"name":"CPF","detector":"cpf","action":"redact","severity":"high"}
```

### `PATCH /admin/v1/dlp/rules/{id}`

```json
{"enabled":false}
```

### `GET /admin/v1/dlp/events?limit=50`

Eventos sem PII bruta: detector, a??o, severidade, request id e hash.

O console carrega regras/eventos da Admin API com fallback mock.

## Endpoints

### `GET /admin/v1/endpoints`

Cat?logo est?tico versionado de endpoints p?blicos, com categoria, path, modelos e estado.

### `GET /admin/v1/endpoints/{id}/policy`

Pol?tica padr?o: auth obrigat?ria, 120 RPM, cache e DLP habilitados, proxy desligado, body de 1 MiB e timeout de 60 s.

O console usa o cat?logo da Admin API com fallback local.
