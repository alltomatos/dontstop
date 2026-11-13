# Página — Rede de proxy

**Rota do console:** `/proxies` (item **Rede de proxy** na sidebar, ícone de rede).

> Override do [MASTER](../MASTER.md). Mock: [`proxy-network.svg`](../mockups/proxy-network.svg) (tabela),
> [`proxy-network-drawer.svg`](../mockups/proxy-network-drawer.svg) (drawer do IP) e
> [`proxy-webshare-config.svg`](../mockups/proxy-webshare-config.svg) (modal Configurar Webshare).
> **Entidade principal: o IP.** Cada linha é um IP/porta do pool Webshare; as contas aparecem dentro da linha.

## Objetivo

Mostrar o pool de saída e responder, por IP: quem usa, de onde sai, em que modo, com que sticky, score e saúde.

- um provedor tem N contas e cada conta recebe o seu próprio IP **daquele provedor**;
- um mesmo IP pode ser reutilizado por contas de **provedores diferentes** (chips empilhados com provedores distintos), sem esconder vínculos;
- dentro do **mesmo provedor**, um IP nunca atende mais de uma conta (regra 1:1 por provedor — ADR-024);
- IPs livres aparecem na tabela com a coluna de contas vazia e ação **Atribuir**;
- contas sem IP **não** viram linha: ficam no bloco **Contas sem proxy**.

## Navegação

- Sidebar: **Rede de proxy** no grupo Provedores (`/providers/proxy`).
- Detalhe do provedor: card **Proxy** das contas (IP, modo, sticky, ações), com link para o IP.
- Ajuda: guia `provider-setup` (âncora `#proxy` no futuro).

## Layout (desktop ≥ 1024 px)

```
┌ Sidebar ┬ Topbar ───────────────────────────────────────────────────────────┐
│         │ Provedores / Rede de proxy                                         │
│         │ Rede de proxy                       [Testar todos][Sincronizar]   │
│         │ Cada IP com as contas que usam…      [Configurar Webshare]         │
│         │ [IPs no pool][Atribuídos][Livres][Quarentena][Países]              │
│         │ [busca] [Todos][Atribuído][Livre][Quarentena][Inválido] [filtros]  │
│         │ ┌ Tabela por IP ────────────────────────────────────────────────┐  │
│         │ │ IP·porta | Contas | Sticky | Score | Últ.verif | Quarentena | Ações│
│         │ └───────────────────────────────────────────────────────────────┘  │
│         │ [! 2 contas sem proxy]                     [Ver contas][Atribuir]  │
│         │ ┌ Localidades ┐ ┌ Pool Webshare ┐ ┌ Atividade recente ┐            │
└─────────┴────────────────────────────────────────────────────────────────────┘
```

Mock: 1440 × 1280, tema claro, sidebar 240 px, conteúdo 1104 px.

## KPIs (sempre contagens de IP)

| KPI | Definição |
|---|---|
| IPs no pool | total sincronizado |
| Atribuídos | IPs com ≥ 1 conta |
| Livres | IPs sem conta |
| Em quarentena | IPs com `quarantined_until > now()` |
| Países | países distintos no pool |

Contas sem proxy não entram como KPI: aparecem no bloco de aviso com contagem e CTA.

## Tabela por IP

Cada linha = `address:port` único.

| Coluna | Conteúdo |
|---|---|
| **IP · porta** | IP em mono + `:porta`; abaixo, localidade (cidade, país), status do IP (Saudável/Quarentena/Inválido) e latência |
| **Contas usando este IP** | 0..1 por provedor, N no total (só provedores distintos). Cada chip: ícone do provedor, provedor, conta/alias e modo (`inherit`/`off`/`manual`/`auto`). Se 2+ provedores, mostrar 2 + `+N` (o drawer lista todas) |
| **Sticky até** | menor expiração entre as contas do IP, com tempo restante; `—` quando livre. O drawer lista a expiração de cada associação |
| **Score** | 0–100 + barra (verde ≥ 85, âmbar 60–84, vermelho < 60) |
| **Últ. verificação** | “há X min” + hora |
| **Quarentena** | `—` ou tempo restante + motivo (`407`, `timeout`, `TLS`) |
| **Ações** | ícones com tooltip/label acessível: Atribuir (+), Desatribuir (×), Rotacionar, Fixar (pin), Colocar/Retirar quarentena (shield), menu `⋯` |

