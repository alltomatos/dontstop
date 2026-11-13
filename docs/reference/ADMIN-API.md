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

### `PATCH /admin/v1/providers/{id}`

Liga/desliga o default de proxy do provedor (`proxy_auto`). Contas com `proxy_mode=inherit` seguem
esse default: `auto` quando ligado, `off` quando desligado (ADR-024).

```json
{ "proxy_auto": true }
```

Retorna `{ "proxy_auto": true }`. Campo ausente retorna `400`; provedor inexistente retorna `404`.

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

### `DELETE /admin/v1/providers/{id}/accounts/{accountId}`

Remove uma conta. Se for a última conta do provedor, o provedor também é removido (com seus
modelos) para sair da lista de provedores configurados.

```json
{"deleted": true, "provider_deleted": true}
```

Resposta `404` quando a conta não existe. O console usa essa rota no botão "Remover conta" e
redireciona para `/providers` quando `provider_deleted` é `true`.

### `PATCH /admin/v1/providers/{id}/accounts/{accountId}`

Atualiza o modo de proxy da conta (`inherit`, `off`, `manual`, `auto`; `preferred` e `required` são
aceitos como legado e normalizados para `manual`). `manual` e `auto` são fail-closed (sem fallback
direto) e `auto` é gerenciado pelo gateway com IPs do Webshare (ADR-024).

```json
{ "proxy_mode": "manual" }
```

Retorna `{ "proxy_mode": "manual" }`. Valor inválido retorna `400`.

### `GET /admin/v1/proxies`

Lista o pool de proxies do tenant sem credenciais:

```json
{
  "object": "list",
  "data": [{
    "id": "…",
    "label": "Webshare SP 01",
    "address": "203.0.113.10",
    "port": 8080,
    "country_code": "BR",
    "city_name": "São Paulo",
    "source": "manual",
    "valid": true,
    "health": "healthy",
    "assigned_count": 1
  }]
}
```

`health` é `healthy`, `quarantined` ou `unhealthy`. Credenciais nunca são retornadas.

### `POST /admin/v1/proxies`

Cadastra um proxy manual no tenant. `address`, `port` e, se houver autenticação, `username`/`password`.

```json
{
  "label": "Webshare SP 01",
  "address": "203.0.113.10",
  "port": 8080,
  "country_code": "BR",
  "city_name": "São Paulo",
  "username": "user",
  "password": "secret"
}
```

Endereço inválido retorna `400`; duplicidade retorna `409`. A senha é cifrada e não volta na resposta.

### `PUT /admin/v1/providers/{id}/accounts/{accountId}/proxy`

Atribui ou reatribui um proxy à conta, limitado a **3 contas por proxy**:

```json
{ "proxy_endpoint_id": "…" }
```

Proxy/conta inexistente ou de outro tenant retorna `404`; limite retorna `409`.

### `DELETE /admin/v1/providers/{id}/accounts/{accountId}/proxy`

Remove a atribuição da conta. O modo `required` passa a falhar fechado até receber um proxy saudável.

### `GET /admin/v1/proxies/webshare`

Retorna o estado sanitizado da integração: `configured`, `plan_id`, `enabled`, `key_version`,
`last_sync_at`, `last_sync_status`, `last_sync_error`, `last_sync_total`, `last_sync_imported`,
`proxy_count` e `healthy_count`. A API key nunca é retornada.

### `POST /admin/v1/proxies/webshare/test`

Testa a conexão real com o Webshare antes de salvar.

```json
{ "api_key": "…", "plan_id": "opcional" }
```

Retorna `{ ok, plan_id, state, available }`. Erros: `400 webshare_unauthorized`,
`400 webshare_forbidden`, `429 webshare_rate_limited`, `502 webshare_unavailable`.

### `PUT /admin/v1/proxies/webshare`

Valida e persiste a credencial cifrada (AES-256-GCM, `key_version`). Envie `api_key` vazio para
manter a chave atual; `plan_id` é opcional quando existe apenas um plano.

```json
{ "api_key": "…", "plan_id": "…", "enabled": true }
```

Retorna o mesmo status sanitizado de `GET`. A operação é auditada em `audit_outbox`.

### `POST /admin/v1/proxies/webshare/sync`

Importa/sincroniza os proxies `direct` do plano, aplicando diff idempotente. Proxies ausentes ficam
`valid=false`, mas atribuições em uso não são removidas. Retorna
`{ imported, total, status }`. Retorna `409 sync_in_progress` quando outro worker está sincronizando.

