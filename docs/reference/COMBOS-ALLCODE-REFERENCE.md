# Combos no AllCode — estudo de referência

> **Natureza:** estudo conceitual. O AllCode **não** é dependência de runtime, relay, upstream,
> catálogo ou SDK do dontstop. Nenhum código desta análise deve ser importado ou chamado em
> produção. A implementação do dontstop é nativa e está especificada em
> [`COMBOS.md`](COMBOS.md).
>
> **Fontes:** `D:\dev\allcode`, revisão de 2026-11-12. Referências de arquivo e linha apontam para
> o estado local analisado.
>
> **Pergunta respondida:** como o AllCode agrupa modelos/provedores, escolhe um membro, faz
> failover, valida a configuração e apresenta a criação/edição — incluindo a criação assistida por
> IA.

## 1. Resumo executivo

No AllCode, um **combo** é um modelo virtual nomeado que aponta para uma lista ordenada de
`providerID/modelID`. Ele é exposto como um pseudo-provedor chamado `combo`; ao escolher
`combo/<id>`, o runtime resolve um membro concreto antes de chamar o provedor.

O modelo é **simples e config-file-first**:

- lista de membros com `priority` (número menor = primeiro);
- estratégia de failover `priority`, `round-robin` ou `priority-round-robin`;
- limites opcionais de RPM/TPM no combo;
- persistência em JSONC (`allcode.jsonc`), não em tabela relacional;
- criação assistida por IA **one-shot** (`POST /combo/generate`), que devolve um rascunho e não
  salva nada;
- não há `tenant_id`, ativação/desativação, versão, pesos, custo, contexto ou health por membro.

O AllCode tem boas ideias de UX e de modelo virtual, mas a implementação tem lacunas relevantes:
dois resolvedores divergentes, `report` de falha não ligado no caminho V1, rate limit inerte fora
de testes e ausência de failover na mesma requisição. O dontstop deve preservar o conceito e
corrigir essas lacunas.

## 2. Entidade e persistência

### 2.1 Campos

Schema V1: `packages/core/src/v1/config/combo.ts:34-45`; schema V2:
`packages/core/src/config/combo.ts:34-42`.

| Campo | Tipo | Regra observada |
|---|---|---|
| `id` | string | chave estável; no V2 é gerado no cliente com `crypto.randomUUID()` |
| `name` | string | nome de exibição; não é único |
| `models[]` | array | lista de membros |
| `models[].model` | string | `"providerID/modelID"`; o split é feito apenas na primeira `/` |
| `models[].priority` | number | menor primeiro; empate pela ordem do array |
| `failover.enabled` | boolean | liga/desliga a tentativa de pular o último membro que falhou |
| `failover.strategy` | enum | `priority`, `round-robin`, `priority-round-robin` |
| `rateLimit.requestsPerMinute` | number? | limite do combo inteiro |
| `rateLimit.tokensPerMinute` | number? | limite do combo inteiro |

O combo **não** contém credenciais, provider object, peso, timeout, temperatura, limite de custo,
contexto, capabilities ou flag por membro. Também não contém combo aninhado: o picker do editor
desabilita a seleção de outro combo (`packages/app/src/components/settings-v2/dialog-combo-v2.tsx:229`).

### 2.2 Persistência

- O arquivo é JSONC (`allcode.jsonc` / `allcode.json`), carregado e mesclado com configuração de
  projeto/instância (`packages/allcode/src/config/config.ts:140-164`, `441-468`).
- `Combo.add` chama `Config.Service.updateGlobal` e grava no arquivo global
  (`packages/allcode/src/combo/index.ts:132-136`).
- `Combo.remove` grava `{ [id]: undefined }` no mesmo caminho
  (`packages/allcode/src/combo/index.ts:138-144`).
- Não há tabela SQL, migração, FK, índice ou `tenant_id`.
- O runtime mantém um overlay em memória para reflexo imediato (`combo/index.ts:101-123`).
- Cursor de round-robin, `lastFailed` e janelas de rate limit são **somente em memória** e somem
  no restart (`combo/index.ts:105-113`).

### 2.3 Ordem, unicidade e ciclo de vida

- A chave do objeto de configuração é a única unicidade; `name` pode repetir.
- Não há limite de membros, validação de duplicata de modelo ou validação de `priority` positiva.
- `priority` é a ordem primária; empate usa a ordem do array (sort estável).
- Não existe `enabled` no combo, `createdAt`, `updatedAt` ou versão.
- O editor impede remover a última linha, mas o schema aceita zero membros; nesse caso a resolução
  falha com `ComboExhaustedError`.

## 3. Resolução e estratégias

Há **dois resolvedores**:

