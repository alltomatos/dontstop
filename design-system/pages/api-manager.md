# Página — API Manager

> Override do [MASTER](../MASTER.md). Foco: chaves, consumidores, escopos, quotas, rotação e uso por endpoint.

## Objetivo

Gerenciar quem consome a API, com qual escopo, limite e orçamento — com segredo exibido uma única vez, rotação, expiração e auditoria.

## Wireframe (desktop ≥ 1024px)

```
┌──────────┬──────────────────────────────────────────────────────────────────────┐
│ Sidebar  │ Título: API Manager                     [+ Nova chave] [Exportar]    │
│          ├──────────────────────────────────────────────────────────────────────┤
│ Dashboard│ Abas: [Chaves] [Consumidores] [Escopos] [Uso] [Auditoria]             │
│ Providers│ ┌───────────────┬───────────────┬───────────────┬───────────────┐      │
│ Endpoints│ │ Chaves ativas │ Requests 24h  │ Custo 24h     │ Expirando     │      │
│ API Mgr  │ │ 42            │ 184.2k        │ US$ 312       │ 3             │      │
│ Cache    │ └───────────────┴───────────────┴───────────────┴───────────────┘      │
│ DLP      │ ┌──────────────────────────────────────────────────────────────┐       │
│ Custo    │ │ Nome          Prefixo   Escopos   Quota   Último uso   Status    │       │
│ Observ.  │ │ prod-web      sk-9f2…   chat      50M     há 2 min     ● Ativa  │       │
│          │ │ etl-batch     sk-1a4…   embed     10M     há 1 h       ◐ 96%    │       │
│          │ │ sandbox       sk-7c1…   chat      1M      há 3 dias    ● Ativa  │       │
│          │ └──────────────────────────────────────────────────────────────┘       │
│          │ ┌──────────────── Drawer: prod-web ─────────────────────────────┐      │
│          │ │ Segredo: sk-9f2…••••  (exibido só na criação)                 │      │
│          │ │ Centro de custo: CC-102                                       │      │
│          │ │ Endpoints: [chat ✓] [models ✓] [embeddings ✗]                 │      │
│          │ │ Modelos: [auto] [gpt-4.1] [claude-sonnet]                     │      │
│          │ │ Quota: 50M tokens/mês · 100k tpm · US$ 500/mês                 │      │
│          │ │ Rate: 20 rps · throttle 0 · max sessions 10 · expira 2027-03-01│      │
│          │ │ Schedule: seg-sex 08–18 America/Sao_Paulo                     │      │
│          │ │ IP allowlist: 10.0.0.0/8   Status: ● Ativa                    │      │
│          │ │ [Rotacionar] [Revogar] [Ver uso] [Auditoria]                  │      │
│          │ └──────────────────────────────────────────────────────────────┘      │
└──────────┴──────────────────────────────────────────────────────────────────────┘
```

## Componentes

> **A página mostra apenas KPIs, abas e a tabela de chaves.** Detalhe da chave e uso/auditoria abrem em **drawer**; não há cards de detalhe inline.
> **Sem dono/tenant na UI do P0** (single-tenant). `user_id` permanece opcional no modelo para o P2.

1. **KPIs**: chaves ativas, requests 24h, custo, chaves expirando.
2. **Tabela de chaves**: nome, prefixo, escopos, quota, último uso, status, expiração.
3. **Drawer da chave**: segredo mascarado, centro de custo, endpoints/modelos permitidos, quota, limites RPM/TPM/USD, throttle, max sessions, access schedule, IP allowlist, expiração e ações.
4. **Nova chave**: formulário com escopos (endpoint/modelo), quota, limites RPM/TPM/USD, expiração, IPs e aprovação (P1).
5. **Consumidores**: times/usuários, chaves vinculadas e uso agregado.
6. **Escopos**: catálogo de permissões (`chat:write`, `models:read`, `embeddings:write`).
7. **Grupos (P1)**: allow/deny por modelo/provider, com deny precedence; chave sem grupo é irrestrita.
8. **Uso/auditoria**: requests, tokens, custo e eventos de criação/rotação/revogação — dentro do drawer ou das abas de uso/auditoria.

## Fluxo “Nova chave”

O botão **+ Nova chave** abre um **modal** (não navega para outra página) em duas etapas.

### Etapa 1 — formulário

