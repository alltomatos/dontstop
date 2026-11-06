const test = require('node:test');
const assert = require('node:assert/strict');
const i18n = require('../src/i18n');

test('detects the best browser locale', () => {
  assert.equal(i18n.detectLocale(['de-DE', 'en-US']), 'de');
  assert.equal(i18n.detectLocale(['pt-BR', 'en']), 'pt-BR');
  assert.equal(i18n.detectLocale(['pt']), 'pt-BR');
  assert.equal(i18n.detectLocale(['zh-CN']), 'en');
});

test('normalizes and resolves locales with English fallback', () => {
  assert.equal(i18n.normalizeLocale('pt_PT'), 'pt-PT');
  assert.equal(i18n.resolveLocale('fr-CA', i18n.CODES), 'fr');
  assert.equal(i18n.resolveLocale('it', i18n.CODES), 'en');
  assert.equal(i18n.resolveLocale('de', ['pt-BR', 'en']), 'en');
});

test('translates with English fallback and never breaks on missing keys', () => {
  const en = { 'common.help': 'Help', 'summary': '{count} items' };
  const fr = { 'common.help': 'Aide' };
  assert.equal(i18n.translate(fr, 'common.help', {}, en), 'Aide');
  assert.equal(i18n.translate(fr, 'summary', { count: 3 }, en), '3 items');
  assert.equal(i18n.translate(fr, 'missing.key', {}, en), 'missing.key');
});

test('pluralization uses Intl.PluralRules', () => {
  const en = { 'providers.connected.one': '1 connected', 'providers.connected.other': '{count} connected' };
  const pt = { 'providers.connected.one': '1 conectada', 'providers.connected.other': '{count} conectadas' };
  assert.equal(i18n.translatePlural(en, 'providers.connected', 1, {}, en, 'en'), '1 connected');
  assert.equal(i18n.translatePlural(en, 'providers.connected', 2, {}, en, 'en'), '2 connected');
  assert.equal(i18n.translatePlural(pt, 'providers.connected', 2, {}, pt, 'pt-BR'), '2 conectadas');
});

test('formats dates, numbers and currency with Intl', () => {
  assert.equal(i18n.formatNumber('de', 1234.5), new Intl.NumberFormat('de').format(1234.5));
  assert.equal(i18n.formatCurrency('en', 12.5, 'USD'), new Intl.NumberFormat('en', { style: 'currency', currency: 'USD' }).format(12.5));
  assert.equal(typeof i18n.formatDate('fr', '2026-01-02'), 'string');
});