### `POST /admin/v1/proxies/webshare/refresh`

Re-sincroniza após uma falha de IP, com debounce de 30 s por tenant (ADR-024). É no-op quando o
Webshare não está configurado. Retorna `{ "ok": true }`.

### `GET /admin/v1/providers/{id}`

Além do resumo consolidado (`models_count`, `models_usable`, `usable_accounts`), cada item de
`accounts` traz `models_available`, `models_usable`, `capabilities`, `health` e `active`.
`health` é `active`, `cooldown`, `quarantined`, `unhealthy` ou `disabled`; contas em
cooldown/quarentena não entram em `usable_accounts` e têm `models_usable=0`.

### `GET /admin/v1/providers/{id}/accounts/{accountId}/summary`

Retorna o agregado do provedor e o resumo explícito da conta selecionada. Conta de outro
tenant/provedor retorna `404`.

### `POST /admin/v1/providers/{id}/accounts/{accountId}/test`

Testa somente a conta informada, usando a credencial, URL e configuração dela. Retorna
`account_id`, `account_name`, `ok`, `status`, `latency_ms`, `model_count` e `error`.
Conta de outro tenant/provedor retorna `404`. Não altera o provedor nem outras contas.

### `POST /admin/v1/providers/{id}/models/test`

Testa o modelo na conta escolhida:

```json
{ "model_id": "deepseek-v4.1-flash", "account_id": "…" }
```

Sem `account_id`, usa a primeira conta ativa. A resposta inclui o agregado e `accounts[]` com
`account_id`, `account_name`, `ok`, `status`, `latency_ms` e `error`. Conta inválida ou de
outro provedor retorna `404`.

### `POST /admin/v1/providers/{id}/test`

Testa a conta ativa de maior prioridade contra o endpoint de modelos do provedor. Retorna `200` mesmo quando o upstream falha, com `ok=false` e uma mensagem sanitizada.

```json
{"ok":true,"status":200,"latency_ms":874,"model_count":425}
```

### `GET /admin/v1/providers/{id}/models`

Lista os modelos anunciados pelo provedor, sem persistir nada.

```json
{"object":"list","data":[{"id":"openai/gpt-4o-mini","name":"OpenAI: GPT-4o mini","owned_by":"openai"}]}
```

### `POST /admin/v1/providers/{id}/models/import`

Persiste apenas modelos que o provedor realmente anunciou. IDs desconhecidos são ignorados; modelos existentes são reativados e renomeados.

```json
{"models":["openai/gpt-4o-mini","anthropic/claude-sonnet-4"]}
```

Resposta: `{"imported":2}`.

Ao criar o provedor (`POST /providers` e `POST /provider-catalog/{slug}/connect`), o gateway já importa
automaticamente todos os modelos anunciados pelo endpoint do provedor. A falha nessa importação não
impede a criação; ela pode ser repetida manualmente.

### `GET /admin/v1/providers/{id}/models/configured`

Lista os modelos persistidos com o estado de roteamento.

```json
{"object":"list","data":[{"model_id":"gpt-4.1","display_name":"GPT-4.1","active":true,"capabilities":{}}]}
```

### `POST /admin/v1/providers/{id}/models/sync`

Importa tudo o que o provedor anuncia, preservando o estado ativo/inativo dos modelos que já existem.

```json
{"imported":12,"total":12}
```

Se o provedor não expõe `/models` (ex.: `perplexity`), responde `200` com `error` descritivo.

### `PATCH /admin/v1/providers/{id}/models`

Ativa ou desativa um modelo. Modelos inativos saem de `/v1/models` e do roteamento.

```json
{"model_id":"gpt-4.1","active":false}
```

### `PATCH /admin/v1/providers/{id}/models/bulk`

Ativa ou desativa vários modelos em uma chamada, sem abortar os demais em caso de erro parcial.
O `PATCH /models` individual continua funcionando.

```json
{"models":["gpt-4.1","gpt-4o-mini","missing-model"],"active":false}
```

```json
{"updated":2,"results":[
  {"model_id":"gpt-4.1","ok":true},
  {"model_id":"gpt-4o-mini","ok":true},
  {"model_id":"missing-model","ok":false,"error":"model not found"}
]}
```

### `POST /admin/v1/providers/{id}/models/test`