Regras:

- IP livre: célula de contas mostra “Nenhuma conta associada” + chip `Livre`; ação primária **Atribuir**.
- IP com N contas: todas visíveis (sem agrupar por provedor).
- Ordenação por IP, score, sticky e última verificação (`aria-sort`).
- Paginação server-side; virtualização acima de 50 linhas.

## Bloco “Contas sem proxy”

- Card de aviso com contagem, nomes (provedor · conta) e motivo (`preferred` em fallback direto, `required` bloqueado).
- Ações: **Ver contas** (filtra a lista de provedores) e **Atribuir proxy** (abre seletor de IPs livres).

## Drawer do IP

Abre ao clicar na linha/IP. Estrutura (mock `proxy-network-drawer.svg`):

1. **Cabeçalho:** IP:porta (mono), copiar, status, latência, cidade/país, ASN, id do endpoint e link “Ver no Webshare”.
2. **Contas usando este IP:** 0..N cartões com provedor, conta, credencial mascarada, modo, sticky até e ações Desatribuir/Fixar. Estado vazio com CTA **Atribuir conta**.
3. **Sticky e score:** expiração, tempo restante, score, p50, falhas 24 h.
4. **Histórico de rotações:** quem/quando (manual, expiração, atribuição, quarentena).
5. **Verificações de saúde:** últimas verificações com status e latência.
6. **Rodapé:** Rotacionar agora (confirma), Fixar, Colocar/Retirar quarentena.

## Integração Webshare (painel e modal)

O topo de `/proxies` mostra o painel **Webshare** com chip `Conectado` / `Erro na sincronização` /
`Não configurado`, plano, contagem de proxies e saudáveis, última sincronização e importados/total.
Ações: **Configurar Webshare**, **Importar do Webshare** e **Sincronizar agora** (com spinner e
resultado). Erros de sincronização são exibidos sanitizados, sem credenciais.

O modal pede **API key** (mascarada; vazio mantém a atual) e **Plan ID opcional**, permite
**Testar conexão** (estado real: 401/403/429/5xx) e **Salvar e sincronizar**. A chave é cifrada e nunca
é exibida novamente; o botão de importar só aparece quando a integração está configurada.

## Modal “Configurar Webshare”

Abre pelo botão **Configurar Webshare** (header) ou pelo empty state. Mock:
[`proxy-webshare-config.svg`](../mockups/proxy-webshare-config.svg).

Campos e estados:

1. **API token** — input de segredo mascarado, botão mostrar/ocultar, link “Criar token no dashboard”,
   botão **Testar conexão** (usa apenas o token digitado).
2. **Status do teste** — `Conectado · token válido`, com plano e quantidade de IPs; erros `401`
   (token inválido), `403` (plano), `429` (limite) e `5xx` exibidos inline com ação de tentar novamente.
3. **Plano detectado** — somente leitura: plano, IPs `128/500`, banda, última sync.
4. **Modo padrão das novas contas** — segmented `Herdar do provedor` / `Off` / `Manual` / `Auto`, com helper do comportamento.
5. **Provedor: proxy auto** — toggle que liga o modo `auto` como default das contas do provedor.
6. **País preferido**, **Sticky (min)** e **Máx. contas por IP por provedor** (padrão 1; reuso cross-provider permitido).
7. **Sincronizar automaticamente** + intervalo (padrão 60 min) e **re-sync automático ao falhar** (debounce 30 s).
8. **Quarentena após N falhas** (padrão 3; 10 min) e **poison por classe de falha** (unreachable 15 min, auth 1 h, ban 24 h).

Rodapé: **Cancelar** e **Salvar e sincronizar** (primário; loading + bloqueio de clique). O salvamento
persiste apenas no admin/dontstop; o primeiro sync com o Webshare acontece após salvar.

Regras de segurança: o token nunca é retornado pela API (máscara fixa), é cifrado em repouso e nunca
entra em log/trace; o teste não persiste o token; o modal não permite URL de proxy arbitrária.

