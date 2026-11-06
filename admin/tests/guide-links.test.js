const test = require('node:test');
const assert = require('node:assert/strict');
const guides = require('../src/guide-links');

const manifest = {
  defaultLocale: 'en',
  locales: ['en', 'pt-BR'],
  guides: {
    dashboard: { locales: ['en', 'pt-BR'] },
    onlyEnglish: { locales: ['en'] }
  }
};

test('resolves the guide for the current locale', () => {
  assert.equal(
    guides.resolveGuideUrl('pt-BR', 'dashboard', manifest),
    'https://github.com/alltomatos/dontstop/blob/main/docs/user-guide/pt-BR/dashboard.md'
  );
});

test('falls back to English and then to the index', () => {
  assert.equal(
    guides.resolveGuideUrl('de', 'dashboard', manifest),
    'https://github.com/alltomatos/dontstop/blob/main/docs/user-guide/en/dashboard.md'
  );
  assert.equal(
    guides.resolveGuideUrl('pt-BR', 'onlyEnglish', manifest),
    'https://github.com/alltomatos/dontstop/blob/main/docs/user-guide/en/onlyEnglish.md'
  );
  assert.equal(
    guides.resolveGuideUrl('pt-BR', 'missing', manifest),
    'https://github.com/alltomatos/dontstop/blob/main/docs/user-guide/README.md'
  );
});

test('accepts a repository and branch override', () => {
  assert.equal(
    guides.resolveGuideUrl('en', 'dashboard', manifest, { repoUrl: 'https://example.test/repo', branch: 'release' }),
    'https://example.test/repo/blob/release/docs/user-guide/en/dashboard.md'
  );
});
