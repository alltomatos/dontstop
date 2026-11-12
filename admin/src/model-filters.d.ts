export const CAPABILITIES: string[];
export const INPUT_MODALITIES: string[];
export const OUTPUT_MODALITIES: string[];
export const EMPTY_FILTERS: {
  query: string;
  price: "all" | "free" | "paid" | "unknown";
  status: "all" | "active" | "inactive";
  inputModalities: string[];
  outputModalities: string[];
  capabilities: string[];
};

export interface FilterableModel {
  model_id: string;
  display_name?: string;
  active: boolean;
  capabilities?: Record<string, unknown> | null;
  input_price_per_million?: number;
  output_price_per_million?: number;
}

export interface ModelFilters {
  query?: string;
  price?: "all" | "free" | "paid" | "unknown";
  status?: "all" | "active" | "inactive";
  inputModalities?: string[];
  outputModalities?: string[];
  capabilities?: string[];
}

export function capabilitiesOf(model: FilterableModel): Record<string, unknown>;
export function metadataKnown(model: FilterableModel): boolean;
export function priceKnown(model: FilterableModel): boolean;
export function isFree(model: FilterableModel): boolean;
export function isPaid(model: FilterableModel): boolean;
export function normalizeFilters(filters?: ModelFilters): Required<ModelFilters>;
export function filterModels<T extends FilterableModel>(models: ReadonlyArray<T>, filters?: ModelFilters): T[];
export function toggleFilterCapability(capabilities: ReadonlyArray<string>, capability: string): string[];
export function selectAllFiltered(models: ReadonlyArray<FilterableModel>, selected: Set<string>): Set<string>;
export function toggleSelection(selected: Set<string>, modelID: string): Set<string>;
export function selectedModels<T extends FilterableModel>(models: ReadonlyArray<T>, selected: Set<string>): T[];
export function bulkPayload(models: ReadonlyArray<string>, active: boolean): { models: string[]; active: boolean };
export function priceLabel(model: FilterableModel): "free" | "paid" | null;