| Campo | Regra |
|---|---|
| Nome | Obrigatório, 1–120 chars |
| Centro de custo | Texto (ex.: `CC-102`) |
| Expiração | Data **ou** opção **Sem expiração** (desabilita a data e envia `expires_at = null`, sujeito à política) |
| Quota de tokens/mês | Texto/numérico (ex.: `50M`) |
| Max sessions | Inteiro ≥ 0 |
| Endpoints permitidos | Checkboxes por categoria (Chat, Models, Embeddings, Imagens, Áudio, Web Search) |
| Modelos permitidos | **Seletor com busca + lazy loading** e múltipla seleção (ver abaixo) |
| RPM / TPM / USD mês | Numéricos ≥ 0 |
| IP allowlist | CIDRs separados por vírgula |
| Escopos de gestão | `self:usage`, `self:account-quota`, `manage` (desmarcado por padrão) |

Ações: **Cancelar** (fecha sem criar) e **Criar chave** (submit). Enter no formulário cria; nome vazio foca e mostra validação nativa.

**Sem expiração:** ao marcar, o campo de data é desabilitado e limpo; a etapa 2 mostra “Sem expiração”. A política do tenant pode proibir chaves sem expiração (nesse caso o checkbox fica bloqueado).

#### Seletor de modelos (busca + lazy loading)

A fonte do seletor são **apenas provedores conectados e modelos ativos** — não um catálogo estático. Provedores desconectados/desabilitados e modelos inativos ficam fora da lista; provedores em cooldown permanecem (estão conectados) com indicação de status.

- **Busca** por nome/id (e provider), com **debounce de 250 ms** e feedback “Carregando…”.
- **Lazy loading** em lotes de 20; botão **Carregar mais** até esgotar os resultados.
- **Origem visível:** “N modelos ativos de M provedores conectados”, agrupável por provider.
- **Múltipla seleção** com `Set`; modelos marcados aparecem como **chips removíveis** e continuam visíveis mesmo quando o filtro atual não os inclui.
- **Selecionar todos os resultados** age sobre todo o resultado filtrado (não só o lote renderizado).
- **Limpar seleção** remove todos os chips.
- **Estados:** loading (skeletons), vazio (“Nenhum modelo encontrado”), erro com “Tentar novamente”. No mockup, buscar `erro` simula a falha.
- **Acessibilidade:** `role="listbox"`, `aria-multiselectable`, opções com `aria-selected`, status em `aria-live`, chips com `aria-label` de remoção; teclado via Tab/Enter (focus trap do modal).
- **Etapa 2:** o resumo mostra os modelos selecionados (com “+N” quando muitos) e a contagem também aparece nos limites.

### Etapa 2 — sucesso e segredo único

- Ícone de sucesso + “Chave criada”.
- Segredo completo em bloco mono, com botão **Copiar chave** (clipboard + feedback “Copiado”).
- Resumo: endpoints, **modelos selecionados**, quota/limites e expiração.
- Aviso explícito: “Copie agora. Por segurança, ela não será exibida novamente.”
- **Concluir** fecha o modal, insere a linha no topo da tabela e incrementa o KPI de chaves ativas.
- Fechar por X/Esc/backdrop após a criação descarta a exibição; a chave continua válida, mas não pode ser revelada de novo (política padrão).

### Comportamento de modal

- `role="dialog"`, `aria-modal="true"`, título associado, foco inicial no nome.
- Focus trap com Tab/Shift+Tab; Esc fecha; clique no backdrop fecha.
- Ao fechar, o foco volta ao botão **+ Nova chave**.
- Estados: criando (botão loading/disabled), erro (mensagem no topo), sucesso (etapa 2).

## Dados e APIs

- `GET/POST/PATCH /admin/v1/api-keys`
- `POST /admin/v1/api-keys/{id}/rotate`
- `POST /admin/v1/api-keys/{id}/revoke`
- `GET /admin/v1/api-keys/{id}/usage`
- `GET /admin/v1/consumers`
- `GET /admin/v1/scopes`
- `GET /admin/v1/audit/events?resource=api_key`

## Estados

- **Loading:** skeleton na tabela; drawer com spinner.
- **Vazio:** “Nenhuma chave criada” + “Criar chave”.
- **Expirando:** chip `warning` com dias restantes e ação “Renovar”.
- **Revogada:** chip `destructive`; uso bloqueado e auditoria visível.
- **Pendente de aprovação (P1):** chip `info` + aprovador.

## Lazy loading

- Uso e auditoria carregados ao abrir as abas.
- Lista de escopos/modelos sob demanda (não no primeiro paint).
- Exportação assíncrona de uso.

## Segurança e a11y

- Segredo completo só na criação; depois só prefixo/últimos 4.
- Escopo mínimo por padrão; ação de revogar com confirmação.
- Nunca logar chave; auditoria com ator, IP e motivo.
- Tabela acessível; status com ícone + texto; foco no drawer.
