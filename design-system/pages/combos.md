# Página — Combos

**Rota do console:** `/combos` (item **Combos** no grupo Roteamento, ícone de camadas).
**Detalhe/edição:** `/combos/{slug}`; criação manual em `/combos/new`; criação IA em
`/combos/new?mode=ai` ou drawer sobre a lista.
**Mockup navegável:** [`../mockups/combos.html`](../mockups/combos.html).
**Guia:** `combos` nos seis locales.

> Override do [MASTER](../MASTER.md). Tema claro corporate navy/slate, densidade de dashboard,
> WCAG 2.2 AA. Nenhum dado real no mockup; os controles são estáticos e servem para validar fluxo,
> hierarquia e estados.

## 1. Objetivo

Permitir que um operador crie, entenda, ative e opere um modelo virtual (`combo/<slug>`) composto
por modelos/provedores reais, com prioridade, peso, failover e limites — manualmente ou com um
assistente de IA que consulta o catálogo real e produz um rascunho revisável.

A tela deve responder em 5 segundos:

1. Quais combos existem e quais estão ativos?
2. Qual estratégia e quantos membros cada um tem?
3. Qual a saúde/uso e qual o último erro?
4. Como criar um combo manualmente ou com IA?
5. O que acontece se um membro falhar?

## 2. Navegação e layout

- Sidebar: **Combos** no grupo **Roteamento**, entre **Providers** e **Endpoints**.
- Topbar: busca global, idioma, ajuda contextual (`combos`), tenant e ambiente.
- Lista: toolbar + tabela/cards; detalhe em drawer lateral ou rota dedicada.
- Criação manual: página em 3 passos; criação IA: drawer de chat + card de rascunho.
- Breadcrumb: `Combos / <nome>`; deep link por slug.
- Ajuda: `GET /guide/{locale}/combos`, abre em nova aba com `noopener noreferrer`.

## 3. Wireframe — lista desktop (1440 × 1000)

```text
┌ Sidebar 240 ─┬ Topbar: busca | idioma | Ajuda | tenant | avatar ─────────────┐
│ Dashboard    │ Combos                                      [Criar com IA] [+ Novo combo] │
│ Providers    │ Modelos virtuais com prioridade, failover e limites.                     │
│ Combos  ●    │ [Ativos 6] [Rascunhos 2] [Taxa de sucesso 99,4%] [Uso 24h 18,2k]        │
│ Endpoints    │ [Buscar combos...] [Todos] [Ativos] [Inativos] [Estratégia ▾] [Provider ▾]│
│ API Manager  │ ┌ Tabela ──────────────────────────────────────────────────────────────┐ │
│ MCP          │ │ Nome / slug       Status  Estratégia  Membros  Saúde  Uso 24h  Ações │ │
│ Cache        │ │ Fast coding       ● Ativo Priority RR 4        OK     12,4k   ⋯     │ │
│ DLP          │ │ combo/fast-coding       2 providers  99,6%                           │ │
│ Observ.      │ │ Long context      ● Ativo Priority    3        Warn   4,1k    ⋯     │ │
│ Custo        │ │ combo/long-context      1 provider   1 cooldown                      │ │
│ Config.      │ │ Support vision    ○ Inativo Priority  2        —      0       ⋯     │ │
│              │ └─────────────────────────────────────────────────────────────────────┘ │
│              │ [Empty/error/loading state area]                                        │
└──────────────┴─────────────────────────────────────────────────────────────────────────┘
```

## 4. Lista e filtros

### 4.1 KPIs

| KPI | Definição |
|---|---|
| Ativos | `status=active` |
| Rascunhos | `status=draft` |
| Taxa de sucesso 24h | sucesso/tentativas dos combos ativos |
| Uso 24h | requisições atribuídas a combos |
| Custo 24h | custo dos membros selecionados |
| Exhausted 24h | respostas `combo_exhausted` |

KPIs são clicáveis e aplicam o filtro correspondente. Números usam `Intl.NumberFormat`.

### 4.2 Tabela

