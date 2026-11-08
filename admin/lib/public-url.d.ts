type HeaderBag =
  | { get: (name: string) => string | null }
  | Record<string, string | string[] | null | undefined>
  | null
  | undefined;

type EndpointResolution = {
  url: string | null;
  localUrl: string | null;
  base: string;
  basePath: string;
  source: "configured" | "proxy" | "local";
};

declare const publicUrl: {
  buildEndpointUrl(base: string, basePath?: string | null): string | null;
  firstHeaderValue(value: unknown): string | null;
  getHeader(headers: HeaderBag, name: string): string | null;
  normalizeBasePath(value?: string | null): string;
  normalizeEndpointBase(value: string): string;
  normalizeUrl(value?: string | null): string | null;
  parseForwarded(value: unknown): { proto: string | null; host: string | null };
  resolvePublicEndpoint(options?: {
    env?: Record<string, string | undefined>;
    headers?: HeaderBag;
  }): EndpointResolution;
  sanitizeForwardedHost(value: unknown): string | null;
  sanitizeForwardedProto(value: unknown): "http" | "https" | null;
};

export = publicUrl;