| Resolvedor | Caminho | Estratégias | Rate limit |
|---|---|---|---|
| `Combo.Service.resolve` | chat V1, AgentUI, memória, rotinas | as três | sim, se `report` for chamado |
| `SessionRunnerModel.resolveComboModel` | session runner V2 | somente priority | ignorado |

O seletor compartilhado é `packages/core/src/combo/select.ts:30-44`: parseia o membro, aplica
`isSelectable`, ignora o último membro que falhou quando há alternativa e devolve o primeiro
elegível.

### 3.1 `priority`

Ordena por `priority` crescente e escolhe o primeiro elegível
(`packages/allcode/src/combo/index.ts:172-174`). Empate mantém a ordem do array.

### 3.2 `round-robin`

Ordena por prioridade e rotaciona a lista inteira por um cursor por combo
(`combo/index.ts:199-200`). O cursor só avança quando `report` é chamado
(`combo/index.ts:226-228`).

### 3.3 `priority-round-robin`

Agrupa membros com a mesma `priority` em tiers; cada tier tem um cursor próprio; os tiers são
concatenados do menor para o maior (`combo/index.ts:176-196`). É a opção recomendada na UI
(`dialog-combo-v2.tsx:265`).

### 3.4 Failover real

O comportamento documentado é “se um modelo falhar, tentar o próximo”
(`packages/core/src/v1/config/combo.ts:14-16`), mas o código faz apenas:

1. o chamador registra a falha via `report`;
2. o próximo `resolve` ignora **um** membro — o último que falhou;
3. se ele for o único elegível, volta a usá-lo.

Não há loop de retry na mesma requisição. No caminho V1, `Combo.report` **não é chamado em
produção**; portanto `round-robin`, `priority-round-robin` e RPM/TPM ficam inertes fora dos testes.
No V2 há report de falha, mas ele não recebe o membro concreto, não limpa `lastFailed` no sucesso e
não rotaciona contas do provedor concreto (`packages/core/src/session/runner/model.ts:197-223`,
`263-282`).

### 3.5 Disponibilidade e limites

- `isSelectable` só verifica presença do provider/modelo no estado de providers; não verifica
  saúde, cooldown, quota, custo ou contexto.
- O endpoint de debug `GET /combo/:id/resolve` não aplica `isSelectable`
  (`packages/allcode/src/server/routes/instance/httpapi/handlers/combo.ts:24-26`).
- O rate limit é uma janela deslizante de 60 s no próprio processo
  (`combo/index.ts:146-170`); os contadores só são incrementados por `report` e não usam tokens
  reais do provedor.
- Não há circuit breaker, cooldown por membro, quota por membro ou seleção por custo/contexto.

## 4. Exposição como modelo e API

### 4.1 Modelo virtual

O combo é projetado como provider `combo`:

- `providerID = "combo"`, `modelID = id`;
- o picker grava `combo/<id>`;
- também são aceitos `combo:<id>` (legado, AgentUI/rotinas/memória) e o par
  `providerID="combo"` + `modelID=id`;
- o provider virtual é considerado conectado quando há combos;
- o modelo virtual anuncia custo zero e contexto fixo de 1.048.576 no app, o que é inconsistente
  com o catálogo real.

Ver `packages/allcode/src/provider/view.ts:53-94`,
`packages/core/src/config/plugin/provider.ts:110-135` e
`packages/app/src/context/models.tsx:44-92`.

### 4.2 API

`packages/allcode/src/server/routes/instance/httpapi/groups/combo.ts`:

| Método | Rota | Uso |
|---|---|---|
| GET | `/combo` | listar |
| POST | `/combo` | criar **ou substituir** pelo `id` |
| DELETE | `/combo/:id` | remover (idempotente) |
| GET | `/combo/:id/resolve` | debug do membro escolhido |
| POST | `/combo/generate` | rascunho assistido por IA |

Não há `GET /combo/:id` dedicado, PATCH de edição, reordenação, ativação/desativação, histórico ou
endpoint de `report`.

### 4.3 Validação

O schema exige `id`, `name`, `models`, `failover.enabled` e `failover.strategy`, e restringe a
estratégia aos três literais. Não valida lista não vazia, prioridade inteira/positiva, limites
positivos, duplicatas, existência do modelo, credencial conectada, nome único ou máximo de membros.
A UI só bloqueia nome vazio e linha sem modelo (`dialog-combo-v2.tsx:112-116`).

## 5. Criação assistida por IA

Existe, mas é **one-shot** e não conversacional:

- endpoint `POST /combo/generate` recebe `{ description, availableModels? }`;
- monta uma lista de candidatos a partir dos modelos conectados, excluindo `opencode/` e
  `agentrouter/`, e adiciona candidatos hardcoded (`combo/index.ts:271-306`);
