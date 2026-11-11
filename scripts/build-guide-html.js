const fs = require('fs');
const path = require('path');

const ROOT = path.resolve(__dirname, '..');
const GUIDE_DIR = path.join(ROOT, 'docs', 'user-guide');
const OUTPUT_DIR = path.join(ROOT, 'admin', 'public', 'guide');
const LOCALE_NAMES = {
  en: 'English',
  'pt-PT': 'Português (Portugal)',
  'pt-BR': 'Português (Brasil)',
  es: 'Español',
  fr: 'Français',
  de: 'Deutsch'
};

function escapeHtml(value) {
  return String(value)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');
}

function rewriteHref(href) {
  const value = String(href || '').trim();
  if (!value) return value;
  if (/^(https?:|mailto:|#)/i.test(value)) return value;
  return value.replace(/\.md(?=#|$)/, '.html');
}

function inline(text) {
  let output = escapeHtml(text);
  output = output.replace(/!\[([^\]]*)\]\(([^)]+)\)/g, (_match, alt, src) =>
    `<img src="${rewriteHref(src)}" alt="${escapeHtml(alt)}" loading="lazy">`);
  output = output.replace(/\[([^\]]+)\]\(([^)]+)\)/g, (_match, label, href) =>
    `<a href="${rewriteHref(href)}">${label}</a>`);
  output = output.replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>');
  output = output.replace(/`([^`]+)`/g, '<code>$1</code>');
  return output;
}

function splitRow(line) {
  return line.trim().replace(/^\|/, '').replace(/\|$/, '').split('|').map(cell => cell.trim());
}

function isSeparatorRow(line) {
  return /^\|?[\s:-]+\|[\s:|-]*$/.test(line.trim());
}

function renderMarkdown(markdown) {
  const lines = String(markdown || '').replace(/\r\n/g, '\n').split('\n');
  const html = [];
  let paragraph = [];
  let list = null;
  let table = null;
  let code = null;
  let quote = null;

  const flushParagraph = () => {
    if (paragraph.length === 0) return;
    html.push(`<p>${inline(paragraph.join(' '))}</p>`);
    paragraph = [];
  };
  const flushList = () => {
    if (!list) return;
    html.push(`<${list.type}>${list.items.map(item => `<li>${inline(item)}</li>`).join('')}</${list.type}>`);
    list = null;
  };
  const flushTable = () => {
    if (!table) return;
    const [header, ...rows] = table;
    html.push('<table><thead><tr>' + header.map(cell => `<th>${inline(cell)}</th>`).join('') + '</tr></thead><tbody>');
    for (const row of rows) {
      html.push('<tr>' + row.map(cell => `<td>${inline(cell)}</td>`).join('') + '</tr>');
    }
    html.push('</tbody></table>');
    table = null;
  };
  const flushQuote = () => {
    if (!quote) return;
    html.push(`<blockquote><p>${inline(quote.join(' '))}</p></blockquote>`);
    quote = null;
  };
  const flushAll = () => {
    flushParagraph();
    flushList();
    flushTable();
    flushQuote();
  };

  for (const rawLine of lines) {
    const line = rawLine.replace(/\s+$/, '');

    if (code) {
      if (line.trim().startsWith('```')) {
        html.push(`<pre><code>${escapeHtml(code.join('\n'))}</code></pre>`);
        code = null;
      } else {
        code.push(rawLine);
      }
      continue;
    }
    if (line.trim().startsWith('```')) {
      flushAll();
      code = [];
      continue;
    }

    if (!line.trim()) {
      flushAll();
      continue;
    }

    if (/^<a id="[^"]+"><\/a>$/.test(line.trim())) {
      flushAll();
      html.push(line.trim());
      continue;
    }

    const heading = /^(#{1,3})\s+(.*)$/.exec(line);
    if (heading) {
      flushAll();
      const level = heading[1].length;
      html.push(`<h${level}>${inline(heading[2].trim())}</h${level}>`);
      continue;
    }

    if (line.trim().startsWith('|')) {
      flushParagraph();
      flushList();
      flushQuote();
      if (table && isSeparatorRow(line) && table.length === 1) continue;
      if (!table) table = [];
      table.push(splitRow(line));
      continue;
    }
    flushTable();

    const item = /^(\s*)([-*]|\d+\.)\s+(.*)$/.exec(line);
    if (item) {
      flushParagraph();
      flushQuote();
      const type = /\d+\./.test(item[2]) ? 'ol' : 'ul';
      if (!list || list.type !== type) {
        flushList();
        list = { type, items: [] };
      }
      list.items.push(item[3]);
      continue;
    }
    flushList();

    if (line.trim().startsWith('> ')) {
      flushParagraph();
      if (!quote) quote = [];
      quote.push(line.trim().slice(2));
      continue;
    }
    flushQuote();

    if (/^---+$/.test(line.trim())) {
      flushParagraph();
      html.push('<hr>');
      continue;
    }

    paragraph.push(line.trim());
  }

  flushAll();
  if (code) html.push(`<pre><code>${escapeHtml(code.join('\n'))}</code></pre>`);
  return html.join('\n');
}

