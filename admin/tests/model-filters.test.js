const test = require("node:test");
const assert = require("node:assert/strict");
const filters = require("../src/model-filters.js");

const models = [
  {
    model_id: "audio-free",
    display_name: "Audio Free",
    active: true,
    input_price_per_million: 0,
    output_price_per_million: 0,
    capabilities: {
      pricing_known: true,
      metadata_source: "provider:/models",
      input_modalities: ["text", "audio"],
      output_modalities: ["text"],
      text_input: true,
      text_output: true,
      audio_input: true,
      audio_output: false,
      tools: true,
      stream: true
    }
  },
  {
    model_id: "vision-paid",
    display_name: "Vision Paid",
    active: false,
    input_price_per_million: 2.5,
    output_price_per_million: 10,
    capabilities: {
      pricing_known: true,
      metadata_source: "provider:/models",
      input_modalities: ["text", "image"],
      output_modalities: ["text", "image"],
      text_input: true,
      text_output: true,
      image_input: true,
      image_output: true,
      tools: false,
      reasoning: true
    }
  },
  {
    model_id: "no-metadata",
    display_name: "Unknown Model",
    active: true,
    input_price_per_million: 0,
    output_price_per_million: 0,
    capabilities: {}
  }
];

test("free filter only matches models with known zero prices", () => {
  const result = filters.filterModels(models, { price: "free" });
  assert.deepEqual(result.map(model => model.model_id), ["audio-free"]);
});

test("paid and unknown price filters are explicit", () => {
  assert.deepEqual(filters.filterModels(models, { price: "paid" }).map(model => model.model_id), ["vision-paid"]);
  assert.deepEqual(filters.filterModels(models, { price: "unknown" }).map(model => model.model_id), ["no-metadata"]);
});

test("modality and capability filters combine with AND", () => {
  assert.deepEqual(filters.filterModels(models, { inputModalities: ["audio"] }).map(model => model.model_id), ["audio-free"]);
  assert.deepEqual(filters.filterModels(models, { inputModalities: ["image"] }).map(model => model.model_id), ["vision-paid"]);
  assert.deepEqual(filters.filterModels(models, { outputModalities: ["image"] }).map(model => model.model_id), ["vision-paid"]);
  assert.deepEqual(filters.filterModels(models, { capabilities: ["reasoning"] }).map(model => model.model_id), ["vision-paid"]);
  assert.deepEqual(filters.filterModels(models, { capabilities: ["tools", "stream"] }).map(model => model.model_id), ["audio-free"]);
  assert.deepEqual(filters.filterModels(models, { inputModalities: ["audio", "image"] }), []);
  assert.deepEqual(filters.filterModels(models, { inputModalities: ["text"], capabilities: ["reasoning"] }).map(model => model.model_id), ["vision-paid"]);
});

test("models without metadata are not filtered out by default", () => {
  assert.deepEqual(filters.filterModels(models, {}).map(model => model.model_id), ["audio-free", "vision-paid", "no-metadata"]);
  assert.equal(filters.metadataKnown(models[2]), false);
  assert.deepEqual(filters.filterModels(models, { inputModalities: ["text"] }).map(model => model.model_id), ["audio-free", "vision-paid"]);
});

test("status and query filters combine with price and capabilities", () => {
  assert.deepEqual(filters.filterModels(models, { status: "inactive" }).map(model => model.model_id), ["vision-paid"]);
  assert.deepEqual(filters.filterModels(models, { query: "unknown" }).map(model => model.model_id), ["no-metadata"]);
  assert.deepEqual(
    filters.filterModels(models, { price: "free", status: "active", capabilities: ["tools"], query: "audio" })
      .map(model => model.model_id),
    ["audio-free"]
  );
});

test("select all filtered respects the current filter", () => {
  const filtered = filters.filterModels(models, { price: "free" });
  const selected = filters.selectAllFiltered(filtered, new Set());
  assert.deepEqual([...selected], ["audio-free"]);
  assert.deepEqual(filters.selectedModels(models, selected).map(model => model.model_id), ["audio-free"]);
});

test("individual selection toggles without duplicates", () => {
  let selected = filters.toggleSelection(new Set(), "vision-paid");
  selected = filters.toggleSelection(selected, "no-metadata");
  assert.deepEqual([...selected].sort(), ["no-metadata", "vision-paid"]);
  selected = filters.toggleSelection(selected, "vision-paid");
  assert.deepEqual([...selected], ["no-metadata"]);
});

test("bulk payload carries the selection and target state", () => {
  assert.deepEqual(filters.bulkPayload(["a", "b"], false), { models: ["a", "b"], active: false });
  assert.equal(filters.bulkPayload(["a"], true).active, true);
});

test("price labels never guess when metadata is absent", () => {
  assert.equal(filters.priceLabel(models[0]), "free");
  assert.equal(filters.priceLabel(models[1]), "paid");
  assert.equal(filters.priceLabel(models[2]), null);
  assert.equal(filters.priceKnown(models[2]), false);
});