- escolhe um modelo de linguagem com tools e chama `streamText` com `toolChoice: "required"`;
- espera uma chamada da tool `save_draft`, cujo schema pede nome, modelos, failover e estratégia;
- devolve o rascunho; **não salva o combo**;
- não valida os IDs devolvidos contra o catálogo conectado;
- não tem conversa, histórico, revisão, justificativa ou estado de erro rico;
- não testa o caminho de sucesso (só o caso “sem provider”) em
  `packages/allcode/test/combo/combo.test.ts:135-143`.

A UI abre um painel “AI Combo Specialist”, mostra um campo de descrição, chama o endpoint,
preenche o formulário e pede revisão antes de salvar
(`packages/app/src/components/settings-v2/dialog-combo-v2.tsx:150-180`).

## 6. UX observada

- Lista em Configurações: nome, quantidade de modelos, editar e excluir; exclusão sem confirmação;
  erro de carregamento vira lista vazia; loading e empty compartilham o mesmo texto.
- Editor: nome, linhas de modelo com seletor, prioridade, adicionar/remover linha, filtros
  vision/PDF apenas no picker, switch de failover, estratégia e RPM/TPM.
- Sem drawer de detalhe, sem ativação, sem histórico; o botão salvar permanece habilitado mesmo
  com formulário inválido.
- A IA é um painel dentro do editor, não um chat com histórico.

## 7. Testes que especificam o comportamento

- `packages/core/test/combo/select.test.ts`: primeiro elegível, filtro de indisponível, skip do
  último que falhou, uso do falho como último recurso, parsing pelo primeiro `/`.
- `packages/allcode/test/combo/combo.test.ts`: CRUD em memória, rate limit após `report`, escolha
  de membro conectado, skip após falha e falha graciosa sem provider.
- `packages/allcode/test/server/httpapi-exercise/index.ts:1915-1960`: contrato de list/add/remove/
  resolve.
- Não há testes para `priority-round-robin`, `round-robin`, tokens, sucesso do `generate`,
  resolvedor V2, validação de IDs ou UI.

## 8. Lições para o dontstop

### Adotar

- Combo como **modelo virtual** com ID estável e nome de exibição separado.
- Lista ordenada por prioridade, com membros explícitos `provider/model`.
- Estratégias `priority`, `round-robin` e `priority-round-robin`.
- Criação assistida por IA que produz **rascunho revisável**, nunca gravação automática.
- Editor com seleção de modelos, prioridade e estratégia.

### Corrigir

- Persistir em PostgreSQL com `tenant_id`, FK, versão, timestamps e auditoria.
- Um único resolvedor canônico, compartilhado por chat, modelos, admin e testes.
- Failover **na mesma requisição**, antes do primeiro chunk, com limite de tentativas.
- Atomicidade de cursores/rate limit em Redis para múltiplas réplicas.
- Validar membros contra catálogo ativo, provider conectado, conta ativa e capabilities.
- `enabled/status`, ativação, edição, reordenação, confirmação de exclusão e detalhe.
- IA conversacional com ferramentas de catálogo, validação server-side e estados de erro.
- Não anunciar combo com custo/contexto falsos; expor metadados derivados ou `unknown`.

## 9. O que o dontstop não fará

- Não importará `@allcode/*`, config JSONC do AllCode ou seu SDK.
- Não chamará o gateway do AllCode para resolver combos.
- Não usará `combo:` como formato canônico novo; o formato canônico será definido em
  [`COMBOS.md`](COMBOS.md).
- Não copiará as lacunas de `report`, cursores em memória ou rate limit inerte.
- Não permitirá combo de combo em P0.

## 10. Referências verificadas

| Assunto | Arquivo |
|---|---|
| Schema | `packages/core/src/v1/config/combo.ts`, `packages/core/src/config/combo.ts` |
| Runtime | `packages/allcode/src/combo/index.ts` |
| Seletor | `packages/core/src/combo/select.ts` |
| Resolvedor V2 | `packages/core/src/session/runner/model.ts` |
| Projeção como modelo | `packages/allcode/src/provider/view.ts`, `packages/core/src/config/plugin/provider.ts` |
| HTTP | `packages/allcode/src/server/routes/instance/httpapi/groups/combo.ts` |
| UI | `packages/app/src/components/settings-v2/combos.tsx`, `dialog-combo-v2.tsx` |
| Testes | `packages/core/test/combo/select.test.ts`, `packages/allcode/test/combo/combo.test.ts` |
| Contrato | `packages/sdk/js/src/v2/gen/types.gen.ts` |
