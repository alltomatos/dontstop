# Admin API — providers

API usada pelo console Next.js para sair dos dados mockados.

## Autenticação

- `Authorization: Bearer <api-key>`
- A chave precisa do escopo `manage`.
- O console nunca envia a chave ao navegador: o servidor Next lê `DONTSTOP_ADMIN_API_KEY` e chama o gateway.

## Endpoints

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
      "origin": "9router",
      "enabled": true,
      "accounts_count": 3,
      "models_count": 12
    }
  ]
}
```

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
