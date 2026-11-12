const test = require("node:test");
const assert = require("node:assert/strict");
const selection = require("../src/account-selection.js");

const accounts = [
  { id: "a", name: "A", active: true, health: "active", modelsUsable: 10 },
  { id: "b", name: "B", active: false, health: "cooldown", modelsUsable: 0 },
  { id: "c", name: "C", active: true, health: "quarantined", modelsUsable: 0 },
  { id: "d", name: "D", active: true, health: "active", modelsUsable: 4 }
];

test("only active accounts with usable models are eligible", () => {
  assert.deepEqual(selection.eligibleAccounts(accounts).map((account) => account.id), ["a", "d"]);
});

test("saved account wins when still eligible", () => {
  assert.equal(selection.pickTestAccount(accounts, "d").id, "d");
});

test("falls back to the first eligible account", () => {
  assert.equal(selection.pickTestAccount(accounts, "b").id, "a");
  assert.equal(selection.pickTestAccount(accounts).id, "a");
});

test("returns undefined when no account can serve models", () => {
  assert.equal(selection.pickTestAccount([{ id: "x", name: "X", active: true, health: "active", modelsUsable: 0 }]), undefined);
});

test("remembers the last account per provider", () => {
  const store = selection.rememberTestAccount({}, "p1", "a");
  assert.deepEqual(selection.rememberTestAccount(store, "p2", "d"), { p1: "a", p2: "d" });
});
