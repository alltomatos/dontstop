# DonStop IA — runtime do console

Este pacote contém o runtime compartilhado de i18n, preferências e links de ajuda usado pelo console em `design-system/mockups/console.html`, além da validação do guia versionado.

## Estrutura

```text
admin/
├─ locales/               # en, pt-PT, pt-BR, es, fr, de
├─ src/
│  ├─ i18n.js             # detecção, fallback, interpolação e Intl
│  ├─ preferences.js      # onboarding e persistência (dontstop.locale)
│  ├─ source-keys.js      # mapa de migração de textos estáticos do mockup
│  ├─ guide-links.js      # URL base/branch e fallback do guia
│  └─ console-runtime.js  # aplicação no DOM, Settings, onboarding e ajuda
├─ tests/                 # node:test (sem dependências)
└─ scripts/               # validate/build
```

## Comandos

```bash
cd admin
npm run lint
npm test
npm run build
```

## Visualizar o console

Na raiz do repositório:

```bash
python -m http.server 8765 --bind 127.0.0.1
```

Abra `http://127.0.0.1:8765/design-system/mockups/console.html`.

Parâmetros úteis:

- `?lang=pt-BR` — força um idioma sem persistir (usado nas capturas).
- `?onboarding=1` — força o onboarding.
- `?open=new-key#api-manager` — abre o modal de nova chave.
- `#settings` — abre Configurações.

## Guia

O conteúdo e as imagens ficam em `docs/user-guide/`. Para regenerar textos:

```bash
node scripts/generate-user-guide.js
```

Para recapturar as 72 imagens (requer Chrome; use `CHROME_PATH` se necessário):

```bash
node scripts/capture-user-guide.js
```