Acessibilidade: foco preso no modal, `aria-labelledby`, labels explícitos, resumo de erro no topo,
`ESC`/clique fora fecha (com confirmação se houver alterações não salvas).

Chaves i18n: `proxy.webshare.title`, `...subtitle`, `...token`, `...createToken`, `...test`,
`...testing`, `...connected`, `...plan`, `...bands`, `...mode`, `...country`, `...sticky`,
`...maxAccounts`, `...fallback`, `...autoSync`, `...interval`, `...quarantine`, `...save`,
`...saving`, `...errors.*`.

## Filtros

- Status do IP: Todos, Atribuído, Livre, Quarentena, Inválido.
- Provedor, Conta e País/cidade (selects com contagem).
- Busca por IP, cidade, provedor ou conta.
- Persistência na URL (`?status=&provider=&account=&country=&q=`).

## Ações e API

| Ação | Endpoint |
|---|---|
| Visão geral e KPIs | `GET /admin/v1/proxies/overview` |
| Lista de IPs (filtros/paginação) | `GET /admin/v1/proxies` |
| Detalhe do IP (contas, histórico, saúde) | `GET /admin/v1/proxies/{id}` |
| Sincronizar pool | `POST /admin/v1/proxies/sync` |
| Testar IPs | `POST /admin/v1/proxies/test` |
| Rotacionar IP | `POST /admin/v1/proxies/{id}/rotate` |
| Fixar / quarentena | `PATCH /admin/v1/proxies/{id}` |
| Atribuir conta ao IP | `POST /admin/v1/proxies/{id}/assignments` |
| Desatribuir conta | `DELETE /admin/v1/proxies/{id}/assignments/{accountId}` |
| Modo de proxy da conta | `PATCH /admin/v1/providers/{id}/accounts/{accountId}/proxy` |
| Configurar Webshare | `GET/PUT /admin/v1/proxies/webshare` (token nunca retornado) |

Regras de dados: `proxy_assignments` garante **1 conta por IP por provedor** (`UNIQUE(tenant_id, provider_id, proxy_endpoint_id)`); o reuso entre provedores diferentes é permitido e visível; nunca entre tenants. `proxy_poison` mantém o ban por `(provedor, endpoint)` com TTL por classe de falha. Token/credenciais nunca retornam.

## Estados

- Sem Webshare configurado: empty state + CTA **Configurar Webshare**.
- Carregando: skeleton nas KPIs/tabela; cards secundários em paralelo.
- Erro de sync: banner com último sucesso e **Tentar novamente**.
- Parcial: banner de contas sem proxy; contas `manual` em falha ficam fail-closed (sem fallback direto).
- Quarentena total: aviso com recomendação de rotação/sync.
- Ban (poison): banner por provedor com classe da falha e tempo restante.

## Acessibilidade (WCAG 2.2 AA)

- Tabela semântica, caption, `th scope`, `aria-sort`; cada linha focável e acionável por teclado.
- Chips com texto (nunca só cor); contraste ≥ 4.5:1.
- Botões-ícone com `aria-label`/tooltip; foco visível 2 px; foco retorna à linha após ação.
- Confirmação acessível para rotação, desatribuição e quarentena.
- `prefers-reduced-motion` respeitado.

## i18n

Chaves em `admin/locales/*.json` (6 locales, fallback `en`):

`proxy.title`, `proxy.subtitle`, `proxy.kpi.*`, `proxy.filter.*`, `proxy.table.*`, `proxy.mode.*`,
`proxy.health.*`, `proxy.action.*`, `proxy.confirm.*`, `proxy.empty.*`, `proxy.sync.*`,
`proxy.webshare.*`, `proxy.accountsWithout.*`, `proxy.drawer.*`, `proxy.history.*`.

## Performance

- Crítico: KPIs + tabela. Cards secundários e drawer sob demanda.
- Filtros/ordenação client-side na página; servidor pagina.
- Sem chamadas ao Webshare no primeiro paint: sync assíncrono com polling curto.
- Lista > 50 linhas virtualizada.

## Responsivo

- < 1024 px: tabela vira cards por IP (IP, localidade, contas, saúde, ações).
- < 768 px: filtros em drawer, ações em menu, cards empilhados.
