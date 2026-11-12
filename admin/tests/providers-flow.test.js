const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');

const root = path.join(__dirname, '..', '..');
const html = fs.readFileSync(path.join(root, 'design-system', 'mockups', 'console.html'), 'utf8');
const locales = ['en', 'pt-BR', 'pt-PT', 'es', 'fr', 'de'];
const requiredKeys = [
  'providers.configuredTitle',
  'providers.configuredSubtitle',
  'providers.emptyTitle',
  'providers.emptyBody',
  'providers.catalogTitle',
  'providers.catalogSubtitle',
  'providers.catalogSearch',
  'providers.detailSubtitle',
  'providers.linkedAccounts',
  'providers.addAccount',
  'providers.accountName',
  'providers.accountCredential',
  'providers.saveAccount',
  'providers.origin.all',
  'providers.origin.native',
  'providers.origin.custom',
  'providers.origin.label',
  'providers.guideLink',
  'providers.signupLink',
  'providers.configureTitle',
  'providers.configureSubmit',
  'providers.configureFailed',
  'providers.configureConflict',
  'providers.configureUnsupported',
  'providers.notConfiguredTitle',
  'providers.setupGuide',
  'providers.catalogEmptyTitle',
  'providers.modelsTitle',
  'providers.syncModels',
  'providers.modelTest',
  'providers.activateModel',
  'providers.deactivateModel',
  'providers.modelUpdateFailed',
  'providers.testAllModels',
  'providers.testQueueFailed',
  'providers.filterPrice',
  'providers.priceFree',
  'providers.capAudio',
  'providers.filteredCount',
  'providers.selectAllFiltered',
  'providers.bulkActivate',
  'providers.bulkConfirm',
  'providers.bulkDone',
  'providers.metadataUnknown',
  'providers.modelsEmptyBody'
];

function section(id) {
  const start = html.indexOf(`<section id="${id}"`);
  assert.ok(start >= 0, `missing section ${id}`);
  const end = html.indexOf('</section>', start);
  assert.ok(end > start, `unterminated section ${id}`);
  return html.slice(start, end);
}

test('main providers page lists only configured providers', () => {
  const main = section('providers');
  assert.equal((main.match(/data-provider-id=/g) || []).length, 8);
  assert.ok(!main.includes('data-catalog-item'));
  assert.ok(main.includes('id="configured-empty"'));
  assert.ok(main.includes('href="#provider-catalog"'));
  assert.ok(main.includes('data-origin-group="configured"'));
});

test('catalog is a separate page and links to provider detail', () => {
  const catalog = section('provider-catalog');
  assert.equal((catalog.match(/<article[^>]*data-catalog-item/g) || []).length, 12);
  assert.ok(catalog.includes('href="#provider-detail?provider=naga"'));
  assert.ok(catalog.includes('href="#provider-detail?provider=agentrouter"'));
  assert.ok(catalog.includes('data-origin-group="catalog"'));
  assert.ok(html.includes("PROVIDER_ORIGINS = Object.fromEntries"));
  assert.ok(!catalog.includes('data-provider-id="openai"'));
});

test('provider detail exposes accounts and the add-account flow', () => {
  const detail = section('provider-detail');
  assert.ok(detail.includes('id="detail-accounts"'));
  assert.ok(detail.includes('id="add-account-btn"'));
  assert.ok(detail.includes('href="#providers"'));
  assert.ok(detail.includes('id="detail-guide"'));
  assert.ok(detail.includes('id="detail-signup"'));
  assert.ok(html.includes('id="account-modal"'));
  assert.ok(html.includes('id="account-form"'));
});

test('every data-i18n key used by the mockup exists in the catalogs', () => {
  const keys = [...html.matchAll(/data-i18n(?:-[a-z-]+)?="([^"]+)"/g)].map(match => match[1]);
  const en = JSON.parse(fs.readFileSync(path.join(root, 'admin', 'locales', 'en.json'), 'utf8'));
  for (const key of new Set(keys)) {
    assert.equal(typeof en[key], 'string', `missing English key ${key}`);
  }
});

test('provider flow strings exist in every locale', () => {
  for (const locale of locales) {
    const catalog = JSON.parse(fs.readFileSync(path.join(root, 'admin', 'locales', `${locale}.json`), 'utf8'));
    for (const key of requiredKeys) {
      assert.equal(typeof catalog[key], 'string', `${locale} is missing ${key}`);
      assert.ok(catalog[key].trim().length > 0, `${locale}.${key} is empty`);
    }
  }
});
