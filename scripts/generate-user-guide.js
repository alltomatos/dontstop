const fs = require('fs');
const path = require('path');
const { SHARED, GUIDES } = require('./user-guide-content');

const ROOT = path.resolve(__dirname, '..');
const LOCALES_DIR = path.join(ROOT, 'admin', 'locales');
const GUIDE_DIR = path.join(ROOT, 'docs', 'user-guide');
const ASSETS_DIR = path.join(GUIDE_DIR, 'assets');
const LOCALES = ['en', 'pt-PT', 'pt-BR', 'es', 'fr', 'de'];
const LOCALE_NAMES = {
  en: 'English', 'pt-PT': 'Português (Portugal)', 'pt-BR': 'Português (Brasil)',
  es: 'Español', fr: 'Français', de: 'Deutsch'
};
const WHEN = {
  en: 'Use this screen when you need to inspect, configure or operate this part of the gateway.',
  'pt-BR': 'Use esta tela quando precisar inspecionar, configurar ou operar esta parte do gateway.',
  'pt-PT': 'Utilize este ecrã quando precisar de inspecionar, configurar ou operar esta parte do gateway.',
  es: 'Use esta pantalla cuando necesite inspeccionar, configurar u operar esta parte del gateway.',
  fr: 'Utilisez cet écran pour inspecter, configurer ou exploiter cette partie de la passerelle.',
  de: 'Verwenden Sie diese Seite, um diesen Teil des Gateways zu prüfen, zu konfigurieren oder zu betreiben.'
};

const catalogs = Object.fromEntries(LOCALES.map(locale => [
  locale,
  JSON.parse(fs.readFileSync(path.join(LOCALES_DIR, `${locale}.json`), 'utf8'))
]));

function ensureDirectory(directory) {
  fs.mkdirSync(directory, { recursive: true });
}

function buildGuide(locale, guide) {
  const catalog = catalogs[locale];
  const shared = SHARED[locale];
  const title = catalog[guide.pageKey];
  const what = guide.what[locale];
  const specificStep = guide.step[locale];
  const fields = guide.fields.map(key => ({
    label: catalog[key] || key,
    description: shared.fieldDescription
  }));
  const steps = [
    shared.stepOpen.replace('{page}', title),
    shared.stepReview,
    shared.stepAct,
    specificStep
  ];
  const related = guide.related.map(slug => {
    const target = GUIDES.find(item => item.slug === slug);
    return `- [${catalogs[locale][target.pageKey]}]( ${slug}.md )`.replace('( ', '(').replace(' )', ')');
  }).join('\n');
  const alt = shared.imageAlt.replace('{page}', title).replace('{locale}', LOCALE_NAMES[locale]);
  const image = `../assets/${locale}/${guide.slug}.png`;
  return `# ${title}

![${alt}](${image})

## ${shared.whatTitle}

${what}

## ${shared.whenTitle}

${WHEN[locale]}

## ${shared.stepsTitle}

${steps.map((step, index) => `${index + 1}. ${step}`).join('\n')}

## ${shared.fieldsTitle}

| ${locale === 'en' ? 'Field or action' : titleCaseField(locale)} | ${shared.fieldDescription} |
|---|---|
${fields.map(field => `| ${field.label} | ${field.description} |`).join('\n')}

## ${shared.errorsTitle}

- ${shared.errorLoad}
- ${shared.errorPermission}
- ${shared.errorEmpty}

## ${shared.relatedTitle}

${related}

> ${shared.imageAlt.replace('{page}', title).replace('{locale}', LOCALE_NAMES[locale])}
`;
}

function titleCaseField(locale) {
  return {
    'pt-BR': 'Campo ou ação', 'pt-PT': 'Campo ou ação', es: 'Campo o acción',
    fr: 'Champ ou action', de: 'Feld oder Aktion'
  }[locale] || 'Field or action';
}

function buildLocaleIndex(locale) {
  const catalog = catalogs[locale];
  const shared = SHARED[locale];
  const title = locale === 'en' ? 'User guide' : {
    'pt-BR': 'Guia do usuário', 'pt-PT': 'Guia do utilizador', es: 'Guía del usuario',
    fr: 'Guide utilisateur', de: 'Benutzerhandbuch'
  }[locale];
  const intro = locale === 'en'
    ? 'One guide per console page, modal and workflow. Screenshots are captured in the selected language.'
    : {
      'pt-BR': 'Um guia por página, modal e funcionalidade do console. As capturas são feitas no idioma selecionado.',
      'pt-PT': 'Um guia por página, modal e funcionalidade da consola. As capturas são feitas no idioma selecionado.',
      es: 'Una guía por página, modal y funcionalidad de la consola. Las capturas se realizan en el idioma seleccionado.',
      fr: 'Un guide par page, fenêtre modale et fonctionnalité de la console. Les captures sont réalisées dans la langue sélectionnée.',
      de: 'Ein Leitfaden pro Seite, Modal und Funktion der Konsole. Screenshots werden in der ausgewählten Sprache erstellt.'
    }[locale];
  const headers = locale === 'en'
    ? ['Slug', 'Screen', 'Guide', 'Image']
    : {
      'pt-BR': ['Slug', 'Tela', 'Guia', 'Imagem'],
      'pt-PT': ['Slug', 'Ecrã', 'Guia', 'Imagem'],
      es: ['Slug', 'Pantalla', 'Guía', 'Imagen'],
      fr: ['Slug', 'Écran', 'Guide', 'Image'],
      de: ['Slug', 'Seite', 'Leitfaden', 'Bild']
    }[locale];
  const rows = GUIDES.map(guide => {
    const page = catalog[guide.pageKey];
    return `| \`${guide.slug}\` | ${page} | [${page}](${guide.slug}.md) | [PNG](../assets/${locale}/${guide.slug}.png) |`;
  }).join('\n');
  return `# ${title} — ${LOCALE_NAMES[locale]}

${intro}

[← ${locale === 'en' ? 'All languages' : 'Todos os idiomas'}](../README.md)

| ${headers.join(' | ')} |
|---|---|
${rows}
`;
}

