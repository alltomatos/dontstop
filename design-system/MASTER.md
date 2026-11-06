# dontstop — Design System Master

> **Fonte de verdade visual do console corporativo.**
> Base gerada com `/ui-ux-pro-max` (dials: variance 4, motion 3, density 7) e ajustada para contexto enterprise: paleta navy/slate, densidade de dashboard e acessibilidade AA.
> **Regra:** ao construir uma página, leia `design-system/pages/<page>.md`; se existir, ele sobrescreve este Master.

## 1. Princípios

1. **Confiança corporativa:** interface sóbria, previsível e auditável; nada de decoração que compita com dados.
2. **Dados primeiro:** hierarquia por tipografia, contraste e espaço — não por cor.
3. **Densidade controlada:** dashboards densos, mas com respiro e agrupamento claros.
4. **Acessível por padrão:** WCAG 2.2 AA, teclado, foco visível, contraste 4.5:1 e `prefers-reduced-motion`.
5. **Rápido por padrão:** lazy loading agressivo, skeletons, cache de dados e bundle inicial enxuto.
6. **Seguro por padrão:** ações destrutivas exigem confirmação; segredos são mascarados; nada de PII em tela sem política.

## 2. Tokens

### 2.1 Cores (light)

| Token | Hex | Uso |
|---|---|---|
| `--color-primary` | `#1E3A8A` | Ações primárias, navegação ativa |
| `--color-on-primary` | `#FFFFFF` | Texto sobre primário |
| `--color-secondary` | `#0F766E` | Ações secundárias, sucesso analítico |
| `--color-accent` | `#0EA5E9` | Destaques, links, séries de gráfico |
| `--color-background` | `#F8FAFC` | Fundo da aplicação |
| `--color-surface` | `#FFFFFF` | Cards, tabelas, modais |
| `--color-foreground` | `#0F172A` | Texto principal |
| `--color-muted` | `#F1F5F9` | Fundos sutis, hover de linha |
| `--color-muted-foreground` | `#475569` | Texto secundário |
| `--color-border` | `#E2E8F0` | Bordas e divisores |
| `--color-destructive` | `#DC2626` | Erro, exclusão, bloqueio |
| `--color-success` | `#059669` | Saúde OK, hit de cache |
| `--color-warning` | `#D97706` | Cooldown, atenção |
| `--color-info` | `#0284C7` | Informação neutra |
| `--color-ring` | `#1E3A8A` | Foco visível |

### 2.2 Cores (dark)

| Token | Hex |
|---|---|
| `--color-primary` | `#60A5FA` |
| `--color-secondary` | `#2DD4BF` |
| `--color-accent` | `#38BDF8` |
| `--color-background` | `#0B1220` |
| `--color-surface` | `#111827` |
| `--color-foreground` | `#E5E7EB` |
| `--color-muted` | `#1F2937` |
| `--color-muted-foreground` | `#94A3B8` |
| `--color-border` | `#1F2937` |
| `--color-destructive` | `#F87171` |
| `--color-success` | `#34D399` |
| `--color-warning` | `#FBBF24` |

### 2.3 Tipografia

- **Heading/body:** `Fira Sans` (300/400/500/600/700).
- **Mono/dados:** `Fira Code` (400/500/600) para IDs, chaves, latências e tabelas numéricas.
- Base 16px; line-height 1.5 (corpo) e 1.2 (títulos).
- Escala: 12 / 14 / 16 / 18 / 20 / 24 / 30 / 36.
- Números tabulares em tabelas e KPIs (`font-variant-numeric: tabular-nums`).

### 2.4 Espaçamento e raio

- Escala: 4 / 8 / 12 / 16 / 24 / 32 / 48.
- Raio: 8px inputs/botões, 12px cards, 16px modais/drawers.
- Sombras: `sm` 0 1px 2px, `md` 0 4px 6px, `lg` 0 10px 15px, sempre discretas.
- Grid: sidebar 240px, topbar 56px, conteúdo max 1440px, 12 colunas, gutter 24px.
- Breakpoints: 375 / 768 / 1024 / 1440.

## 3. Componentes

### Botões
- Primário: fundo `--color-primary`, texto branco, altura 40px, raio 8px.
- Secundário: superfície + borda; destrutivo: `--color-destructive` com confirmação.
- Estados: hover 150ms, focus ring 2px, disabled 0.5 + `cursor-not-allowed`, loading com spinner e bloqueio de clique.

### Campos
- Label visível acima; helper text persistente; erro abaixo do campo com ícone + texto.
- Validação no blur; foco automático no primeiro erro após submit.
- Inputs de segredo: mascarados, botão “mostrar”, nunca logados.

### Tabelas de dados
- Cabeçalho sticky; linhas com hover; colunas numéricas alinhadas à direita.
- Ações em overflow menu; seleção múltipla com barra de ações em lote.
- Virtualização acima de 50 linhas; paginação server-side; estados vazio/erro/carregando.
- Ordenação com `aria-sort`; filtros persistidos na URL.

