# Mockups — Console dontstop

> Mockup navegável (HTML estático) para validar hierarquia, densidade e fluxos **antes** do desenvolvimento.
> Abra [`console.html`](console.html) no navegador. As abas laterais alternam entre Dashboard, Providers, Cache, DLP, Observabilidade e Custo/Quotas.

## Como visualizar

1. Abra `design-system/mockups/console.html` no Chrome/Edge/Firefox.
2. Use as abas da sidebar; a URL reflete a seção (`#dashboard`, `#providers`, …).
3. Teste navegação por teclado (Tab/Enter), zoom 200% e largura 375px.

## Fluxo de Provedores

A área de Provedores foi separada em três rotas:

- `#providers` — lista apenas os provedores já configurados, com busca, grade/lista e estado vazio.
- `#provider-catalog` — catálogo de provedores preconfigurados, acessível por **Adicionar provedor**; nunca aparece na listagem principal.
- `#provider-detail?provider=<id>` — detalhe do provedor, contas vinculadas, resumo e ação **Adicionar conta**.

O caminho de volta é feito por breadcrumb e pelos botões **Voltar para provedores** / **Voltar ao catálogo**. Para validar o estado vazio, use `?empty=1#providers`.

Cada provedor exibe a origem (`Native` ou `Custom`) em chips e nos filtros. 9Router, OmniRoute e AllCode s?o apenas refer?ncias conceituais de estudo, nunca origem de execu??o. O detalhe também oferece **Como conectar** (guia do usuário) e **Criar conta / entrar** (site do provedor). O catálogo inclui o AgentRouter com a mesma estrutura de detalhe/contas.

## Escopo do mockup

| Tela | O que valida |
|---|---|
| Dashboard | KPIs, saúde de providers, incidentes e hierarquia |
| Providers | Grade/lista, filtros por auth/capability, contas, teste e credencial mascarada |
| Endpoints | Endpoint final, URLs pública/local, catálogo por categoria, aliases e system prompt; **sem túneis** |
| Gerenciador de API | Chaves, escopos, rate limits, quotas, schedule, rotação e auditoria |
| MCP | Ativar/desativar servidor, transporte (stdio/SSE/Streamable HTTP), status, atividade e ferramentas |
| Configurações + onboarding | Troca de idioma imediata, persistência, fallback, ajuda e primeiro acesso |
| Cache | Hit exato/semântico, economia, camadas e eventos |
| DLP | Regras, teste com redaction e eventos sem PII |
| Observabilidade | SLOs e deep links para backend gerenciado (sem Grafana/Prometheus) |
| Custo | Orçamento, quotas, chargeback e exportação |

## Wireframes ASCII

### Dashboard

```
[Sidebar] [Busca ⌘K] [tenant] [ambiente] [avatar]
Visão geral                         [24h] [Atualizar]
[RPS] [Streams] [p95] [Erros] [Cache] [Custo]
[Tráfego e latência (lazy)]        [Custo por provider]
[Saúde dos providers — tabela]
[Incidentes]                       [Ações rápidas]
```

### Providers

```
Provedores                 [Testar todos] [Importar] [+ Adicionar]
[Buscar provedores] [Buscar por modelo] [Grade] [Lista]
[Total 42] [OAuth 12/18] [API key 22/50] [Compatível 6/6] ...
[Imagem] [Vídeo] [Áudio] [Embedding] [Visão] [Tools]
Provedores compatíveis 6/6                 [Adicionar compatível]
[Card] [Card] [Card] [Card]           (Grade)
Provedores OAuth 12/18              [Reparar env] [Testar todos]
[Card] [Card] [Card] [Card] [Card] [Card]
Provedores por chave de API 22/50                [Testar todos]
[Card] [Card] [Card] [Card] [Card] [Card]
Locais e sem autenticação 3/12
[Card] [Card] [Card]
Card: ícone, nome, auth, status, capabilities, contas, switch, Testar
Lista: mesma informação em linha: provedor | status | caps | contas | ações
```

### Endpoint final

```
Endpoint final                     [Testar] [Gerenciar políticas]
( APIs compatíveis com OpenAI ) Políticas  Aliases  Saúde
Endpoint da API
  ● Público https://gateway.../v1            [Copiar]
  ● Local   http://localhost:20128/v1        [Copiar]
  Servidor local 0.0.0.0:20128 127.0.0.1:20128 [::]:20128 ● Running
  Require API key ● Ativado [Abrir API Manager]
  System prompt   ○ Desligado [Editar]
Endpoints disponíveis (1.208 modelos em 15 endpoints)
  [Chat] [Responses] [Messages] [Completions]
  [Embeddings] [Images] [Edits] [Audio] [Video]
  [Search] [Rerank] [Moderations] [Batches] [Files] [Models]
Aliases para CLIs: base URL, /v1/models, /v1/chat/completions
SEM TÚNEIS.
```

