(function (root, factory) {
  const api = factory();
  if (typeof module === "object" && module.exports) module.exports = api;
  if (root) root.DontstopModelFilters = api;
})(typeof globalThis !== "undefined" ? globalThis : this, function () {
  const CAPABILITIES = ["tools", "reasoning", "embeddings", "stream"];
  const INPUT_MODALITIES = ["text", "image", "audio", "video"];
  const OUTPUT_MODALITIES = ["text", "image", "audio", "video"];
  const EMPTY_FILTERS = Object.freeze({
    query: "",
    price: "all",
    status: "all",
    inputModalities: [],
    outputModalities: [],
    capabilities: []
  });

  function capabilitiesOf(model) {
    const caps = model && model.capabilities;
    return caps && typeof caps === "object" ? caps : {};
  }

  function priceKnown(model) {
    return capabilitiesOf(model).pricing_known === true;
  }

  function priceOf(model, key) {
    const value = Number(model && model[key]);
    return Number.isFinite(value) ? value : 0;
  }

  function isFree(model) {
    return priceKnown(model) && priceOf(model, "input_price_per_million") === 0 && priceOf(model, "output_price_per_million") === 0;
  }

  function isPaid(model) {
    return priceKnown(model) && !isFree(model);
  }

  function matchesQuery(model, query) {
    const normalized = String(query || "").trim().toLowerCase();
    if (!normalized) return true;
    const id = String(model.model_id || "").toLowerCase();
    const name = String(model.display_name || "").toLowerCase();
    return id.includes(normalized) || name.includes(normalized);
  }

  function matchesPrice(model, price) {
    switch (price) {
      case "free":
        return isFree(model);
      case "paid":
        return isPaid(model);
      case "unknown":
        return !priceKnown(model);
      default:
        return true;
    }
  }

  function matchesStatus(model, status) {
    if (status === "active") return model.active === true;
    if (status === "inactive") return model.active === false;
    return true;
  }

  function matchesCapabilities(model, capabilities) {
    if (!Array.isArray(capabilities) || capabilities.length === 0) return true;
    const caps = capabilitiesOf(model);
    return capabilities.every(capability => CAPABILITIES.includes(capability) && caps[capability] === true);
  }

  function normalizeFilters(filters) {
    const input = filters && typeof filters === "object" ? filters : {};
    const pick = (values, allowed) =>
      Array.isArray(values) ? values.filter(value => allowed.includes(value)) : [];
    return {
      query: typeof input.query === "string" ? input.query : "",
      price: ["all", "free", "paid", "unknown"].includes(input.price) ? input.price : "all",
      status: ["all", "active", "inactive"].includes(input.status) ? input.status : "all",
      inputModalities: pick(input.inputModalities, INPUT_MODALITIES),
      outputModalities: pick(input.outputModalities, OUTPUT_MODALITIES),
      capabilities: pick(input.capabilities, CAPABILITIES)
    };
  }

  function matchesModalities(model, direction, selected) {
    if (!Array.isArray(selected) || selected.length === 0) return true;
    const caps = capabilitiesOf(model);
    return selected.every(modality => {
      if (direction === "input") {
        const explicit = caps[`${modality}_input`];
        if (typeof explicit === "boolean") return explicit;
        if (modality === "image") return caps.vision === true;
        if (modality === "audio") return caps.audio === true;
        return Array.isArray(caps.input_modalities) && caps.input_modalities.includes(modality);
      }
      const explicit = caps[`${modality}_output`];
      if (typeof explicit === "boolean") return explicit;
      return Array.isArray(caps.output_modalities) && caps.output_modalities.includes(modality);
    });
  }

  function filterModels(models, filters) {
    const active = normalizeFilters(filters);
    return (Array.isArray(models) ? models : []).filter(model =>
      matchesQuery(model, active.query) &&
      matchesPrice(model, active.price) &&
      matchesStatus(model, active.status) &&
      matchesModalities(model, "input", active.inputModalities) &&
      matchesModalities(model, "output", active.outputModalities) &&
      matchesCapabilities(model, active.capabilities)
    );
  }

  function metadataKnown(model) {
    const caps = capabilitiesOf(model);
    if (typeof caps.metadata_source === "string" && caps.metadata_source !== "") return true;
    return ["input_modalities", "output_modalities", "text_input", "text_output", "tools", "reasoning", "embeddings", "stream"]
      .some(key => caps[key] !== undefined);
  }

  function toggleFilterCapability(capabilities, capability) {
    const current = Array.isArray(capabilities) ? capabilities.slice() : [];
    if (current.includes(capability)) return current.filter(item => item !== capability);
    if (!CAPABILITIES.includes(capability)) return current;
    return [...current, capability];
  }

  function selectAllFiltered(models, selected) {
    const next = new Set(selected instanceof Set ? selected : []);
    for (const model of Array.isArray(models) ? models : []) {
      if (model && model.model_id) next.add(model.model_id);
    }
    return next;
  }

  function toggleSelection(selected, modelID) {
    const next = new Set(selected instanceof Set ? selected : []);
    if (!modelID) return next;
    if (next.has(modelID)) next.delete(modelID);
    else next.add(modelID);
    return next;
  }

  function selectedModels(models, selected) {
    const ids = selected instanceof Set ? selected : new Set();
    return (Array.isArray(models) ? models : []).filter(model => ids.has(model.model_id));
  }

  function bulkPayload(models, active) {
    return { models: [...models], active: active === true };
  }

  function priceLabel(model) {
    if (!priceKnown(model)) return null;
    if (isFree(model)) return "free";
    return "paid";
  }

  return {
    CAPABILITIES,
    INPUT_MODALITIES,
    OUTPUT_MODALITIES,
    EMPTY_FILTERS,
    metadataKnown,
    capabilitiesOf,
    priceKnown,
    isFree,
    isPaid,
    normalizeFilters,
    filterModels,
    toggleFilterCapability,
    selectAllFiltered,
    toggleSelection,
    selectedModels,
    bulkPayload,
    priceLabel
  };
});