| Coluna | Conteúdo |
|---|---|
| Nome / slug | nome forte; slug em mono; `source=ai` com chip **IA** |
| Status | `Ativo`, `Inativo`, `Rascunho` com ícone + texto |
| Estratégia | `Priority`, `Round-robin`, `Priority + round-robin` |
| Membros | contagem + mini chips de providers; `+N` se exceder 3 |
| Saúde | `OK`, `Atenção`, `Exhausted` com motivo resumido |
| Uso 24h | requisições e custo; `—` se nunca usado |
| Atualização | data relativa + tooltip absoluto |
| Ações | abrir, editar, ativar/desativar, duplicar (P1), excluir |

- Ordenação por nome, status, uso e atualização (`aria-sort`).
- Busca por nome/slug/provider/modelo.
- Filtros persistidos na URL (`?status=&strategy=&provider=&q=`).
- Acima de 50 linhas, virtualização; paginação server-side.
- Ações destrutivas em overflow menu e confirmação.

### 4.3 Estados

| Estado | Comportamento |
|---|---|
| Carregando | skeleton de 6 linhas; KPIs em skeleton; sem salto de layout |
| Vazio | ícone de camadas, “Nenhum combo ainda”, explicação e CTAs **Criar combo** / **Criar com IA** |
| Erro | banner “Não foi possível carregar combos”, causa, **Tentar novamente** e link de ajuda |
| Sem permissão | 403 sem dados; mensagem de papel/tenant |
| Sucesso | toast `aria-live="polite"`, linha atualizada, foco preservado |

## 5. Criação manual

### 5.1 Passos

```text
1 Identidade → 2 Estratégia → 3 Membros → 4 Revisão
[Cancelar]                                      [Voltar] [Salvar rascunho] [Salvar e ativar]
```

1. **Identidade:** nome, slug (pré-preenchido a partir do nome, editável até salvar), descrição e
   `source=manual`.
2. **Estratégia:** três cards com rádio, explicação e exemplo. `Priority + round-robin` é o padrão
   recomendado; failover ligado por padrão; limites RPM/TPM e máximo de tentativas.
3. **Membros:** busca de modelo com debounce de 250 ms, filtros de capability (stream, tools,
   vision, reasoning), contexto mínimo e provider. Cada linha tem provider/modelo, tier
   (0 = primeiro), peso, remover e mover para cima/baixo.
4. **Revisão:** resumo, validação server-side, warnings e ações.

### 5.2 Wireframe — editor de membros

```text
┌ Membros do combo ─────────────────────────────────────────────────────────────┐
│ [Buscar modelo...] [Tools] [Vision] [Contexto ≥ 128k] [Provider ▾]            │
│ ┌ Tier 0 ───────────────────────────────────────────────────────────────────┐ │
│ │ ≡  GPT-5 mini        OpenAI        tools · 128k · US$ 0.25/1.00  [peso 60] │ │
│ │    [Mover ↑] [Mover ↓] [Desativar] [Remover]                              │ │
│ │ ≡  Claude Sonnet     Anthropic     tools · 200k · US$ 3.00/15.00 [peso 40] │ │
│ └───────────────────────────────────────────────────────────────────────────┘ │
│ ┌ Tier 1 (fallback) ────────────────────────────────────────────────────────┐ │
│ │ ≡  Gemini Flash      Google        tools · 1M · US$ 0.10/0.40   [peso 100]│ │
│ └───────────────────────────────────────────────────────────────────────────┘ │
│ [+ Adicionar membro]                           4 membros · 2 providers · 2 tiers│
└───────────────────────────────────────────────────────────────────────────────┘
```

Regras visuais:

- Tier 0 no topo, agrupado e com legenda “menor número = primeira tentativa”.
- Peso só aparece/edita quando há mais de um membro no mesmo tier.
- Membro inelegível fica visível com chip `warning` e motivo; não é removido automaticamente.
- Duplicata provider/modelo é bloqueada com erro inline.
- Reordenação por botões acessíveis; drag é enhancement, nunca o único caminho.
- O rodapé mostra contagem de membros/providers/tiers e um resumo de cobertura.

