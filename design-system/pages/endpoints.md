# Página — Endpoint final

> Override do [MASTER](../MASTER.md). Modelo visual inspirado no endpoint page do OmniRoute (e no do 9Router): URLs prontas, servidor local, catálogo de endpoints e aliases para CLIs.
> **Fora de escopo:** túneis (Cloudflare/Tailscale/ngrok) **não** fazem parte do dontstop.

## Objetivo

Entregar ao desenvolvedor o “copiar e colar” para integrar qualquer SDK/CLI, mostrar o catálogo de endpoints disponíveis e dar acesso rápido às políticas e ao API Manager.

## Wireframe (desktop ≥ 1024px)

```
┌──────────┬──────────────────────────────────────────────────────────────────────┐
│ Sidebar  │ Título: Endpoint final       [Testar endpoint] [Gerenciar políticas] │
│          ├──────────────────────────────────────────────────────────────────────┤
│ Dashboard│ ( APIs compatíveis com OpenAI )  Políticas   Aliases   Saúde         │
│ Providers│ ┌──────────────────────────────────────────────────────────────┐       │
│ Endpoints│ │ Endpoint da API                                              │       │
│ API Mgr  │ │ ENDPOINTS ATIVOS                                             │       │
│ Cache    │ │ ● Público  https://gateway.dontstop.corp/v1        [Copiar]  │       │
│ DLP      │ │ ● Local    http://localhost:20128/v1               [Copiar]  │       │
│ Custo    │ │ SERVIDOR LOCAL  0.0.0.0:20128  127.0.0.1:20128  [::]:20128  │       │
│ Observ.  │ │                  ● Running                                   │       │
│          │ │ SEGURANÇA E PROMPT                                           │       │
│          │ │ Require API key  ● Ativado          [Abrir API Manager]      │       │
│          │ │ System prompt    ○ Desligado        [Editar]                 │       │
│          │ └──────────────────────────────────────────────────────────────┘       │
│          │ ┌──────────────────────────────────────────────────────────────┐       │
│          │ │ Endpoints disponíveis          1.208 modelos em 15 endpoints │       │
│          │ │ APIs principais                                              │       │
│          │ │ [Chat Completions] [Respostas] [Mensagens] [Completions]    │       │
│          │ │ Mídia e multimodal                                           │       │
│          │ │ [Embeddings] [Imagens] [Edições] [Áudio] [Vídeo]            │       │
│          │ │ Search & discovery / Utilidades e gerenciamento              │       │
│          │ │ [Web Search] [Rerank] [Moderações] [Batches] [Files] [Models]│       │
│          │ └──────────────────────────────────────────────────────────────┘       │
│          │ ┌──────────────────────────────────────────────────────────────┐       │
│          │ │ Aliases para CLIs                                            │       │
│          │ │ Base URL        https://gateway.../v1            [Copiar]    │       │
│          │ │ Modelos         https://gateway.../v1/models     [Copiar]    │       │
│          │ │ Chat            https://gateway.../v1/chat/...   [Copiar]    │       │
│          │ └──────────────────────────────────────────────────────────────┘       │
└──────────┴──────────────────────────────────────────────────────────────────────┘
```

## Componentes

1. **Endpoints ativos**: URL pública e local com botão “Copiar”; caminho canônico `/v1`.
2. **Servidor local**: endereços de bind e status (`Running`/`Stopped`); copiar endereço.
3. **Segurança e prompt**: toggle “Require API key” (atalho para o API Manager) e “System prompt padrão” (injeção opcional, auditada).
4. **Endpoints disponíveis**: catálogo agrupado por categoria (APIs principais, mídia/multimodal, search, utilidades), com contagem de modelos, path e copiar. Clicar abre a política do endpoint (aba Políticas).
5. **Aliases para CLIs**: base URL, `/v1/models` e `/v1/chat/completions` prontos para colar em ferramentas.
6. **Políticas** (aba secundária): status, auth, rate limit, modelos, cache, DLP, proxy, body/timeout, CORS e versão/deprecação.
7. **Sem túneis**: nenhuma seção, botão ou status de Cloudflare/Tailscale/ngrok.

## Dados e APIs

- `GET /admin/v1/endpoint/public-urls`
- `GET /admin/v1/endpoint/server`
- `GET /admin/v1/endpoints?status=active`
- `GET /admin/v1/endpoints/{id}/policy`
- `GET/PATCH /admin/v1/endpoint/system-prompt`
- `GET /admin/v1/endpoint/aliases`
- `POST /admin/v1/endpoint/test`

## Estados

- **Loading:** skeleton nas URLs e tiles.
- **Vazio:** “Nenhum endpoint ativo” + link para Políticas.
- **Erro:** banner com retry; servidor `Stopped` mostra motivo.
- **Perigo:** desativar endpoint ou alterar system prompt exige confirmação e auditoria.

## Lazy loading

- URLs e servidor no primeiro paint; catálogo de endpoints e aliases carregados sob demanda.
- Contagem de modelos vem de cache agregado; tiles usam skeleton.
- Botão “Copiar” usa clipboard API com fallback e feedback “Copiado”.

## Acessibilidade e segurança

- Botões de copiar têm `aria-label` com o que será copiado.
- Status com ícone + texto; foco visível; teclado em todos os tiles.
- System prompt exige permissão de admin e é auditado.
- Nenhum segredo é exibido nesta página; chaves ficam no API Manager.