### KPIs
- Card com rótulo, valor, variação e sparkline.
- Para meta vs. atual, usar **bullet chart** (valor sempre visível em texto).
- Nunca usar cor sozinha: incluir ícone e rótulo (“Saudável”, “Em cooldown”).

### Status e feedback
- Chips: `success`, `warning`, `destructive`, `info`, sempre com ícone + texto.
- Skeleton para >300ms; spinner apenas para ações pontuais.
- Toasts 3–5s, `aria-live="polite"`, sem roubar foco.
- Empty state com explicação + ação primária.

### Navegação
- Sidebar fixa com ícone + label; item ativo com barra e contraste.
- Breadcrumbs em hierarquias de 3+ níveis.
- Command palette (`Ctrl/Cmd+K`) para busca global.
- Deep links para todas as telas; filtros e abas na URL.

## 4. Layout

- **Desktop:** sidebar + topbar + conteúdo; cards em grid de 12 colunas.
- **Tablet:** sidebar colapsável; cards em 2 colunas.
- **Mobile:** topbar com menu; cards em 1 coluna; tabelas viram lista de cartões.
- Sem scroll horizontal na página; tabelas podem rolar dentro do container.
- Conteúdo crítico acima da dobra; gráficos e editores carregados sob demanda.

## 5. Lazy loading e performance de UI

- Rotas com code splitting (`loading.tsx`/Suspense) e skeleton.
- Charts, Monaco, logs e tabelas grandes carregados sob demanda.
- Imagens com `next/image`, `loading="lazy"` e dimensões explícitas.
- Fontes com `display: swap`; preload apenas da fonte principal.
- Dados com cache (SWR/React Query) e `stale-while-revalidate`; prefetch no idle.
- Metas: LCP ≤ 2,5s, INP ≤ 200ms, CLS ≤ 0,1, bundle inicial ≤ 250KB gzip.

## 6. Acessibilidade

- Contraste ≥ 4,5:1 (texto) e ≥ 3:1 (ícones/linhas de dados).
- Foco visível em todos os interativos; ordem de tabulação lógica.
- `aria-label` em ícones sem texto; labels em formulários.
- `prefers-reduced-motion` respeitado; animações ≤ 300ms.
- Não usar emoji como ícone estrutural; usar SVG (Lucide/Phosphor).
- Testar 375px, 768px, 1024px e 1440px; zoom 200%.

## 7. Movimento

- Microinterações 150–300ms, ease-out na entrada e ease-in na saída.
- Sem animação decorativa; movimento comunica causa e efeito.
- Stagger de listas 30–50ms; nunca animar layout (usar transform/opacity).

## 8. Voz e conteúdo

- PT-BR claro e objetivo; termos técnicos preservados (provider, cache, DLP).
- Números com separador local e unidade explícita (ms, req/s, US$).
- Erros: causa + como resolver + ação de retry.
- Nunca exibir segredo completo; chave só na criação.

## 9. Anti-patterns

- Gradientes chamativos, neon ou estética “gamer”.
- Excesso de cores por status; usar ícone + texto.
- Tabelas sem estado vazio/erro/carregando.
- Modais para navegação primária.
- Ações destrutivas sem confirmação.
- Texto cinza claro sobre fundo claro.
- Gráficos sem legenda, eixo ou alternativa textual.
- Carregar tudo no primeiro paint.

## 10. Páginas e mockups

| Página | Especificação | Mockup |
|---|---|---|
| Dashboard | [pages/dashboard.md](pages/dashboard.md) | `mockups/console.html#dashboard` |
| Providers | [pages/providers.md](pages/providers.md) | `mockups/console.html#providers` |
| Endpoint final + Políticas | [pages/endpoints.md](pages/endpoints.md) | `mockups/console.html#endpoints` |
| Gerenciador de API | [pages/api-manager.md](pages/api-manager.md) | `mockups/console.html#api-manager` |
| MCP | [pages/mcp.md](pages/mcp.md) | `mockups/console.html#mcp` |
| Configurações + onboarding | [pages/settings.md](pages/settings.md) | `mockups/console.html#settings` |
| Cache | [pages/cache.md](pages/cache.md) | `mockups/console.html#cache` |
| DLP | [pages/dlp.md](pages/dlp.md) | `mockups/console.html#dlp` |
| Observabilidade | [pages/observability.md](pages/observability.md) | `mockups/console.html#observability` |
| Custo/Quotas | [pages/cost.md](pages/cost.md) | `mockups/console.html#cost` |

> **Fonte:** `/ui-ux-pro-max` + regras corporativas do dontstop. A paleta gerada (AI purple/pink) foi substituída por navy/slate para uso enterprise.