### 5.3 Validação e erros

- Erros de campo aparecem abaixo do campo; foco no primeiro erro após submit.
- Erro de conflito de revision: banner “Este combo foi alterado por outra pessoa” +
  **Recarregar**; não descartar o formulário.
- Slug reservado/duplicado: erro inline e sugestão (`fast-coding-2`).
- Sem membro elegível: bloqueia **Salvar e ativar**, mas permite **Salvar rascunho**.
- Falha de rede: banner + retry; dados do formulário preservados.
- Sucesso: toast e redirect para o detalhe com foco no título.

## 6. Criação assistida por IA

### 6.1 Layout

```text
┌ Drawer de IA 520px ───────────────────────┬ Rascunho ao vivo ──────────────────────┐
│ Criar com IA                    [Fechar]  │ Fast coding                  [Rascunho] │
│ Assistente: [GPT-5 mini ▾]                │ Priority + round-robin · fallback ON   │
│ ┌ Conversa ──────────────────────────────┐ │ Tier 0 · 3 membros · 2 providers       │
│ │ Você: código barato, baixa latência,   │ │ ┌ GPT-5 mini · OpenAI · 60 ─────────┐ │
│ │       com fallback se der 429.         │ │ │ tools · 128k · US$ 0.25/1.00       │ │
│ │ IA: Vou priorizar...                  │ │ └───────────────────────────────────┘ │
│ │ [✓ Lendo catálogo] [✓ Comparando]     │ │ [Validar] [Aplicar ao formulário]    │
│ │ [✓ Validando]                          │ │ [Descartar]                          │
│ └────────────────────────────────────────┘ │ Warnings: 1 membro em cooldown        │
│ [Descreva o objetivo do combo...] [Enviar]│ [Salvar rascunho] [Salvar e ativar]   │
└───────────────────────────────────────────┴───────────────────────────────────────┘
```

### 6.2 Estados do chat

| Estado | UI |
|---|---|
| Inicial | sugestões de objetivo em chips (“Código”, “Resumos longos”, “Visão”, “Barato”) |
| Pensando | chip “Analisando catálogo…” e botão Enviar desabilitado |
| Ferramenta | chips compactos com nome traduzido, spinner e check; nunca JSON bruto |
| Rascunho | card lateral com membros, rationale e warnings; edição manual habilitada |
| Erro | mensagem inline com causa, retry e ação sugerida; rascunho anterior preservado |
| Limite | aviso “Limite de análise atingido” com **Continuar depois** / **Salvar rascunho** |
| Concluído | resumo, **Aplicar ao formulário** e **Salvar e ativar** |

### 6.3 Regras de UX da IA

- A IA não salva nem ativa; o botão final é sempre do usuário.
- Perguntas de clarificação têm opções rápidas e campo livre; no máximo 3 antes de propor.
- O rascunho mostra **por que** cada membro foi escolhido.
- Membros indisponíveis aparecem com warning e alternativa; não são escondidos.
- Edição manual do rascunho pausa atualizações automáticas da IA até o usuário pedir
  “Revalidar”.
- Se o catálogo mudar, o card marca `Desatualizado` e exige revalidação.
- Nenhum segredo, conta ou base URL privada aparece no chat.
- O modelo assistente não pode ser um combo; se não houver modelo com tools, o drawer abre em
  estado de erro com CTA para Providers.

## 7. Detalhe e edição

### 7.1 Cabeçalho

- Nome, slug em mono, status, revision, estratégia, provider count.
- Ações: **Editar**, **Ativar/Desativar**, **Testar resolução**, **Excluir**.
- Ativação/desativação pede confirmação quando o combo está em uso; invalida cache.

### 7.2 Abas

| Aba | Conteúdo |
|---|---|
| Membros | tabela com ordem, tier, peso, provider/modelo, capabilities, contexto, preço, saúde e motivo de inelegibilidade |
| Validação | último dry-run, erros bloqueantes, warnings e membros removidos |
| Uso | requests, tokens, custo, cache hit, taxa de sucesso, por membro |
| Auditoria | create/update/activate/delete, autor, revision, antes/depois resumido |

