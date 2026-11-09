import { test as base, expect } from "@playwright/test";

export const BASE_URL = process.env.E2E_BASE_URL ?? "http://localhost:3000";

export const test = base;

test.beforeEach(async ({ request }) => {
  const response = await request
    .get(`${BASE_URL}/login`, { failOnStatusCode: false, maxRedirects: 0 })
    .catch(() => null);
  test.skip(response === null, `console indisponível em ${BASE_URL}`);
});

export { expect };
