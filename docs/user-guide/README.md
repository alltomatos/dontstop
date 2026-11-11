# DonStop IA — User guide

> Versioned user guide for every console page, modal and workflow. Six languages share the same slugs.
> **Fallback:** if a localized guide is missing, the help link opens the English guide; if no guide exists, it opens this index.
>
> **Console links:** the admin serves these guides locally as static HTML. After editing a guide or
> its screenshots, run `node scripts/build-guide-html.js` (or `npm run guide:build` in `admin/`) to
> regenerate `admin/public/guide/`. The help links never point to an external repository, so they work
> offline and inside the container.

## Languages

| Code | Language | Index |
|---|---|---|
| `en` | English | [English](en/README.md) |
| `pt-PT` | Português (Portugal) | [Português (Portugal)](pt-PT/README.md) |
| `pt-BR` | Português (Brasil) | [Português (Brasil)](pt-BR/README.md) |
| `es` | Español | [Español](es/README.md) |
| `fr` | Français | [Français](fr/README.md) |
| `de` | Deutsch | [Deutsch](de/README.md) |

## Guides

| Slug | Screen | English guide | Image |
|---|---|---|---|
| `dashboard` | Dashboard | [English](en/dashboard.md) | [PNG](assets/en/dashboard.png) |
| `providers` | Providers | [English](en/providers.md) | [PNG](assets/en/providers.png) |
| `endpoints` | Endpoints | [English](en/endpoints.md) | [PNG](assets/en/endpoints.png) |
| `api-manager` | API Manager | [English](en/api-manager.md) | [PNG](assets/en/api-manager.png) |
| `mcp` | MCP | [English](en/mcp.md) | [PNG](assets/en/mcp.png) |
| `cache` | Cache | [English](en/cache.md) | [PNG](assets/en/cache.png) |
| `dlp` | DLP | [English](en/dlp.md) | [PNG](assets/en/dlp.png) |
| `observability` | Observability | [English](en/observability.md) | [PNG](assets/en/observability.png) |
| `cost` | Cost | [English](en/cost.md) | [PNG](assets/en/cost.png) |
| `settings` | Settings | [English](en/settings.md) | [PNG](assets/en/settings.png) |
| `new-api-key` | New API key | [English](en/new-api-key.md) | [PNG](assets/en/new-api-key.png) |
| `onboarding` | Onboarding | [English](en/onboarding.md) | [PNG](assets/en/onboarding.png) |

## Locale preference

- The console stores the first-access choice in `localStorage` under the key **`dontstop.locale`** as a plain BCP-47 code (`en`, `pt-PT`, `pt-BR`, `es`, `fr`, `de`).
- On first access, `navigator.languages` preselects the closest supported locale. Confirming persists the choice; later visits do not show onboarding.
- Settings applies the new locale immediately and updates `<html lang>` without reloading.
- Missing translation keys fall back to **English**; if a localized guide is missing, the help link opens the English guide; if neither exists, it opens this index.

## Help links

Each page, modal and onboarding screen exposes a contextual **Help** link. Its URL is built from the repository configuration (`repoUrl`, `branch`, `basePath`) plus the active locale and guide slug:

```text
/guide/<locale>/<slug>.html
```

The admin serves the rendered HTML from `admin/public/guide/` (built by `scripts/build-guide-html.js`),
so help links open locally with images and anchors. The static design-system mockup still uses
`admin/src/guide-links.js` (`GUIDE_CONFIG`) and may point to the repository. Links open in a new tab
with `rel="noopener noreferrer"` and translated `aria-label`/`title`.

## Conventions

- **Slug:** stable, lowercase, hyphenated, identical in every language.
- **Image:** `assets/<locale>/<slug>.png`, captured at 1440×1200 from the running console.
- **Alt text:** translated and describes the screen, not the filename.
- **Adding a guide:** add the slug to `scripts/user-guide-content.js`, run `node scripts/generate-user-guide.js`, capture the six images, and update the manifest.
- **Missing translation:** keep the English body and add an explicit exception note at the top of the localized file until it is translated.
