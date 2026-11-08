function getHeader(headers, name) {
  if (!headers) return null;
  if (typeof headers.get === "function") return headers.get(name);
  const value = headers[name] ?? headers[name.toLowerCase()];
  return Array.isArray(value) ? value[0] ?? null : value ?? null;
}

function firstHeaderValue(value) {
  if (typeof value !== "string") return null;
  return value.split(",")[0].trim() || null;
}

function normalizeUrl(value) {
  const trimmed = String(value ?? "").trim();
  if (!trimmed) return null;
  try {
    return new URL(trimmed).toString().replace(/\/+$/, "");
  } catch {
    return null;
  }
}

function normalizeBasePath(value) {
  const trimmed = String(value ?? "").trim();
  if (!trimmed || trimmed === "/") return "";
  const withSlash = trimmed.startsWith("/") ? trimmed : `/${trimmed}`;
  return withSlash.replace(/\/+$/, "");
}

function sanitizeForwardedProto(value) {
  const normalized = String(value ?? "").trim().toLowerCase();
  return normalized === "http" || normalized === "https" ? normalized : null;
}

function sanitizeForwardedHost(value) {
  const trimmed = String(value ?? "").trim();
  if (!trimmed || /[/\\\s\x00-\x1f\x7f]/.test(trimmed)) return null;
  try {
    const parsed = new URL(`http://${trimmed}`);
    if (parsed.username || parsed.password || parsed.pathname !== "/" || parsed.search || parsed.hash) return null;
    return parsed.host.toLowerCase();
  } catch {
    return null;
  }
}

function parseForwarded(value) {
  const first = firstHeaderValue(value);
  if (!first) return { proto: null, host: null };
  let proto = null;
  let host = null;
  for (const segment of first.split(";")) {
    const [rawKey, ...rawValue] = segment.split("=");
    const key = rawKey?.trim().toLowerCase();
    const part = rawValue.join("=").trim().replace(/^"|"$/g, "");
    if (!key || !part) continue;
    if (key === "proto") proto = part;
    if (key === "host") host = part;
  }
  return { proto, host };
}

function normalizeEndpointBase(value) {
  const trimmed = String(value ?? "").trim().replace(/\/+$/, "");
  const withoutV1 = trimmed.replace(/(?:\/v1)+$/i, "");
  return `${withoutV1}/v1`;
}

function buildEndpointUrl(base, basePath) {
  const normalized = normalizeUrl(base);
  if (!normalized) return null;
  const parsed = new URL(normalized);
  const pathFromBase = normalizeBasePath(parsed.pathname);
  const extraPath = pathFromBase ? "" : normalizeBasePath(basePath);
  return normalizeEndpointBase(`${parsed.origin}${pathFromBase}${extraPath}`);
}

function resolvePublicEndpoint(options = {}) {
  const env = options.env ?? process.env;
  const headers = options.headers;
  const configured = normalizeUrl(env.DONTSTOP_PUBLIC_BASE_URL);
  const localBase = normalizeUrl(env.DONTSTOP_LOCAL_BASE_URL) ?? `http://localhost:${env.PORT || "8080"}`;
  const basePath = normalizeBasePath(env.DONTSTOP_BASE_PATH);
  let base = configured;
  let source = configured ? "configured" : "local";

  if (!base) {
    const trustProxy = ["1", "true", "yes", "on"].includes(String(env.DONTSTOP_TRUST_PROXY ?? "").toLowerCase());
    if (trustProxy) {
      const forwarded = parseForwarded(getHeader(headers, "forwarded"));
      const proto = sanitizeForwardedProto(
        forwarded.proto ?? firstHeaderValue(getHeader(headers, "x-forwarded-proto"))
      );
      const host = sanitizeForwardedHost(
        forwarded.host ??
          firstHeaderValue(getHeader(headers, "x-forwarded-host")) ??
          firstHeaderValue(getHeader(headers, "host"))
      );
      if (proto && host) {
        base = `${proto}://${host}`;
        source = "proxy";
      }
    }
  }
  if (!base) base = localBase;

  return {
    url: buildEndpointUrl(base, basePath),
    localUrl: buildEndpointUrl(localBase, basePath),
    base,
    basePath,
    source
  };
}

module.exports = {
  buildEndpointUrl,
  firstHeaderValue,
  getHeader,
  normalizeBasePath,
  normalizeEndpointBase,
  normalizeUrl,
  parseForwarded,
  resolvePublicEndpoint,
  sanitizeForwardedHost,
  sanitizeForwardedProto
};