const STYLES = `
:root { color-scheme: light; }
* { box-sizing: border-box; }
body {
  margin: 0; padding: 40px 24px 64px; background: #f5f6fa; color: #1c2541;
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
  line-height: 1.6;
}
main { max-width: 980px; margin: 0 auto; background: #fff; border: 1px solid #e2e5ee; border-radius: 14px; padding: 36px 44px; }
h1, h2, h3 { color: #0f1c3f; line-height: 1.25; }
h1 { font-size: 30px; margin: 0 0 20px; }
h2 { font-size: 21px; margin: 32px 0 12px; padding-top: 8px; border-top: 1px solid #eef0f6; }
h3 { font-size: 17px; margin: 24px 0 8px; }
p, li { font-size: 15px; }
a { color: #1f4fd1; }
a:focus-visible, img:focus-visible { outline: 3px solid #1f4fd1; outline-offset: 2px; }
img { display: block; max-width: 100%; height: auto; margin: 12px 0; border: 1px solid #dfe3ee; border-radius: 10px; background: #fff; }
code { background: #eef1f8; border-radius: 5px; padding: 1px 5px; font-size: 13px; }
pre { background: #0f1c3f; color: #eef1f8; padding: 16px; border-radius: 10px; overflow: auto; }
pre code { background: transparent; color: inherit; padding: 0; }
table { width: 100%; border-collapse: collapse; margin: 12px 0 18px; font-size: 14px; }
th, td { text-align: left; padding: 8px 10px; border-bottom: 1px solid #e7eaf3; vertical-align: top; }
th { background: #f2f4fa; font-weight: 700; }
blockquote { margin: 14px 0; padding: 10px 16px; border-left: 4px solid #1f4fd1; background: #f2f5ff; border-radius: 0 8px 8px 0; }
ul, ol { padding-left: 24px; }
li { margin: 4px 0; }
hr { border: 0; border-top: 1px solid #e7eaf3; margin: 28px 0; }
@media (max-width: 640px) { main { padding: 22px 18px; } body { padding: 16px 10px 40px; } }
`;

function renderPage(title, bodyHtml, locale) {
  const localeLabel = LOCALE_NAMES[locale] || locale;
  return `<!doctype html>
<html lang="${locale}">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>${escapeHtml(title)} — DonStop IA (${escapeHtml(localeLabel)})</title>
<style>${STYLES}</style>
</head>
<body>
<main>
${bodyHtml}
</main>
</body>
</html>
`;
}

function extractTitle(markdown, fallback) {
  const match = /^#\s+(.+)$/m.exec(String(markdown || ''));
  return match ? match[1].trim() : fallback;
}

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

module.exports = { renderMarkdown, renderPage, rewriteHref, extractTitle, build };

if (require.main === module) {
  build();
}
