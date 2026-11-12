const test = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");

const catalog = require("../src/catalog.js");

const root = path.join(__dirname, "..", "..");
const locales = ["en", "pt-PT", "pt-BR", "es", "fr", "de"];
const supportedSlugs = [
  "alibaba", "anthropic", "cerebras", "cohere", "groq", "mistral", "nvidia",
  "openai", "openrouter", "perplexity", "together", "venice", "vercel-ai-gateway", "xai"
];

test("configure path always targets the configure route", () => {
  assert.equal(catalog.configurePath("openrouter"), "/providers/configure/openrouter");
  assert.equal(catalog.configurePath("vercel ai gateway"), "/providers/configure/vercel%20ai%20gateway");
  assert.equal(catalog.configurePath(""), "/providers/configure/");
});

test("configured providers leave the catalog", () => {
  const entries = [
    { slug: "openrouter", display_name: "OpenRouter" },
    { slug: "groq", display_name: "Groq" },
    { slug: "mistral", display_name: "Mistral" }
  ];
  const configured = [
    { name: "OpenRouter", catalogSlug: "openrouter" },
    { name: "Groq" }
  ];
  const available = catalog.filterAvailableCatalog(entries, configured);
  assert.deepEqual(available.map(entry => entry.slug), ["mistral"]);
});

test("configured detection accepts API and view-model shapes", () => {
  const keys = catalog.configuredKeys([
    { name: "OpenRouter", catalog_slug: "openrouter" },
    { name: "Groq", slug: "groq" }
  ]);
  assert.ok(keys.has("openrouter"));
  assert.ok(keys.has("groq"));
  assert.ok(keys.has("openrouter") && catalog.filterAvailableCatalog(
    [{ slug: "openrouter" }], keys
  ).length === 0);
});

test("catalog capabilities and anchors are stable", () => {
  assert.deepEqual(
    catalog.capabilitiesOf({ slug: "x", capabilities: { tools: true, vision: true, reasoning: true, stream: true } }),
    ["Tools", "Vision", "Reasoning", "Stream"]
  );
  for (const slug of supportedSlugs) {
    assert.equal(catalog.provideGuideAnchor(slug), slug);
  }
});

test("catalog page is driven by the gateway catalog and excludes configured providers", () => {
  const page = fs.readFileSync(path.join(root, "admin", "app", "(console)", "providers", "catalog", "page.tsx"), "utf8");
  assert.match(page, /getProviderCatalogResult/);
  assert.match(page, /getConfiguredProviders/);
  assert.match(page, /availableCatalogProviders/);

  const view = fs.readFileSync(path.join(root, "admin", "app", "(console)", "providers", "catalog", "catalog-view.tsx"), "utf8");
  assert.match(view, /configurePath\(provider\.id\)/);
  assert.doesNotMatch(view, /CATALOG_PROVIDERS/);
});

test("add provider and configure routes exist in the console", () => {
  const providersView = fs.readFileSync(path.join(root, "admin", "app", "(console)", "providers", "providers-view.tsx"), "utf8");
  assert.match(providersView, /href="\/providers\/catalog"/);

  const configurePage = path.join(root, "admin", "app", "(console)", "providers", "configure", "[slug]", "page.tsx");
  const configureForm = path.join(root, "admin", "app", "(console)", "providers", "configure", "[slug]", "configure-form.tsx");
  const proxy = path.join(root, "admin", "app", "api", "admin", "provider-catalog", "[slug]", "connect", "route.ts");
  assert.ok(fs.existsSync(configurePage), "configure page is missing");
  assert.ok(fs.existsSync(configureForm), "configure form is missing");
  assert.ok(fs.existsSync(proxy), "connect proxy route is missing");

  const form = fs.readFileSync(configureForm, "utf8");
  assert.match(form, /provider-catalog\/\$\{encodeURIComponent\(entry\.slug\)\}\/connect/);
  assert.match(form, /router\.refresh\(\)/);
  assert.match(form, /setError\(/);
});

test("provider setup guide documents every supported provider with anchors", () => {
  for (const locale of locales) {
    const file = path.join(root, "docs", "user-guide", locale, "provider-setup.md");
    assert.ok(fs.existsSync(file), `${locale}/provider-setup.md is missing`);
    const body = fs.readFileSync(file, "utf8");
    for (const slug of supportedSlugs) {
      assert.ok(body.includes(`<a id="${slug}"></a>`), `${locale} guide has no anchor for ${slug}`);
    }
  }
});
