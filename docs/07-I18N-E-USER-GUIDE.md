# 07 — Internacionalização e guia do usuário

> Implementação do console administrativo (mockup funcional em `design-system/mockups/console.html`) e do guia versionado em `docs/user-guide/`.

## 1. Idiomas suportados

| Código | Idioma | Direção |
|---|---|---|
| `en` | English | LTR |
| `pt-PT` | Português (Portugal) | LTR |
| `pt-BR` | Português (Brasil) | LTR |
| `es` | Español | LTR |
| `fr` | Français | LTR |
| `de` | Deutsch | LTR |

O fallback é sempre **inglês (`en`)**. Uma chave ausente nunca quebra a UI: o runtime devolve a tradução inglesa; se ela também não existir, devolve a própria chave.

## 2. Preferência de idioma

- **Chave:** `dontstop.locale` no `localStorage`.
- **Formato:** string BCP-47 simples, por exemplo `"pt-BR"`, `"de"` ou `"fr"`. Não é JSON.
- **Primeiro acesso:** `navigator.languages` é consultado e a melhor correspondência é pré-selecionada no onboarding. Confirmar persiste a escolha. Nas visitas seguintes o onboarding não reaparece.
- **Backend (P1):** a preferência também deve ser gravada em `user_preferences.locale`; o valor local tem precedência até o login sincronizar a conta.
- **Settings:** altera o idioma, persiste a nova escolha e atualiza `<html lang>` imediatamente, sem recarregar.
- **Reset local:** remover `dontstop.locale` no DevTools e recarregar reabre o onboarding.

## 3. Arquivos de locale

```text
admin/locales/
├─ en.json
├─ pt-PT.json
├─ pt-BR.json
├─ es.json
├─ fr.json
└─ de.json
```

- Chaves semânticas e planas com namespace por página/feature: `nav.*`, `common.*`, `dashboard.*`, `providers.*`, `endpoints.*`, `api.*`, `mcp.*`, `cache.*`, `dlp.*`, `observability.*`, `cost.*`, `newKey.*`, `settings.*`, `onboarding.*`, `models.*`.
- Interpolação usa `{name}` (ex.: `models.footer` → `{shown} of {total} models`).
- Pluralização usa `Intl.PluralRules` com chaves `.one` / `.other` (ex.: `providers.connected.one`).
- Datas, números e moeda usam `Intl.DateTimeFormat`, `Intl.NumberFormat` e `Intl.RelativeTimeFormat` no runtime.

## 4. Como adicionar ou editar traduções

1. Adicione a chave em `admin/locales/en.json`.
2. Adicione a mesma chave nos outros cinco arquivos (o teste de paridade falha se faltar).
3. Use a chave no HTML (`data-i18n="..."`, `data-i18n-aria-label`, `data-i18n-title`, `data-i18n-placeholder`) ou no JS (`window.tr('chave', { vars })`).
4. Rode `cd admin && npm test` (paridade, fallback, plural, links e guias).
5. Rode `cd admin && npm run lint` para validar JSON, cobertura e imagens.
6. Rode `cd admin && npm run build` para gerar o artefato de build.

Strings estáticas antigas do mockup são traduzidas por um mapa de origem em `admin/src/source-keys.js`. Ao criar uma tela nova, prefira `data-i18n`; o mapa existe apenas para a migração incremental do mockup.

## 5. Resolução de locale e fallback

Implementação em `admin/src/i18n.js`:

- `detectLocale(navigator.languages)`: normaliza `pt` → `pt-BR`, `pt-PT`, `en-*` → `en`, etc.
- `resolveLocale(locale, available)`: usa o locale se suportado; senão `en`; senão o primeiro disponível.
- `translate(catalog, key, vars, fallbackCatalog)`: catálogo atual → inglês → chave.
- `translatePlural(...)`: `Intl.PluralRules` → `.one`/`.other`, com fallback.