- Charts só na aba Uso e carregados sob demanda.
- Tabela de membros sempre mostra o tier como primeira coluna visual.
- Ações de reordenação no detalhe atualizam a revision.

### 7.3 Exclusão

- Confirmação em modal com nome, slug e impacto (“em uso por 2 API keys”).
- Exige digitar o slug ou marcar confirmação; botão destrutivo desabilitado até validar.
- Em uso: oferece **Desativar** em vez de excluir; exclusão continua disponível com aviso.
- Sucesso remove a linha, invalida cache e mostra toast.

## 8. Estados globais

| Estado | Onde | Regra |
|---|---|---|
| Loading | lista, detalhe, editor, IA | skeleton > 300 ms; spinner apenas em ação |
| Empty | lista, membros, uso, auditoria | explicação + CTA |
| Error | lista, detalhe, salvamento, IA | causa + retry + ajuda; preservar formulário |
| Success | salvar/ativar/excluir/testar | toast e atualização local |
| Conflict | edição concorrente | banner de revision + recarregar |
| Permission | rota/ação | 403 sem vazar recurso |
| Exhausted | detalhe/teste | chip `warning`, link para membros e health |

## 9. i18n

- Slug `combos`; chaves `combos.*` e `combos.ai.*` nos seis locales.
- Estratégias e status traduzidos; slug e IDs não traduzidos.
- Números, moeda e datas com `Intl`; moeda em USD com rótulo explícito.
- Link de ajuda resolvido por locale com fallback `en`.
- Textos de erro: causa + como resolver + ação; nunca stack trace.

## 10. Acessibilidade

- Contraste ≥ 4,5:1; foco visível; alvos ≥ 44 px.
- Tabela com `aria-sort`, cabeçalho sticky e ações com label.
- Reordenação por botões; drag não é requisito.
- Drawer/modal com `aria-labelledby`, foco preso, ESC e retorno de foco.
- Chat com `role="log"`/`aria-live="polite"`; status de ferramenta anunciado sem spam.
- Status nunca apenas por cor; ícone + texto.
- Testar 375, 768, 1024, 1440 px e zoom 200%; `prefers-reduced-motion`.
- `axe` sem violações críticas.

## 11. Lazy loading e performance

- Lista não carrega editor, catálogo, IA, charts nem histórico.
- Catálogo do editor em lotes de 20, debounce 250 ms, cache de 60 s.
- Detalhe carrega abas sob demanda; uso/auditoria só ao abrir.
- IA usa streaming e renderização incremental; tool logs são compactos.
- Imagens/ícones em SVG; sem biblioteca de ícones pesada no primeiro paint.
- Metas: LCP ≤ 2,5 s; INP ≤ 200 ms; CLS ≤ 0,1; bundle inicial ≤ 250 KB gzip.

## 12. Dados e APIs

| Ação | Endpoint |
|---|---|
| Listar | `GET /admin/v1/combos` |
| Criar | `POST /admin/v1/combos` |
| Detalhe | `GET /admin/v1/combos/{id}` |
| Editar | `PATCH /admin/v1/combos/{id}` |
| Membros/reordenação | `PUT /admin/v1/combos/{id}/members` |
| Ativar/desativar | `POST /admin/v1/combos/{id}/activate` / `deactivate` |
| Validar | `POST /admin/v1/combos/{id}/validate` |
| Testar | `POST /admin/v1/combos/{id}/test` |
| Excluir | `DELETE /admin/v1/combos/{id}` |
| IA — sessão | `POST /admin/v1/combos/ai/sessions` |
| IA — mensagem | `POST /admin/v1/combos/ai/sessions/{id}/messages` (SSE) |
| IA — validar/aceitar | `POST /admin/v1/combos/ai/drafts/{draftId}/validate` / `accept` |
| Modelo virtual | `GET /v1/models` → `combo/<slug>` |
| Consumo | `POST /v1/chat/completions` com `model=combo/<slug>` |

O console nunca envia `tenant_id` no body; o gateway deriva do principal autenticado.