### Gerenciador de API

```
Gerenciador de API                 [Exportar] [+ Nova chave]
[Chaves ativas] [Requests 24h] [Custo 24h] [Expirando]
[Chaves] [Consumidores] [Escopos] [Uso] [Auditoria]
[Tabela: nome, prefixo, escopos, quota, último uso, status]
[Drawer: segredo mascarado, endpoints/modelos, quota, RPM/TPM/USD,
 schedule, IP allowlist, expiração, rotacionar/revogar, auditoria]
```

> Detalhe e uso/auditoria ficam no drawer; a página não exibe cards inline.
> **+ Nova chave** abre um modal em duas etapas: formulário → chave exibida uma única vez.
> O seletor de modelos usa busca com debounce de 250 ms, lazy loading em lotes de 20, chips removíveis e “Selecionar todos os resultados”.

### MCP

```
MCP                                           [Reiniciar]
[Servidor MCP: 6 ferramentas · 5 escopos · 3 transportes]
[● Online  [switch] ON]  [stdio] [SSE] [Streamable HTTP]
[Endpoint/comando copiável]
[Processo] [PID] [Uptime] [Heartbeat]
[Atividade 24h: chamadas, sucesso, latência, top ferramentas]
[Ferramentas: nome, escopo, switch]
```

> Desligado por padrão; o painel “MCP desativado” substitui os detalhes até ativar.
> **Sem túneis** (Cloudflare/Tailscale/ngrok); stdio é local e HTTP exige auth/escopo.

### Configurações + onboarding

```
Configurações                                      [Ajuda]
[Idioma: 6 cartões com radio, nomes nativos]
[Preferência: aplicação imediata, dontstop.locale, fallback en]
[Abrir guia do usuário]
[Onboarding: mesmo grid + idioma detectado + Continuar]
```

> Troca de idioma sem reload; `html lang` e link de ajuda atualizados juntos.
> Onboarding só no primeiro acesso; a escolha persiste em `dontstop.locale`.

### Cache

```
Cache                              [Invalidar]
[Hit total] [Exact] [Semantic] [Economia]
[Hit rate por dia]                 [Camadas/TTL]
[Eventos recentes — sem payload]
```

### DLP

```
DLP e Políticas                    [Testar] [+ Regra]
[Regras] [Políticas] [Eventos] [Auditoria]
[Tabela de regras]
[Teste com prévia redigida]        [Eventos sem PII]
```

### Observabilidade

```
Observabilidade                    [Abrir backend ↗]
[SLO disp.] [p95] [Erros] [Saturação]
[Latência]                         [SLOs/burn rate]
[Alertas ativos com deep link]
```

### Custo

```
Custo e Quotas                     [Exportar] [+ Quota]
[Custo mês] [Economia] [Orçamento] [Projeção]
[Custo por dia]                    [Top modelos]
[Quotas: escopo, métrica, limite, usado, status]
```

## i18n e guia

- Seis idiomas: en, pt-PT, pt-BR, es, r, de; fallback para inglês.
- Primeiro acesso: onboarding com idioma detectado; escolha em localStorage (dontstop.locale).
- Settings troca o idioma sem reload; <html lang> e links de ajuda são atualizados juntos.
- Guia por página/modal em docs/user-guide/<locale>/<slug>.md; link **Ajuda** contextual no topbar e nos modais.
- Imagens em docs/user-guide/assets/<locale>/<slug>.png (1440×1200).

## O que o mockup NÃO é

- Não é código de produção: sem autenticação real, dados reais ou integração com APIs.
- Não define biblioteca final de gráficos; a decisão fica no P0.
- Não substitui os specs de página em [`../pages/`](../pages/).

## Mockups de proxy

- `provider-account-proxy.svg` - bloco de proxy por conta (modo, endpoint, saúde e ações).
- `proxy-network.svg` - visão IP-first do pool.
- `proxy-network-drawer.svg` - drawer do IP.
- `proxy-webshare-config.svg` - modal de configuração do Webshare (API key, plan ID opcional e teste real).
