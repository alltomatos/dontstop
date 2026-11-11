const test = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");

const { renderMarkdown, rewriteHref } = require("../../scripts/build-guide-html.js");

const root = path.join(__dirname, "..", "..");
const guideDir = path.join(root, "docs", "user-guide");
const publicDir = path.join(root, "admin", "public", "guide");
const manifest = JSON.parse(fs.readFileSync(path.join(guideDir, "manifest.json"), "utf8"));
const locales = ["en", "pt-PT", "pt-BR", "es", "fr", "de"];
const stepImages = ["provider-catalog", "provider-configure", "provider-models"];

test("markdown renderer covers the guide blocks", () => {
  const html = renderMarkdown([
    "# Title",
    "",
    "Intro with **bold** and `code`.",
    "",
    "- one",
    "- two",
    "",
    "1. first",
    "2. second",
    "",
    "| A | B |",
    "|---|---|",
    "| 1 | 2 |",
    "",
    "![Alt](../assets/en/x.png)",
    "",
    "<a id=\"openrouter\"></a>",
    "",
    "See [Providers](providers.md#top)."
  ].join("\n"));
  assert.match(html, /<h1>Title<\/h1>/);
  assert.match(html, /<strong>bold<\/strong>/);
  assert.match(html, /<code>code<\/code>/);
  assert.match(html, /<ul><li>one<\/li><li>two<\/li><\/ul>/);
  assert.match(html, /<ol><li>first<\/li><li>second<\/li><\/ol>/);
  assert.match(html, /<table><thead><tr><th>A<\/th><th>B<\/th><\/tr><\/thead><tbody>/);
  assert.match(html, /<img src="\.\.\/assets\/en\/x\.png" alt="Alt"/);
  assert.match(html, /<a id="openrouter"><\/a>/);
  assert.match(html, /href="providers\.html#top"/);
});

test("guide links are rewritten to the generated html", () => {
  assert.equal(rewriteHref("providers.md"), "providers.html");
  assert.equal(rewriteHref("providers.md#models"), "providers.html#models");
  assert.equal(rewriteHref("https://example.test/a.md"), "https://example.test/a.md");
  assert.equal(rewriteHref("#anchor"), "#anchor");
});

test("every guide is rendered to static html for the console", () => {
  assert.ok(fs.existsSync(path.join(publicDir, "index.html")), "guide index is missing");
  for (const locale of locales) {
    for (const slug of Object.keys(manifest.guides)) {
      const file = path.join(publicDir, locale, `${slug}.html`);
      assert.ok(fs.existsSync(file), `${locale}/${slug}.html is missing`);
      const body = fs.readFileSync(file, "utf8");
      assert.match(body, /<h1>/, `${locale}/${slug}.html has no title`);
      assert.ok(body.includes(`lang="${locale}"`), `${locale}/${slug}.html has no lang attribute`);
    }
  }
});

test("provider setup guide embeds the step screenshots", () => {
  for (const locale of locales) {
    const body = fs.readFileSync(path.join(publicDir, locale, "provider-setup.html"), "utf8");
    for (const image of stepImages) {
      assert.ok(body.includes(`../assets/${locale}/${image}.png`), `${locale} guide misses ${image}`);
      const file = path.join(guideDir, "assets", locale, `${image}.png`);
      assert.ok(fs.existsSync(file), `missing screenshot ${locale}/${image}.png`);
      assert.ok(fs.statSync(file).size > 5000, `screenshot ${locale}/${image}.png looks empty`);
    }
    assert.ok(body.includes('<a id="openrouter"></a>'), `${locale} guide misses provider anchors`);
  }
});

test("console links point to the local guide bundle", () => {
  const source = fs.readFileSync(path.join(root, "admin", "lib", "i18n.ts"), "utf8");
  assert.ok(source.includes("/guide/${locale}/${slug}"), "guideUrl must use the guide route");
});
