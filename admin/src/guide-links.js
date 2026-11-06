(function (root, factory) {
  const api = factory();
  if (typeof module === 'object' && module.exports) module.exports = api;
  if (root) root.DontstopGuideLinks = api;
})(typeof globalThis !== 'undefined' ? globalThis : this, function () {
  const GUIDE_CONFIG = {
    repoUrl: 'https://github.com/alltomatos/dontstop',
    branch: 'main',
    basePath: 'docs/user-guide',
    defaultLocale: 'en'
  };

  function normalizeLocale(locale, available) {
    const pool = Array.isArray(available) && available.length ? available : [GUIDE_CONFIG.defaultLocale];
    if (pool.includes(locale)) return locale;
    if (pool.includes(GUIDE_CONFIG.defaultLocale)) return GUIDE_CONFIG.defaultLocale;
    return pool[0];
  }

  function guideLocale(slug, locale, manifest) {
    if (!slug || !manifest || !manifest.guides || !manifest.guides[slug]) return null;
    const guide = manifest.guides[slug];
    const locales = Array.isArray(guide.locales) ? guide.locales : manifest.locales;
    return normalizeLocale(locale, locales);
  }

  function blobUrl(path, config) {
    const active = { ...GUIDE_CONFIG, ...(config || {}) };
    return `${active.repoUrl}/blob/${active.branch}/${path}`;
  }

  function resolveGuideUrl(locale, slug, manifest, config) {
    const active = { ...GUIDE_CONFIG, ...(config || {}) };
    const resolvedLocale = guideLocale(slug, locale, manifest);
    if (resolvedLocale) {
      return blobUrl(`${active.basePath}/${resolvedLocale}/${slug}.md`, active);
    }
    return blobUrl(`${active.basePath}/README.md`, active);
  }

  function helpLabel(locale, pageTitle, translate) {
    if (typeof translate === 'function') {
      return translate(locale, 'common.helpFor', { page: pageTitle });
    }
    return `Help for ${pageTitle}`;
  }

  return {
    GUIDE_CONFIG,
    resolveGuideUrl,
    guideLocale,
    helpLabel
  };
});
