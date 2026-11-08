# Página — Provedores

> Override do [MASTER](../MASTER.md). Modelo visual inspirado no OmniRoute: catálogo em cards, filtros por tipo/capability e ações em lote.
> O mockup usa o tema claro corporativo; os tokens suportam dark mode.

## Categorias e origem

Cada provedor declara uma origem (`Native` ou `Custom`) no modelo de dados e na UI. 9Router, OmniRoute e AllCode s?o apenas refer?ncias conceituais de estudo, nunca origem de execu??o. A origem aparece nos cards, no resumo do detalhe e nos filtros da listagem principal e do catálogo. O detalhe também expõe **Como conectar** (guia do usuário) e **Criar conta / entrar** (site do provedor).

## Revisão do fluxo (mockup atual)

A página principal lista **somente provedores configurados**. O catálogo de provedores preconfigurados fica em uma rota separada, aberta por **Adicionar provedor** (`#provider-catalog`). O detalhe (`#provider-detail?provider=<id>`) concentra contas, status, capabilities e **Adicionar conta**.

O wireframe e o drawer abaixo são a referência histórica do catálogo original; o mockup atual implementa a separação em três rotas e mantém o caminho de volta por breadcrumb.

## Objetivo

Gerenciar conexões de provedores de IA com visão rápida de saúde, contas, capabilities e status — sem perder o acesso ao detalhe de contas/modelos/políticas.

## Wireframe (desktop ≥ 1024px)

```
┌──────────┬──────────────────────────────────────────────────────────────────────┐
│ Sidebar  │ Título: Provedores     [Testar todos] [Importar] [+ Adicionar]      │
│          ├──────────────────────────────────────────────────────────────────────┤
│ Dashboard│ ┌──────────────────────────────────────────────────────────────┐       │
│ Providers│ │ [Buscar provedores...] [Buscar por modelo...]                 │       │
│ Endpoints│ │ [Tudo] [Configurado] [Compacto]                              │       │
│ API Mgr  │ │ Total 42  OAuth 12/18  API key 22/50  Compatível 6/6 ...     │       │
│ Cache    │ │ Imagem 2/8  Vídeo 1/4  Áudio 0/5  Embedding 3/9  Tools 8/20  │       │
│ DLP      │ └──────────────────────────────────────────────────────────────┘       │
│ Custo    │ Provedores compatíveis 6/6                    [Adicionar compatível] │
│ Observ.  │ [Naga EMB] [TokenRouter] [OpenAI-compat] [Anthropic-compat]          │
│          │ Provedores OAuth 12/18                  [Reparar env] [Testar todos]  │
│          │ [Antigravity] [Claude Code] [Kilo Code] [Kiro AI] [Codex] [Gemini]   │
│          │ Provedores por chave de API 22/50                     [Testar todos]  │
│          │ [OpenAI] [Anthropic] [Gemini] [Azure] [Bedrock] [Groq]              │
│          │ Locais e sem autenticação 3/12                                      │
│          │ [Ollama] [LM Studio] [OpenCode Free]                                │
└──────────┴──────────────────────────────────────────────────────────────────────┘
```

## Card de provedor

```
┌──────────────────────────────────────┐
│ [OA] OpenAI                  ●       │
│      API key · OpenAI                │
│ [Chat] [Vision] [Tools]              │
│ 3 conectadas        [switch] [Testar]│
└──────────────────────────────────────┘
```

- **Ícone** com iniciais/marca (SVG; nunca emoji).
- **Nome** + tipo de auth/prefixo.
- **Status** por ícone + cor (`ok`, `cooldown`, `off`, `error`).
- **Capabilities** em chips: Chat, Vision, Tools, Image, Audio, Embed.
- **Contas conectadas** (ex.: `3 conectadas`, `2/3`).
- **Switch** de ativação e **Testar**.
- Clique no card abre o drawer de detalhes (contas, modelos, saúde, políticas).

