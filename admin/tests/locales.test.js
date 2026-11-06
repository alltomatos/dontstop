const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');

const localesDir = path.join(__dirname, '..', 'locales');
const locales = ['en', 'pt-PT', 'pt-BR', 'es', 'fr', 'de'];
const catalogs = Object.fromEntries(locales.map(locale => [
  locale,
  JSON.parse(fs.readFileSync(path.join(localesDir, `${locale}.json`), 'utf8'))
]));

test('every locale is valid JSON with the same semantic keys', () => {
  const reference = Object.keys(catalogs.en).sort();
  for (const locale of locales) {
    assert.deepEqual(Object.keys(catalogs[locale]).sort(), reference, `${locale} has different keys`);
  }
});

test('no locale has empty translations', () => {
  for (const locale of locales) {
    for (const [key, value] of Object.entries(catalogs[locale])) {
      assert.equal(typeof value, 'string', `${locale}.${key} must be a string`);
      assert.ok(value.trim().length > 0, `${locale}.${key} is empty`);
    }
  }
});

test('locale metadata is complete', () => {
  assert.equal(catalogs.en['meta.name'], 'English');
  assert.ok(catalogs['pt-BR']['meta.name']);
  assert.ok(catalogs['pt-PT']['meta.name']);
  assert.ok(catalogs.es['meta.name']);
  assert.ok(catalogs.fr['meta.name']);
  assert.ok(catalogs.de['meta.name']);
});
