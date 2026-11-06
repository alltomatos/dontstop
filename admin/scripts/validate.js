const fs = require('fs');
const path = require('path');

const ROOT = path.resolve(__dirname, '..', '..');
const ADMIN = path.join(ROOT, 'admin');
const LOCALES = ['en', 'pt-PT', 'pt-BR', 'es', 'fr', 'de'];

function fail(message) {
  const error = new Error(message);
  error.validation = true;
  throw error;
}

function validate() {
  const catalogs = {};
  for (const locale of LOCALES) {
    const file = path.join(ADMIN, 'locales', `${locale}.json`);
    if (!fs.existsSync(file)) fail(`Missing locale file: ${file}`);
    try {
      catalogs[locale] = JSON.parse(fs.readFileSync(file, 'utf8'));
    } catch (error) {
      fail(`Invalid JSON in ${locale}.json: ${error.message}`);
    }
  }
  const reference = Object.keys(catalogs.en).sort();
  for (const locale of LOCALES) {
    const keys = Object.keys(catalogs[locale]).sort();
    if (JSON.stringify(keys) !== JSON.stringify(reference)) {
      const missing = reference.filter(key => !(key in catalogs[locale]));
      const extra = keys.filter(key => !(key in catalogs.en));
      fail(`${locale}.json key mismatch. missing=${missing.slice(0, 5)} extra=${extra.slice(0, 5)}`);
    }
    for (const [key, value] of Object.entries(catalogs[locale])) {
      if (typeof value !== 'string' || !value.trim()) fail(`${locale}.${key} must be a non-empty string`);
    }
  }

  require(path.join(ADMIN, 'src', 'i18n.js'));
  require(path.join(ADMIN, 'src', 'preferences.js'));
  require(path.join(ADMIN, 'src', 'guide-links.js'));
  require(path.join(ADMIN, 'src', 'source-keys.js'));

  const manifestFile = path.join(ROOT, 'docs', 'user-guide', 'manifest.json');
  if (!fs.existsSync(manifestFile)) fail('Missing docs/user-guide/manifest.json');
  const manifest = JSON.parse(fs.readFileSync(manifestFile, 'utf8'));
  if (manifest.defaultLocale !== 'en') fail('Guide manifest defaultLocale must be en');
  const slugs = Object.keys(manifest.guides || {});
  if (!slugs.length) fail('Guide manifest has no guides');

  for (const locale of LOCALES) {
    const index = path.join(ROOT, 'docs', 'user-guide', locale, 'README.md');
    if (!fs.existsSync(index)) fail(`Missing guide index: ${locale}/README.md`);
    const indexBody = fs.readFileSync(index, 'utf8');
    for (const slug of slugs) {
      const file = path.join(ROOT, 'docs', 'user-guide', locale, `${slug}.md`);
      const image = path.join(ROOT, 'docs', 'user-guide', 'assets', locale, `${slug}.png`);
      if (!fs.existsSync(file)) fail(`Missing guide: ${locale}/${slug}.md`);
      if (!fs.existsSync(image)) fail(`Missing screenshot: assets/${locale}/${slug}.png`);
      if (fs.statSync(image).size < 5000) fail(`Screenshot looks empty: assets/${locale}/${slug}.png`);
      const body = fs.readFileSync(file, 'utf8');
      if (!/^# .+/m.test(body)) fail(`${locale}/${slug}.md has no title`);
      if (!/!\[[^\]]+\]\(\.\.\/assets\//.test(body)) fail(`${locale}/${slug}.md has no image with alt`);
      if (!body.includes('## ')) fail(`${locale}/${slug}.md has no sections`);
      if (!indexBody.includes(`${slug}.md`)) fail(`${locale}/README.md does not link ${slug}`);
      if (!indexBody.includes(`assets/${locale}/${slug}.png`)) fail(`${locale}/README.md does not link the ${slug} image`);
    }
  }

  return {
    locales: LOCALES,
    keys: reference.length,
    guides: slugs.length,
    docs: slugs.length * LOCALES.length,
    images: slugs.length * LOCALES.length
  };
}

if (require.main === module) {
  try {
    const report = validate();
    console.log(`OK: ${report.locales.length} locales, ${report.keys} keys, ${report.guides} guides, ${report.docs} docs, ${report.images} images.`);
  } catch (error) {
    console.error(`FAIL: ${error.message}`);
    process.exit(1);
  }
}

module.exports = { validate, LOCALES };
