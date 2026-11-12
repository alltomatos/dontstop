export const CONFIGURE_PREFIX: string;
export const PROVIDERS_PATH: string;
export const CATALOG_PATH: string;

export interface CatalogLikeEntry {
  slug: string;
  display_name?: string;
  capabilities?: Record<string, boolean>;
}

export interface ConfiguredLikeProvider {
  name?: string;
  slug?: string;
  catalogSlug?: string;
  catalog_slug?: string;
}

export function configurePath(slug: string): string;
export function provideGuideAnchor(slug: string): string;
export function configuredKeys(providers: ReadonlyArray<ConfiguredLikeProvider>): Set<string>;
export function filterAvailableCatalog<T extends CatalogLikeEntry>(
  entries: ReadonlyArray<T>,
  configured: Set<string> | ReadonlyArray<ConfiguredLikeProvider>
): T[];
export function capabilitiesOf(entry: CatalogLikeEntry): string[];
export function normalizeKey(value: unknown): string;
