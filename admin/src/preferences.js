(function (root, factory) {
  const api = factory(root && root.DontstopI18n ? root.DontstopI18n : require('./i18n'));
  if (typeof module === 'object' && module.exports) module.exports = api;
  if (root) root.DontstopPreferences = api;
})(typeof globalThis !== 'undefined' ? globalThis : this, function (I18N) {
  const STORAGE_KEY = I18N.STORAGE_KEY;

  function readLocale(storage) {
    try { return I18N.normalizeLocale(storage.getItem(STORAGE_KEY)); } catch { return null; }
  }

  function saveLocale(storage, locale) {
    const normalized = I18N.normalizeLocale(locale);
    if (!normalized) return null;
    try { storage.setItem(STORAGE_KEY, normalized); } catch { /* storage disabled */ }
    return normalized;
  }

  function decide(input) {
    const options = input || {};
    const queryLocale = I18N.normalizeLocale(options.queryLang);
    const storedLocale = I18N.normalizeLocale(options.stored);
    const forced = options.onboarding === '1' || options.onboarding === true;
    const languages = Array.isArray(options.languages) ? options.languages : [];
    let locale = I18N.DEFAULT_LOCALE;
    let source = 'detected';
    if (queryLocale) { locale = queryLocale; source = 'query'; }
    else if (storedLocale) { locale = storedLocale; source = 'stored'; }
    else {
      locale = I18N.detectLocale(languages);
      source = 'detected';
    }
    return {
      locale,
      source,
      showOnboarding: forced || (!storedLocale && !queryLocale),
      persist: false
    };
  }

  return { STORAGE_KEY, readLocale, saveLocale, decide };
});
