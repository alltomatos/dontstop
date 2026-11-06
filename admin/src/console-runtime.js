(function () {
  const I18N = window.DontstopI18n;
  const PREFS = window.DontstopPreferences;
  const GUIDES = window.DontstopGuideLinks;
  const SOURCE_KEYS = window.DontstopSourceKeys || {};
  const PANEL_GUIDES = {
    dashboard: 'dashboard',
    providers: 'providers',
    endpoints: 'endpoints',
    'api-manager': 'api-manager',
    mcp: 'mcp',
    cache: 'cache',
    dlp: 'dlp',
    observability: 'observability',
    cost: 'cost',
    settings: 'settings'
  };
  const GUIDE_PAGE_KEYS = {
    dashboard: 'page.dashboard',
    providers: 'page.providers',
    endpoints: 'page.endpoints',
    'api-manager': 'page.apiManager',
    mcp: 'page.mcp',
    cache: 'page.cache',
    dlp: 'page.dlp',
    observability: 'page.observability',
    cost: 'page.cost',
    settings: 'page.settings',
    'new-api-key': 'page.newApiKey',
    onboarding: 'page.onboarding'
  };

  let locale = I18N.DEFAULT_LOCALE;
  let catalogs = {};
  let fallbackCatalog = {};
  let manifest = { defaultLocale: 'en', locales: I18N.CODES, guides: {} };
  let initialized = false;
  let onboardingForced = false;

  function t(key, values) {
    return I18N.translate(catalogs[locale] || fallbackCatalog, key, values, fallbackCatalog);
  }

  function plural(key, count, values) {
    return I18N.translatePlural(catalogs[locale] || fallbackCatalog, key, count, values, fallbackCatalog, locale);
  }

  function formatNumber(value, options) {
    return I18N.formatNumber(locale, value, options);
  }

  function safeStorageGet(key) {
    try { return localStorage.getItem(key); } catch { return null; }
  }

  function safeStorageSet(key, value) {
    try { localStorage.setItem(key, value); } catch { /* storage can be disabled */ }
  }

  async function loadJson(url) {
    const response = await fetch(url, { cache: 'no-store' });
    if (!response.ok) throw new Error(`HTTP ${response.status}`);
    return response.json();
  }

  async function loadCatalogs() {
    const entries = await Promise.all(I18N.CODES.map(async code => {
      try {
        return [code, await loadJson(`../../admin/locales/${code}.json`)];
      } catch (error) {
        console.warn(`[i18n] Unable to load locale ${code}`, error);
        return [code, null];
      }
    }));
    catalogs = Object.fromEntries(entries.filter(([, catalog]) => catalog));
    fallbackCatalog = catalogs.en || {};
  }

  async function loadManifest() {
    try {
      manifest = await loadJson('../../docs/user-guide/manifest.json');
    } catch (error) {
      console.warn('[guide] Unable to load manifest; falling back to README', error);
    }
  }

  function normalizeText(value) {
    return String(value || '').replace(/\s+/g, ' ').trim();
  }

  function translatedText(source) {
    const normalized = normalizeText(source);
    const key = SOURCE_KEYS[normalized];
    if (!key) return null;
    return t(key);
  }

  function patternTranslation(source) {
    const replicas = source.match(/^(\d+)\s+réplicas?$/);
    if (replicas) return t('observability.replicas', { count: replicas[1] });
    const ratio = source.match(/^(\d+)\/(\d+)\s+conectadas?$/);
    if (ratio) return t('providers.connectedRatio', { connected: ratio[1], total: ratio[2] });
    const relative = source.match(/^há\s+(\d+)\s+(min|h|dias?)$/i);
    if (relative) {
      const amount = Number(relative[1]);
      const rawUnit = relative[2].toLowerCase();
      const unit = rawUnit === 'min' ? 'minute' : rawUnit === 'h' ? 'hour' : 'day';
      return new Intl.RelativeTimeFormat(locale, { numeric: 'auto' }).format(-amount, unit);
    }
    return null;
  }

  function applyTextNode(node) {
    const parent = node.parentElement;
    if (!parent) return;
    if (parent.closest('script, style, [data-i18n], [data-i18n-skip]')) return;
    const raw = node.nodeValue;
    if (!raw || !raw.trim()) return;
    if (!node.__i18nSource) node.__i18nSource = raw;
    const leading = node.__i18nSource.match(/^\s*/)[0];
    const trailing = node.__i18nSource.match(/\s*$/)[0];
    const patterned = patternTranslation(normalizeText(node.__i18nSource));
    if (patterned) {
      node.nodeValue = `${leading}${patterned}${trailing}`;
      return;
    }
    const translated = translatedText(node.__i18nSource);
    if (!translated) return;
    node.nodeValue = `${leading}${translated}${trailing}`;
  }

  function applyAttribute(element, attribute, keyAttribute) {
    const originalKey = `__i18n_${keyAttribute}`;
    if (!element[originalKey]) element[originalKey] = element.getAttribute(attribute) || '';
    const translated = translatedText(element[originalKey]);
    if (translated) element.setAttribute(attribute, translated);
  }

  function translateProviderCounts() {
    document.querySelectorAll('.provider-card .foot > .meta').forEach(element => {
      if (!element.__providerCountSource) element.__providerCountSource = element.textContent.trim();
      const match = element.__providerCountSource.match(/^(\d+)\s+conectadas?$/);
      if (!match) return;
      element.textContent = I18N.translatePlural(catalogs[locale] || fallbackCatalog, 'providers.connected', Number(match[1]), {}, fallbackCatalog, locale);
    });
  }

  function translateEndpointCounts() {
    document.querySelectorAll('.endpoint-tile .count').forEach(element => {
      if (!element.__endpointCountSource) element.__endpointCountSource = element.textContent.trim();
      const match = element.__endpointCountSource.match(/^([\d.,]+)\s+modelos$/);
      if (!match) return;
      element.textContent = t('endpoints.modelsCount', { count: match[1] });
    });
  }

  function applyTranslations() {
    document.documentElement.lang = locale;
    document.documentElement.dir = (I18N.SUPPORTED_LOCALES.find(item => item.code === locale) || {}).dir || 'ltr';
    document.querySelectorAll('[data-i18n]').forEach(element => {
      element.textContent = t(element.dataset.i18n);
    });
    document.querySelectorAll('[data-i18n-aria-label]').forEach(element => {
      element.setAttribute('aria-label', t(element.dataset.i18nAriaLabel));
    });
    document.querySelectorAll('[data-i18n-title]').forEach(element => {
      element.setAttribute('title', t(element.dataset.i18nTitle));
    });
    document.querySelectorAll('[data-i18n-placeholder]').forEach(element => {
      element.setAttribute('placeholder', t(element.dataset.i18nPlaceholder));
    });
    const walker = document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT);
    const nodes = [];
    while (walker.nextNode()) nodes.push(walker.currentNode);
    nodes.forEach(applyTextNode);
    document.querySelectorAll('[aria-label]').forEach(element => applyAttribute(element, 'aria-label', 'ariaLabel'));
    document.querySelectorAll('[title]').forEach(element => applyAttribute(element, 'title', 'title'));
    document.querySelectorAll('[placeholder]').forEach(element => applyAttribute(element, 'placeholder', 'placeholder'));
    document.querySelectorAll('img[alt]').forEach(element => applyAttribute(element, 'alt', 'alt'));
    translateProviderCounts();
    translateEndpointCounts();
    updateLanguageControls();
    updateAllGuideLinks();
    if (typeof window.onLocaleChanged === 'function') window.onLocaleChanged(locale);
  }

  function fillLanguageGrid(container, groupName, selected, onChange) {
    if (!container) return;
    const legend = container.querySelector('legend');
    container.querySelectorAll('.language-card').forEach(card => card.remove());
    I18N.SUPPORTED_LOCALES.forEach(item => {
      const label = document.createElement('label');
      label.className = 'language-card';
      const input = document.createElement('input');
      input.type = 'radio';
      input.name = groupName;
      input.value = item.code;
      input.checked = item.code === selected;
      input.addEventListener('change', () => onChange(item.code));
      const text = document.createElement('span');
      text.innerHTML = `<strong>${item.name}</strong><span>${item.code}</span>`;
      label.append(input, text);
      container.append(label);
    });
    if (legend) container.prepend(legend);
  }

  function updateLanguageControls() {
    const select = document.getElementById('locale-select');
    if (select) select.value = locale;
    document.querySelectorAll('#settings-language-grid input, #onboarding-language-grid input').forEach(input => {
      input.checked = input.value === locale;
    });
  }

  function currentGuideSlug() {
    if (!document.getElementById('onboarding-modal')?.classList.contains('hidden')) return 'onboarding';
    if (!document.getElementById('new-key-modal')?.classList.contains('hidden')) return 'new-api-key';
    const panel = [...document.querySelectorAll('.panel')].find(item => !item.classList.contains('hidden'));
    return panel ? (PANEL_GUIDES[panel.id] || panel.id) : 'dashboard';
  }

  function guideLabel(slug) {
    const pageKey = GUIDE_PAGE_KEYS[slug] || 'page.dashboard';
    return t('common.helpFor', { page: t(pageKey) });
  }

  function updateAllGuideLinks() {
    document.querySelectorAll('[data-guide-slug]').forEach(link => {
      const slug = link.dataset.guideSlug;
      link.href = GUIDES.resolveGuideUrl(locale, slug, manifest);
      link.target = '_blank';
      link.rel = 'noopener noreferrer';
      link.setAttribute('aria-label', guideLabel(slug));
      link.setAttribute('title', guideLabel(slug));
    });
  }

  function updateContextHelp() {
    const link = document.getElementById('context-help');
    if (!link) return;
    const slug = currentGuideSlug();
    link.dataset.guideSlug = slug;
    link.href = GUIDES.resolveGuideUrl(locale, slug, manifest);
    link.setAttribute('aria-label', guideLabel(slug));
    link.setAttribute('title', guideLabel(slug));
    link.textContent = t('common.help');
  }

  function announce(message) {
    const live = document.getElementById('locale-live');
    if (!live) return;
    live.textContent = '';
    window.setTimeout(() => { live.textContent = message; }, 30);
  }

  function setLocale(nextLocale, options) {
    const settings = { persist: true, announce: true, ...(options || {}) };
    const resolved = I18N.resolveLocale(nextLocale, I18N.CODES);
    locale = resolved;
    if (settings.persist) PREFS.saveLocale(window.localStorage, resolved);
    applyTranslations();
    updateContextHelp();
    if (settings.announce) {
      const name = (I18N.SUPPORTED_LOCALES.find(item => item.code === resolved) || {}).name || resolved;
      announce(t('common.language') + ': ' + name);
    }
    return resolved;
  }

  function buildSelect() {
    const select = document.getElementById('locale-select');
    if (!select) return;
    select.innerHTML = '';
    I18N.SUPPORTED_LOCALES.forEach(item => {
      const option = document.createElement('option');
      option.value = item.code;
      option.textContent = item.name;
      select.append(option);
    });
    select.addEventListener('change', event => setLocale(event.target.value, { persist: true }));
  }

  function showOnboarding() {
    const modal = document.getElementById('onboarding-modal');
    if (!modal) return;
    const params = new URLSearchParams(location.search);
    const detected = I18N.detectLocale(navigator.languages || [navigator.language]);
    const stored = safeStorageGet(I18N.STORAGE_KEY);
    const forcedLang = I18N.normalizeLocale(params.get('lang'));
    const selected = forcedLang || (stored ? I18N.resolveLocale(stored, I18N.CODES) : detected);
    fillLanguageGrid(document.getElementById('onboarding-language-grid'), 'onboarding-locale', selected, code => {
      document.querySelectorAll('#onboarding-language-grid input').forEach(input => { input.checked = input.value === code; });
    });
    const detectedLabel = I18N.SUPPORTED_LOCALES.find(item => item.code === selected);
    document.getElementById('onboarding-detected').textContent = t('onboarding.detected', { locale: detectedLabel ? detectedLabel.name : selected });
    modal.classList.remove('hidden');
    updateAllGuideLinks();
    const focusTarget = modal.querySelector('input:checked') || document.getElementById('onboarding-confirm');
    window.setTimeout(() => focusTarget?.focus(), 30);
  }

  function hideOnboarding() {
    const modal = document.getElementById('onboarding-modal');
    if (!modal) return;
    modal.classList.add('hidden');
    document.getElementById('context-help')?.focus();
  }

  function initOnboarding(decision) {
    const modal = document.getElementById('onboarding-modal');
    if (!modal) return;
    onboardingForced = new URLSearchParams(location.search).get('onboarding') === '1';
    document.getElementById('onboarding-confirm')?.addEventListener('click', () => {
      const selected = document.querySelector('#onboarding-language-grid input:checked');
      setLocale(selected ? selected.value : locale, { persist: true, announce: true });
      hideOnboarding();
    });
    modal.addEventListener('keydown', event => {
      if (event.key !== 'Tab') return;
      const focusable = [...modal.querySelectorAll('button, input, [href]')].filter(el => !el.disabled && el.offsetParent !== null);
      if (!focusable.length) return;
      const first = focusable[0];
      const last = focusable[focusable.length - 1];
      if (event.shiftKey && document.activeElement === first) { last.focus(); event.preventDefault(); }
      else if (!event.shiftKey && document.activeElement === last) { first.focus(); event.preventDefault(); }
    });
    modal.addEventListener('keydown', event => {
      if (event.key === 'Escape' && !onboardingForced) event.preventDefault();
    }, true);
    if (decision && decision.showOnboarding) showOnboarding();
  }

  function initSettings() {
    fillLanguageGrid(document.getElementById('settings-language-grid'), 'settings-locale', locale, code => {
      setLocale(code, { persist: true, announce: true });
    });
  }

  async function init() {
    if (initialized) return;
    initialized = true;
    await Promise.all([loadCatalogs(), loadManifest()]);
    buildSelect();
    const params = new URLSearchParams(location.search);
    const decision = PREFS.decide({
      stored: PREFS.readLocale(window.localStorage),
      queryLang: params.get('lang'),
      onboarding: params.get('onboarding'),
      languages: navigator.languages || [navigator.language]
    });
    setLocale(decision.locale, { persist: false, announce: false });
    initSettings();
    initOnboarding(decision);
    applyTranslations();
    updateContextHelp();
    updateAllGuideLinks();
    document.addEventListener('keydown', event => {
      if (event.key === 'Tab') updateContextHelp();
    });
  }

  window.DontstopConsoleI18n = {
    t,
    plural,
    setLocale,
    applyTranslations,
    updateContextHelp,
    updateAllGuideLinks,
    getLocale: () => locale,
    formatNumber
  };
  window.tr = t;
  window.trPlural = plural;

  document.addEventListener('DOMContentLoaded', init);
  if (document.readyState !== 'loading') init();
})();
