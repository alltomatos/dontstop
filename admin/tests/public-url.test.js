const test = require("node:test");
const assert = require("node:assert/strict");
const publicUrl = require("../lib/public-url.js");

const env = (values = {}) => ({ ...values });
const headers = values => ({ get: name => values[name.toLowerCase()] ?? null });

test("normalizes base path and removes trailing slashes", () => {
  assert.equal(publicUrl.normalizeBasePath("dontstop/"), "/dontstop");
  assert.equal(publicUrl.normalizeBasePath("/"), "");
  assert.equal(publicUrl.normalizeUrl("https://host.example/"), "https://host.example");
});

test("builds an idempotent /v1 endpoint", () => {
  assert.equal(publicUrl.buildEndpointUrl("https://host.example"), "https://host.example/v1");
  assert.equal(publicUrl.buildEndpointUrl("https://host.example/", "/dontstop"), "https://host.example/dontstop/v1");
  assert.equal(publicUrl.buildEndpointUrl("https://host.example/dontstop/v1///"), "https://host.example/dontstop/v1");
  assert.equal(publicUrl.buildEndpointUrl("https://host.example/v1/v1"), "https://host.example/v1");
});

test("configured public URL wins over IP or domain access", () => {
  const config = env({ DONTSTOP_PUBLIC_BASE_URL: "https://ia.example.com/dontstop" });
  const byIp = publicUrl.resolvePublicEndpoint({ env: config, headers: headers({ host: "192.168.0.10:3000" }) });
  const byDomain = publicUrl.resolvePublicEndpoint({ env: config, headers: headers({ host: "ia.example.com" }) });
  assert.equal(byIp.url, "https://ia.example.com/dontstop/v1");
  assert.equal(byIp.source, "configured");
  assert.equal(byDomain.url, byIp.url);
});

test("trusted proxy headers supply the public origin", () => {
  const config = env({ DONTSTOP_TRUST_PROXY: "on", PORT: "3000" });
  const result = publicUrl.resolvePublicEndpoint({
    env: config,
    headers: headers({
      forwarded: 'for=192.0.2.10;proto=https;host="gateway.corp.example"',
      "x-forwarded-proto": "http",
      host: "127.0.0.1:3000"
    })
  });
  assert.equal(result.url, "https://gateway.corp.example/v1");
  assert.equal(result.source, "proxy");
  assert.equal(result.localUrl, "http://localhost:3000/v1");
});

test("untrusted forwarded headers are ignored", () => {
  const result = publicUrl.resolvePublicEndpoint({
    env: env({ PORT: "3000" }),
    headers: headers({
      "x-forwarded-proto": "https",
      "x-forwarded-host": "attacker.example",
      host: "192.168.0.10:3000"
    })
  });
  assert.equal(result.url, "http://localhost:3000/v1");
  assert.equal(result.source, "local");
});

test("rejects malformed or injected forwarded hosts", () => {
  assert.equal(publicUrl.sanitizeForwardedHost("host.example/path"), null);
  assert.equal(publicUrl.sanitizeForwardedHost("host.example@evil.example"), null);
  assert.equal(publicUrl.sanitizeForwardedHost("host.example:443"), "host.example:443");
  assert.equal(publicUrl.sanitizeForwardedProto("javascript"), null);
});