Envia uma requisição mínima de chat com a conta ativa de maior prioridade. Retorna `200` mesmo quando o
upstream falha, com `ok=false` e mensagem sanitizada.

```json
{"ok":true,"status":200,"latency_ms":42}
```

### `POST /admin/v1/providers/{id}/models/test-all`

Enfileira o teste de vários modelos. O processamento é feito por um worker no gateway, um modelo por
vez, com lease no banco — a requisição não bloqueia a UI. Um provedor tem no máximo uma execução ativa:
chamar de novo devolve a execução existente.

```json
{"models":["gpt-4.1","gpt-4o-mini"]}
```

Corpo vazio testa todos os modelos persistidos. Resposta `202` com a execução (`status=queued`).

### `GET /admin/v1/providers/{id}/models/test-runs/{runId}`

Retorna a execução com progresso e resultado por modelo (`pending`, `running`, `ok`, `failed`).

### `GET /admin/v1/providers/{id}/models/test-runs/latest`

Retorna a execução mais recente do provedor (`404` se nunca houve teste).

## Catálogo versionado de provedores

O catálogo é global (não por tenant) e versionado por `catalog_version`. Ele é a fonte de verdade de
protocolo, base URL, path de modelos, header/scheme de autenticação, headers fixos, capabilities,
rede e quirks. Ao conectar por `slug`, o operador informa apenas a credencial.

Provedores criados por catálogo gravam `catalog_slug` em `providers` e o `kind` do adapter é
derivado do protocolo (`openai` → `openai`, `anthropic` → `anthropic`).

### `GET /admin/v1/provider-catalog`

Lista as entradas habilitadas com os modelos declarados.

```json
{
  "object": "list",
  "data": [
    {
      "slug": "openrouter",
      "display_name": "OpenRouter",
      "protocol": "openai",
      "default_base_url": "https://openrouter.ai/api/v1",
      "models_path": "/models",
      "auth_type": "api_key",
      "auth_header": "Authorization",
      "auth_scheme": "bearer",
      "capabilities": {"stream": true, "tools": true, "vision": true, "reasoning": true},
      "network": {"proxy_policy": "optional", "private_host": false},
      "quirks": ["dynamic_catalog", "optional_http_referer_and_x_title"],
      "source_ref": "omniroute@3.8.53+9router@0.5.112+allcode@1.27.6",
      "catalog_version": "2026-11",
      "models": [
        {"model_id": "auto", "context_window": 200000, "supports_tools": true,
         "input_price_per_million": 2, "output_price_per_million": 8,
         "pricing_source": "9router:pricing.js@0.5.112"}
      ]
    }
  ]
}
```

### `POST /admin/v1/provider-catalog/{slug}/connect`

Cria provedor, conta e modelos do catálogo em uma transação. `name`, `account_name` e `base_url` são
opcionais (default: nome do catálogo, `default` e a base do catálogo). `base_url` só aceita
`http`/`https`. Conflito de nome retorna `409`; `slug` inexistente retorna `404`.

```json
{"credential": "sk-or-...", "name": "OpenRouter", "account_name": "produção"}
```

Resposta: `201` com o provedor criado (inclui `catalog_slug`).

Provedores cujo catálogo declara `models_path` vazio (ex.: `perplexity`) não expõem endpoint de
modelos: `GET /providers/{id}/models` e o import respondem `400`, e `POST /providers/{id}/test`
retorna `ok=false` com a causa.

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
- As rotas `/api/admin/*` do Next repassam o token da sessão do console (cookie `dontstop_session`); o gateway valida sessão ou API key com escopo `manage`. Cookie forjado resulta em 401/403.
- O layout `(console)` revalida a sessão em `/admin/v1/auth/me`; presença de cookie não é autenticação.
- `POST /admin/v1/auth/login` aplica janela deslizante de 10 falhas por e-mail em 5 minutos e responde 429 com `Retry-After`.
- `POST /admin/v1/setup` é serializado por advisory lock transacional; chamadas concorrentes resultam em um `201` e um `409` (apenas um owner por instância).
- Os testes negativos ficam em `tests/security/*.spec.sec.ts` e rodam com `npm run test:sec` a partir de `tests/security`.

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

## Combos (planejado)

O contrato completo de combos — CRUD, membros, ativação, validação, teste e criação assistida por
IA — está em [`COMBOS.md`](COMBOS.md). A Admin API de combos segue o mesmo padrão de autenticação,
`tenant_id` derivado do principal e erros sanitizados descrito neste documento.
