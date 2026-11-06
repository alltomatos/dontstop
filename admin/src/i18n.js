(function (root, factory) {
  const api = factory();
  if (typeof module === 'object' && module.exports) module.exports = api;
  if (root) root.DontstopI18n = api;
})(typeof globalThis !== 'undefined' ? globalThis : this, function () {
  const SUPPORTED_LOCALES = [
    { code: 'en', name: 'English', dir: 'ltr' },
    { code: 'pt-PT', name: 'Português (Portugal)', dir: 'ltr' },
    { code: 'pt-BR', name: 'Português (Brasil)', dir: 'ltr' },
    { code: 'es', name: 'Español', dir: 'ltr' },
    { code: 'fr', name: 'Français', dir: 'ltr' },
    { code: 'de', name: 'Deutsch', dir: 'ltr' }
  ];
  const DEFAULT_LOCALE = 'en';
  const STORAGE_KEY = 'dontstop.locale';
  const CODES = SUPPORTED_LOCALES.map(item => item.code);

  function normalizeLocale(input) {
    if (!input || typeof input !== 'string') return null;
    const raw = input.trim().replace(/_/g, '-');
    if (!raw) return null;
    const lower = raw.toLowerCase();
    if (lower === 'pt' || lower === 'pt-br') return 'pt-BR';
    if (lower === 'pt-pt') return 'pt-PT';
    if (lower === 'en' || lower.startsWith('en-')) return 'en';
    if (lower === 'es' || lower.startsWith('es-')) return 'es';
    if (lower === 'fr' || lower.startsWith('fr-')) return 'fr';
    if (lower === 'de' || lower.startsWith('de-')) return 'de';
    return CODES.find(code => code.toLowerCase() === lower) || null;
  }

  function isSupported(input) {
    return CODES.includes(input);
  }

  function detectLocale(languages) {
    const list = Array.isArray(languages) ? languages : [];
    for (const language of list) {
      const normalized = normalizeLocale(language);
      if (normalized) return normalized;
    }
    return DEFAULT_LOCALE;
  }

  function resolveLocale(input, available) {
    const pool = Array.isArray(available) && available.length ? available : CODES;
    const normalized = normalizeLocale(input);
    if (normalized && pool.includes(normalized)) return normalized;
    if (pool.includes(DEFAULT_LOCALE)) return DEFAULT_LOCALE;
    return pool[0] || DEFAULT_LOCALE;
  }

  function getValue(source, key) {
    if (!source || !key) return undefined;
    if (Object.prototype.hasOwnProperty.call(source, key)) return source[key];
    return key.split('.').reduce((value, part) => {
      if (value && typeof value === 'object' && part in value) return value[part];
      return undefined;
    }, source);
  }

  function interpolate(template, values) {
    if (typeof template !== 'string') return template;
    if (!values) return template;
    return template.replace(/\{(\w+)\}/g, (match, name) => (
      Object.prototype.hasOwnProperty.call(values, name) ? String(values[name]) : match
    ));
  }

  function translate(catalog, key, values, fallbackCatalog) {
    const value = getValue(catalog, key);
    if (typeof value === 'string') return interpolate(value, values);
    const fallback = getValue(fallbackCatalog, key);
    if (typeof fallback === 'string') return interpolate(fallback, values);
    return key;
  }

  function translatePlural(catalog, key, count, values, fallbackCatalog, locale) {
    const rules = new Intl.PluralRules(locale || DEFAULT_LOCALE);
    const category = rules.select(Number(count));
    const pluralKey = `${key}.${category}`;
    const pluralFallbackKey = `${key}.other`;
    const resolved = getValue(catalog, pluralKey) || getValue(catalog, pluralFallbackKey);
    if (typeof resolved === 'string') return interpolate(resolved, { ...values, count });
    const fallback = getValue(fallbackCatalog, pluralKey) || getValue(fallbackCatalog, pluralFallbackKey);
    if (typeof fallback === 'string') return interpolate(fallback, { ...values, count });
    return translate(catalog, key, { ...values, count }, fallbackCatalog);
  }

  function formatNumber(locale, value, options) {
    return new Intl.NumberFormat(locale || DEFAULT_LOCALE, options).format(value);
  }

  function formatDate(locale, value, options) {
    const date = value instanceof Date ? value : new Date(value);
    return new Intl.DateTimeFormat(locale || DEFAULT_LOCALE, options).format(date);
  }

  function formatCurrency(locale, value, currency, options) {
    return new Intl.NumberFormat(locale || DEFAULT_LOCALE, {
      style: 'currency',
      currency: currency || 'USD',
      ...options
    }).format(value);
  }

  return {
    SUPPORTED_LOCALES,
    DEFAULT_LOCALE,
    STORAGE_KEY,
    CODES,
    normalizeLocale,
    isSupported,
    detectLocale,
    resolveLocale,
    translate,
    translatePlural,
    interpolate,
    formatNumber,
    formatDate,
    formatCurrency
  };
});
