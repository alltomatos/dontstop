const fs = require('fs');
const path = require('path');
const renderer = require('../admin/src/guide-renderer.js');

const ROOT = path.resolve(__dirname, '..');
const GUIDE_DIR = path.join(ROOT, 'docs', 'user-guide');
const OUTPUT_DIR = path.join(ROOT, 'admin', 'public', 'guide');

const { LOCALE_NAMES, escapeHtml, renderMarkdown, renderPage, extractTitle } = renderer;

function copyAssets() {
  const source = path.join(GUIDE_DIR, 'assets');
  const target = path.join(OUTPUT_DIR, 'assets');
  if (!fs.existsSync(source)) return;
  fs.cpSync(source, target, { recursive: true });
}

function buildIndex(manifest) {
  const rows = Object.keys(manifest.guides).sort().map(slug => {
    const links = manifest.locales
      .map(locale => `<a href="${locale}/${slug}.html">${LOCALE_NAMES[locale] || locale}</a>`)
      .join(' · ');
    return `<tr><td><code>${escapeHtml(slug)}</code></td><td>${links}</td></tr>`;
  }).join('\n');
  return renderPage('Guias do usuário', `<h1>Guias do usuário</h1>
<p>Documentação versionada por página e funcionalidade, nos seis idiomas suportados.</p>
<table><thead><tr><th>Guia</th><th>Idiomas</th></tr></thead><tbody>${rows}</tbody></table>`, 'pt-BR');
}

function build() {
  const manifestFile = path.join(GUIDE_DIR, 'manifest.json');
  const manifest = JSON.parse(fs.readFileSync(manifestFile, 'utf8'));
  fs.mkdirSync(OUTPUT_DIR, { recursive: true });
  copyAssets();

  let written = 0;
  for (const locale of manifest.locales) {
    const localeDir = path.join(OUTPUT_DIR, locale);
    fs.mkdirSync(localeDir, { recursive: true });
    for (const slug of Object.keys(manifest.guides)) {
      const source = path.join(GUIDE_DIR, locale, `${slug}.md`);
      if (!fs.existsSync(source)) {
        console.warn(`missing guide ${locale}/${slug}.md`);
        continue;
      }
      const markdown = fs.readFileSync(source, 'utf8');
      const title = extractTitle(markdown, slug);
      const html = renderPage(title, renderMarkdown(markdown), locale);
      fs.writeFileSync(path.join(localeDir, `${slug}.html`), html, 'utf8');
      written += 1;
    }
  }
  fs.writeFileSync(path.join(OUTPUT_DIR, 'index.html'), buildIndex(manifest), 'utf8');
  console.log(`guide html: ${written} pages + index -> ${path.relative(ROOT, OUTPUT_DIR)}`);
}

module.exports = { renderMarkdown, renderPage, rewriteHref: renderer.rewriteHref, extractTitle, build };

if (require.main === module) {
  build();
}