## Visões: Grade e Lista

O usuário alterna entre **Grade** (padrão) e **Lista** no toolbar; a escolha persiste na URL (`?view=grid|list`) e no localStorage.

### Grade

- Cards em 4 colunas (desktop), 2 (tablet) e 1 (mobile).
- Agrupamento por tipo de conexão (compatíveis, OAuth, API key, locais).
- Ideal para varredura visual de capabilities e status.

```
[Card] [Card] [Card] [Card]
[Card] [Card] [Card] [Card]
```

### Lista

- Linhas densas, mantendo os mesmos grupos.
- Colunas: provedor (ícone+nome+auth), status, capabilities, contas, p95/cache, ações (switch + Testar).
- Em telas < 900px a linha vira um card empilhado.

```
Provedor            Status   Capabilities      Contas      Ações
[OA] OpenAI         ● OK     Chat Vision Tools 3 conectadas [switch] Testar
[AN] Anthropic      ◐ Cool   Chat Tools        2/3          [switch] Testar
```

- A troca de visão não perde filtros, busca nem posição de scroll.
- Acessível por teclado; botões com `aria-pressed` e rótulos “Grade”/“Lista”.

## Filtros

| Grupo | Exemplos |
|---|---|
| Tipo de auth | OAuth, Chave de API, Compatível, Sem auth, Local, Upstream proxy, Cookie web |
| Capacidade | Chat, Visão, Tools, Imagem, Vídeo, Áudio, Embedding |
| Estado | Total, Configurado, Com erro, Em cooldown |
| Busca | Por provedor, por modelo |

- Filtros são combináveis e persistidos na URL.
- Contadores refletem o filtro atual (`12/18` = configurados/total).
- Filtro **Configurado** disponível como chip de estado; a visão (Grade/Lista) é independente do filtro.

## Ações em lote

- Testar todos os provedores visíveis.
- Reparar env de OAuth.
- Importar configuração (arquivo/JSON) — sem túneis.
- Adicionar compatível OpenAI/Anthropic.

## Drawer de detalhes

- Cabeçalho: nome, auth, status, ações (testar, desativar, remover).
- **Contas**: nome, prioridade/peso, credencial mascarada, proxy, cooldown, último erro.
- **Modelos**: busca, capabilities, contexto; carregados sob demanda.
- **Saúde**: p95, erros, cache hit, circuit breaker, últimos eventos.
- **Políticas**: modelos permitidos, rate limit, proxy, DLP (link para a política do endpoint quando aplicável).

## Dados e APIs

- `GET /admin/v1/providers?status=&auth=&capability=&q=&model=`
- `POST /admin/v1/providers`
- `POST /admin/v1/providers/import`
- `POST /admin/v1/providers/test` (lote)
- `GET /admin/v1/providers/{id}`
- `GET /admin/v1/providers/{id}/accounts`
- `GET /admin/v1/providers/{id}/models`
- `GET /admin/v1/providers/{id}/health`

## Estados

- **Loading:** skeleton nos cards e contadores.
- **Vazio:** “Nenhum provedor encontrado” + limpar filtros.
- **Erro:** card com status `error`, tooltip e ação “Testar”.
- **Cooldown:** chip `warning` com tempo restante.
- **Desativado:** card com opacidade reduzida e switch off.

## Lazy loading

- Cards do primeiro grupo renderizados no SSR; grupos abaixo da dobra carregados sob demanda.
- Modelos e saúde do drawer só ao abrir.
- Contadores agregados em cache de 30–60 s.
- Ícones de provedor em SVG otimizado; sem biblioteca pesada.

## Acessibilidade e segurança

- Cada card é focável e abre com Enter; switch tem `aria-checked`.
- Status nunca só por cor: ícone + texto no tooltip/label.
- Credencial nunca completa; teste mostra apenas status/latência.
- Ações destrutivas com confirmação; filtros e busca com labels.
