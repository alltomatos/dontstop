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

## Contas e testes segmentados

O detalhe do provedor mantém o **Resumo** consolidado e adiciona um card **Resumo por conta**
(data-dense, uma linha por conta) com: nome, indicador Ativa/Inativa, estado de saúde
(Saudável, Cooldown, Quarentena, Indisponível, Desabilitada), modelos disponíveis, modelos
utilizáveis e chips de capacidades. Contas em cooldown/quarentena mostram `0 utilizáveis` e não
entram em `usable_accounts`.

Acima da lista de modelos há o seletor **Conta para teste**, persistido em
`localStorage:dontstop.modelTestAccount.v1` por provedor. Somente contas ativas com modelos
utilizáveis aparecem; o padrão é a última conta válida ou a primeira ativa. O botão Testar de
cada modelo envia `account_id`, exibe spinner local e o resultado nomeia a conta usada. O botão
Testar da conta usa o endpoint individual e mostra o resultado sob a própria conta, com
`role="status"`; falhas são isoladas.

## Proxy por conta (detalhe da conta)

Mock: [`provider-account-proxy.svg`](../mockups/provider-account-proxy.svg) (bloco dentro da conta) e
[`proxy-webshare-config.svg`](../mockups/proxy-webshare-config.svg) (configuração do pool).

Cada conta em `/providers/{id}` exibe:

1. **Modo de proxy** (`inherit` | `off` | `manual` | `auto`), com helper:
   - `inherit`: segue o default do provedor (`proxy_auto`);
   - `off`: ignora o proxy e conecta direto;
   - `manual`: usa o proxy fixo atribuído, **fail-closed** (sem fallback direto);
   - `auto`: o gateway gerencia IPs do Webshare (1 por conta do provedor) e rotaciona na falha.
2. **Proxy atribuído**: `endpoint:porta`, região (cidade/país) e chip de saúde
   (`Saudável`, `Em quarentena`, `Banido`, `Indisponível`). Sem atribuição mostra “Sem proxy atribuído”.
3. **Ações**: `Atribuir proxy` / `Reatribuir` (modal com o pool e seleção única) e
   `Remover atribuição`. O modal permite cadastrar um proxy manual (endereço, porta, região,
   rótulo, usuário e senha) quando o pool está vazio. No modo `auto`, a atribuição é gerenciada
   pelo gateway (botão desabilitado, com badge `Auto`).
4. **Default do provedor**: toggle **Proxy auto** no cabeçalho do provedor, aplicado às contas em `inherit`.

Regra de alocação (ADR-024): **1 conta por IP por provedor**; o mesmo IP pode ser reutilizado por
contas de **provedores diferentes**. O modal de atribuição exibe “em uso por outra conta deste
provedor” e desabilita o item (contagem por provedor). Poison é por `(provedor, endpoint)`.

Estados do bloco/modal: carregando (skeleton no modo e spinner na lista), vazio
(“Nenhum proxy cadastrado”), erro de carregamento/salvamento com `Tentar novamente`, item em uso
pelo mesmo provedor desabilitado com contagem, e sucesso reflete via `router.refresh`.

APIs: `PATCH /admin/v1/providers/{id}` (`proxy_auto`), `PATCH /admin/v1/providers/{id}/accounts/{accountId}` (`proxy_mode`),
`GET|POST /admin/v1/proxies`, `PUT|DELETE /admin/v1/providers/{id}/accounts/{accountId}/proxy`.
Credenciais do proxy nunca voltam ao cliente (somente `id`, `label`, `address`, `port`, região e saúde).

i18n: `providers.proxy*` (título, modos, helpers, saúde, ações, modal, cadastro, erros e limite) nos
seis locales. Acessibilidade: `select` com `aria-label` por conta, modal com foco preso,
`aria-labelledby`, itens de rádio navegáveis e mensagens `role="alert"`.

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
