const test = require('node:test');
const assert = require('node:assert/strict');
const prefs = require('../src/preferences');

function memoryStorage(initial) {
  const values = new Map(Object.entries(initial || {}));
  return {
    getItem: key => (values.has(key) ? values.get(key) : null),
    setItem: (key, value) => values.set(key, value)
  };
}

test('first access detects the browser language and shows onboarding', () => {
  const decision = prefs.decide({ stored: null, queryLang: null, onboarding: null, languages: ['fr-CA', 'en'] });
  assert.equal(decision.locale, 'fr');
  assert.equal(decision.source, 'detected');
  assert.equal(decision.showOnboarding, true);
});

test('a stored preference skips onboarding', () => {
  const decision = prefs.decide({ stored: 'de', queryLang: null, onboarding: null, languages: ['en'] });
  assert.equal(decision.locale, 'de');
  assert.equal(decision.source, 'stored');
  assert.equal(decision.showOnboarding, false);
});

test('a query language wins for screenshots and skips onboarding', () => {
  const decision = prefs.decide({ stored: 'de', queryLang: 'es', onboarding: null, languages: ['en'] });
  assert.equal(decision.locale, 'es');
  assert.equal(decision.source, 'query');
  assert.equal(decision.showOnboarding, false);
});

test('forced onboarding still respects the stored locale', () => {
  const decision = prefs.decide({ stored: 'pt-BR', queryLang: null, onboarding: '1', languages: ['en'] });
  assert.equal(decision.locale, 'pt-BR');
  assert.equal(decision.showOnboarding, true);
});

test('locale persistence uses the documented key and BCP-47 value', () => {
  const storage = memoryStorage();
  assert.equal(prefs.saveLocale(storage, 'pt_PT'), 'pt-PT');
  assert.equal(storage.getItem('dontstop.locale'), 'pt-PT');
  assert.equal(prefs.readLocale(storage), 'pt-PT');
});
