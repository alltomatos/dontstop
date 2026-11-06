const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');

const root = path.resolve(__dirname, '..', '..');
const guideDir = path.join(root, 'docs', 'user-guide');
const manifest = JSON.parse(fs.readFileSync(path.join(guideDir, 'manifest.json'), 'utf8'));
const locales = ['en', 'pt-PT', 'pt-BR', 'es', 'fr', 'de'];
const slugs = Object.keys(manifest.guides);

test('manifest covers every locale and guide', () => {
  assert.equal(manifest.defaultLocale, 'en');
  assert.deepEqual(manifest.locales, locales);
  assert.equal(slugs.length, 12);
  for (const slug of slugs) assert.deepEqual(manifest.guides[slug].locales, locales);
});

test('every guide exists with image, alt, steps, fields and errors', () => {
  for (const locale of locales) {
    for (const slug of slugs) {
      const file = path.join(guideDir, locale, `${slug}.md`);
      assert.ok(fs.existsSync(file), `${locale}/${slug}.md is missing`);
      const body = fs.readFileSync(file, 'utf8');
      assert.match(body, /^# .+/m, `${locale}/${slug} has no title`);
      assert.match(body, /!\[[^\]]+\]\(\.\.\/assets\//, `${locale}/${slug} has no image with alt`);
      assert.match(body, /## /, `${locale}/${slug} has no sections`);
      assert.match(body, /^## .*\n\n\|/m, `${locale}/${slug} has no fields table`);
      assert.match(body, /^## .*\n\n-/m, `${locale}/${slug} has no error list`);
    }
  }
});

test('all screenshots are captured in the matching language', () => {
  for (const locale of locales) {
    for (const slug of slugs) {
      const image = path.join(guideDir, 'assets', locale, `${slug}.png`);
      assert.ok(fs.existsSync(image), `${locale}/${slug}.png is missing`);
      assert.ok(fs.statSync(image).size > 5000, `${locale}/${slug}.png looks empty`);
    }
  }
});

test('global and per-language indexes link guides and images', () => {
  const global = fs.readFileSync(path.join(guideDir, 'README.md'), 'utf8');
  assert.match(global, /dontstop\.locale/);
  assert.match(global, /en\/dashboard\.md/);
  for (const locale of locales) {
    const index = fs.readFileSync(path.join(guideDir, locale, 'README.md'), 'utf8');
    for (const slug of slugs) {
      assert.match(index, new RegExp(`${slug}\\.md`));
      assert.match(index, new RegExp(`assets/${locale}/${slug}\\.png`));
    }
  }
});