## 6. Onboarding e acessibilidade

- Modal `role="dialog"`, `aria-modal="true"`, `aria-labelledby` e `aria-describedby`.
- Foco inicial no idioma pré-selecionado; `Tab` fica preso ao modal; `Esc` não fecha sem escolha (quando forçado para captura, é apenas uma sessão de teste).
- A mudança de idioma é anunciada por uma região `aria-live="polite"` (`#locale-live`).
- Settings usa `fieldset`/`legend` e rótulos nativos; foco visível em todos os cartões.
- Layout testado com alemão e francês; o grid de idiomas quebra para uma coluna abaixo de 900 px.

## 7. Guia do usuário versionado

### Estrutura

```text
docs/user-guide/
├─ README.md
├─ manifest.json
├─ en/README.md + <slug>.md
├─ pt-PT/...
├─ pt-BR/...
├─ es/...
├─ fr/...
├─ de/...
└─ assets/<locale>/<slug>.png
```

### Inventário de slugs

| Slug | Item |
|---|---|
| `dashboard` | Página Dashboard |
| `providers` | Página Provedores |
| `endpoints` | Página Endpoints |
| `api-manager` | Página Gerenciador de API |
| `mcp` | Página MCP |
| `cache` | Página Cache |
| `dlp` | Página DLP |
| `observability` | Página Observabilidade |
| `cost` | Página Custo |
| `settings` | Página Configurações |
| `new-api-key` | Modal Nova chave de API |
| `onboarding` | Modal de primeiro acesso |

O slug é estável, minúsculo e igual em todos os idiomas. Ao criar, renomear ou remover uma página/modal, atualize `scripts/user-guide-content.js`, rode `node scripts/generate-user-guide.js` e atualize os índices (o gerador já reescreve `README.md` global, os seis índices e `manifest.json`).

### Conteúdo mínimo por guia

- Imagem com `alt` traduzido.
- O que faz, quando usar, passo a passo, campos/ações, erros/estados e guias relacionados.

### Imagens

- Viewport consistente **1440×1200**, capturada do console em execução.
- Uma imagem por idioma; o idioma é aplicado via `?lang=<code>` antes da captura.
- Script: `node scripts/capture-user-guide.js` (requer Chrome em `CHROME_PATH` ou no caminho padrão).
- `alt` traduzido em cada guia; se uma captura não puder ser refeita no idioma, registre a exceção no topo do arquivo localizado.

### Link contextual de ajuda

- Componente: botão **Ajuda** no topbar (muda com a página ativa) e link próprio em cada modal.
- Configuração central em `admin/src/guide-links.js`:
  - `repoUrl`: `https://github.com/alltomatos/dontstop`
  - `branch`: `main`
  - `basePath`: `docs/user-guide`
- URL: `<repoUrl>/blob/<branch>/<basePath>/<locale>/<slug>.md`.
- Fallback: guia do locale → guia em inglês → `docs/user-guide/README.md`.
- `target="_blank"`, `rel="noopener noreferrer"`, `aria-label`/`title` traduzidos; o href é recalculado ao trocar de idioma sem recarregar.

## 8. Testes e validação

```bash
cd admin
npm run lint    # valida JSON, paridade, guias e imagens
npm test        # i18n, fallback, plural, links e guia
npm run build   # artefato de build + validação final
```

Cobertura atual: resolução de locale, fallback, interpolação, plural, onboarding/persistência (teste de contrato do runtime), troca em Settings e resolução dos links de ajuda.

## 9. Limitações conhecidas

- O repositório ainda não contém o app Next.js do admin; o console deste documento é o mockup funcional em `design-system/mockups/console.html`, com runtime compartilhado em `admin/src/`.
- A preferência de backend (`user_preferences.locale`) fica para o P1; o `localStorage` é a fonte no P0.
- Textos técnicos (nomes de modelos, paths, scopes e URLs) permanecem como identificadores e não são traduzidos.
