(function (root, factory) {
  const api = factory();
  if (typeof module === "object" && module.exports) module.exports = api;
  if (root) root.DontstopCatalog = api;
})(typeof globalThis !== "undefined" ? globalThis : this, function () {
  const CONFIGURE_PREFIX = "/providers/configure/";
  const PROVIDERS_PATH = "/providers";
  const CATALOG_PATH = "/providers/catalog";

  function normalizeKey(value) {
    return String(value || "").trim().toLowerCase();
  }

  function configurePath(slug) {
    return CONFIGURE_PREFIX + encodeURIComponent(String(slug || "").trim());
  }

  function provideGuideAnchor(slug) {
    return normalizeKey(slug).replace(/[^a-z0-9-]+/g, "-").replace(/^-+|-+$/g, "");
  }

  function configuredKeys(providers) {
    const keys = new Set();
    for (const provider of Array.isArray(providers) ? providers : []) {
      if (!provider) continue;
      if (provider.catalogSlug) keys.add(normalizeKey(provider.catalogSlug));
      if (provider.catalog_slug) keys.add(normalizeKey(provider.catalog_slug));
      if (provider.slug) keys.add(normalizeKey(provider.slug));
      if (provider.name) keys.add(normalizeKey(provider.name));
    }
    return keys;
  }

  function filterAvailableCatalog(entries, configured) {
    const keys = configured instanceof Set ? configured : configuredKeys(configured);
    return (Array.isArray(entries) ? entries : []).filter(entry => {
      if (!entry || !entry.slug) return false;
      if (keys.has(normalizeKey(entry.slug))) return false;
      if (entry.display_name && keys.has(normalizeKey(entry.display_name))) return false;
      return true;
    });
  }

  function capabilitiesOf(entry) {
    const caps = (entry && entry.capabilities) || {};
    const labels = [];
    if (caps.tools) labels.push("Tools");
    if (caps.vision) labels.push("Vision");
    if (caps.reasoning) labels.push("Reasoning");
    if (caps.stream) labels.push("Stream");
    return labels;
  }

  return {
    CONFIGURE_PREFIX,
    PROVIDERS_PATH,
    CATALOG_PATH,
    configurePath,
    provideGuideAnchor,
    configuredKeys,
    filterAvailableCatalog,
    capabilitiesOf,
    normalizeKey
  };
});
