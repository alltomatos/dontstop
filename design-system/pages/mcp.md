# Página — MCP Server

> Override do [MASTER](../MASTER.md). Modelo inspirado no OmniRoute (`/dashboard/mcp`) e no 9Router (`/dashboard/mcp`), com **enable/disable global**, transportes e ferramentas.
> **Sem túneis** e **desligado por padrão** no dontstop.

## Objetivo

Permitir que o operador ative/desative o servidor MCP, escolha o transporte, copie o endpoint/comando e acompanhe saúde, atividade e ferramentas expostas aos agentes.

## Wireframe (desktop ≥ 1024px)

```
┌──────────┬──────────────────────────────────────────────────────────────────────┐
│ Sidebar  │ Título: MCP                                   [Reiniciar]             │
│          ├──────────────────────────────────────────────────────────────────────┤
│ Dashboard│ ┌──────────────────────────────────────────────────────────────┐       │
│ Provedores│ │ Servidor MCP  6 ferramentas · 5 escopos · 3 transportes    │       │
│ Endpoints│ │                                    ● Online  [switch] ON     │       │
│ API Mgr  │ │ Como conectar: 1) dontstop mcp 2) configure o cliente      │       │
│ MCP      │ │ 3) invoque dontstop_chat                                  │       │
│ Cache    │ │ [stdio] [SSE] [Streamable HTTP]                            │       │
│ DLP      │ │ https://gateway.../api/mcp/stream              [Copiar]    │       │
│ Custo    │ │ [Processo Online] [PID —] [Uptime 2h14m] [Heartbeat 12s]   │       │
│ Observ.  │ └──────────────────────────────────────────────────────────────┘       │
│          │ ┌──────────────────────────────────────────────────────────────┐       │
│          │ │ Atividade (24h)                                              │       │
│          │ │ 1.284 chamadas · 99,2% sucesso · 107 ms                      │       │
│          │ │ Top ferramentas          Detalhes de runtime                │       │
│          │ │ dontstop_chat 812         streamable-http                   │       │
│          │ │ dontstop_models 221       chat:write, models:read          │       │
│          │ └──────────────────────────────────────────────────────────────┘       │
│          │ ┌──────────────────────────────────────────────────────────────┐       │
│          │ │ Ferramentas e escopos                                        │       │
│          │ │ dontstop_chat       chat:write         [switch]             │       │
│          │ │ dontstop_models     models:read        [switch]             │       │
│          │ │ dontstop_embeddings embeddings:write   [switch]             │       │
│          │ │ dontstop_web_search web:read           [switch]             │       │
│          │ └──────────────────────────────────────────────────────────────┘       │
└──────────┴──────────────────────────────────────────────────────────────────────┘
```

## Componentes

1. **Toggle global**: `role="switch"`, estado `Online/Offline` e `ON/OFF`; desligado mostra painel “MCP desativado” com botão “Ativar MCP”.
2. **Como conectar**: passos para stdio/HTTP e exemplo de ferramenta.
3. **Transporte**: cartões `stdio`, `SSE`, `Streamable HTTP`; o selecionado define o comando/URL exibido e o campo “Transporte” no runtime.
4. **Endpoint/comando**: `dontstop mcp` (stdio), `/api/mcp/sse` ou `/api/mcp/stream`; botão copiar com feedback.
5. **Status**: processo, PID, uptime, último heartbeat.
6. **Atividade 24h**: total de chamadas, taxa de sucesso, latência média, top ferramentas e detalhes de runtime.
7. **Ferramentas e escopos**: lista com scope e switch por ferramenta; desativar remove o anúncio e chamadas retornam erro de escopo.
8. **Sem túneis**: nenhuma opção de Cloudflare/Tailscale/ngrok.

## Dados e APIs

- `GET /admin/v1/mcp/status`
- `PATCH /admin/v1/settings` `{ mcpEnabled, mcpTransport }`
- `POST /admin/v1/mcp/restart`
- `GET /admin/v1/mcp/tools`
- `PATCH /admin/v1/mcp/tools/{id}` `{ enabled }`
- `GET /admin/v1/mcp/audit?range=24h`

## Estados

- **Desligado:** painel “MCP desativado” + “Ativar MCP”; transportes e atividade ocultos.
- **Ligado/Online:** status verde, endpoint copiável, atividade visível.
- **Erro:** status `Offline`, motivo e “Reiniciar”.
- **Ferramenta desativada:** switch off; scope permanece visível.
- **Loading:** skeleton nos cartões de status/atividade.

## Segurança

- **Desligado por padrão**; habilitar é ação auditada.
- Rotas MCP que iniciarem processo são **local-only** (loopback) ou exigem scope de gestão; sem túneis.
- Ferramentas exigem escopo (`chat:write`, `models:read`, `embeddings:write`, `web:read`, `usage:read`).
- Nenhum segredo em URL/log; endpoint exibido é o do gateway, não credencial.
- Auditoria de ativação, troca de transporte, chamada e desativação de ferramenta.

## Lazy loading

- Status e atividade carregados ao abrir a página; detalhes por ferramenta sob demanda.
- Logs/auditoria paginados; não bloquear o primeiro paint.