function buildGlobalIndex() {
  const languageRows = LOCALES.map(locale => {
    const label = LOCALE_NAMES[locale];
    return `| \`${locale}\` | ${label} | [${label}]( ${locale}/README.md ) |`.replace('( ', '(').replace(' )', ')');
  }).join('\n');
  const guideRows = GUIDES.map(guide => {
    const title = catalogs.en[guide.pageKey];
    return `| \`${guide.slug}\` | ${title} | [English](en/${guide.slug}.md) | [PNG](assets/en/${guide.slug}.png) |`;
  }).join('\n');
  return `# DonStop IA — User guide

> Versioned user guide for every console page, modal and workflow. Six languages share the same slugs.
> **Fallback:** if a localized guide is missing, the help link opens the English guide; if no guide exists, it opens this index.

## Languages

| Code | Language | Index |
|---|---|---|
${languageRows}

## Guides

| Slug | Screen | English guide | Image |
|---|---|---|---|
${guideRows}

## Locale preference

- The console stores the first-access choice in \`localStorage\` under the key **\`dontstop.locale\`** as a plain BCP-47 code (\`en\`, \`pt-PT\`, \`pt-BR\`, \`es\`, \`fr\`, \`de\`).
- On first access, \`navigator.languages\` preselects the closest supported locale. Confirming persists the choice; later visits do not show onboarding.
- Settings applies the new locale immediately and updates \`<html lang>\` without reloading.
- Missing translation keys fall back to **English**; if a localized guide is missing, the help link opens the English guide; if neither exists, it opens this index.

## Help links

Each page, modal and onboarding screen exposes a contextual **Help** link. Its URL is built from the repository configuration (\`repoUrl\`, \`branch\`, \`basePath\`) plus the active locale and guide slug:

\`\`\`text
https://github.com/alltomatos/dontstop/blob/main/docs/user-guide/<locale>/<slug>.md
\`\`\`

Configure the base URL and branch in \`admin/src/guide-links.js\` (\`GUIDE_CONFIG\`). The link opens in a new tab with \`rel="noopener noreferrer"\` and translated \`aria-label\`/\`title\`.

## Conventions

- **Slug:** stable, lowercase, hyphenated, identical in every language.
- **Image:** \`assets/<locale>/<slug>.png\`, captured at 1440×1200 from the running console.
- **Alt text:** translated and describes the screen, not the filename.
- **Adding a guide:** add the slug to \`scripts/user-guide-content.js\`, run \`node scripts/generate-user-guide.js\`, capture the six images, and update the manifest.
- **Missing translation:** keep the English body and add an explicit exception note at the top of the localized file until it is translated.
`;
}

function main() {
  ensureDirectory(GUIDE_DIR);
  ensureDirectory(ASSETS_DIR);
  for (const locale of LOCALES) {
    const localeDir = path.join(GUIDE_DIR, locale);
    ensureDirectory(localeDir);
    ensureDirectory(path.join(ASSETS_DIR, locale));
    for (const guide of GUIDES) {
      fs.writeFileSync(path.join(localeDir, `${guide.slug}.md`), buildGuide(locale, guide), 'utf8');
    }
    fs.writeFileSync(path.join(localeDir, 'README.md'), buildLocaleIndex(locale), 'utf8');
  }
  fs.writeFileSync(path.join(GUIDE_DIR, 'README.md'), buildGlobalIndex(), 'utf8');
  const manifest = {
    defaultLocale: 'en',
    locales: LOCALES,
    guides: Object.fromEntries(GUIDES.map(guide => [guide.slug, { locales: LOCALES }]))
  };
  fs.writeFileSync(path.join(GUIDE_DIR, 'manifest.json'), JSON.stringify(manifest, null, 2) + '\n', 'utf8');
  console.log(`Generated ${GUIDES.length * LOCALES.length} guides in ${GUIDE_DIR}`);
}

main();
