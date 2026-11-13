import { test, expect } from "@playwright/test";

// Security gate for the proxy isolation feature (ADR-024).
// Requires a running console/gateway: E2E_SECURITY=true npx playwright test --grep @security
const ENABLED = process.env.E2E_SECURITY === "true";

test.describe("Security Gate — proxy isolation @security", () => {
  test.skip(!ENABLED, "set E2E_SECURITY=true with a running console");
  test.use({ storageState: { cookies: [], origins: [] } });

  test("unauthenticated /proxies redirects to login @security", async ({ page }) => {
    await page.goto("/proxies");
    await expect(page).toHaveURL(/\/login/);
  });

  test("unauthenticated admin proxy API does not expose credentials @security", async ({ request }) => {
    const response = await request.get("/api/admin/proxies");
    expect([401, 403, 503]).toContain(response.status());
    const body = await response.text();
    expect(body).not.toMatch(/"(username|password|user|pass)"\s*:/i);
  });

  test("unauthenticated refresh is rejected @security", async ({ request }) => {
    const response = await request.post("/api/admin/proxies/webshare/refresh");
    expect([401, 403, 503]).toContain(response.status());
  });

  test("provider proxy_auto patch without session is rejected @security", async ({ request }) => {
    const response = await request.patch("/api/admin/providers/00000000-0000-0000-0000-000000000000", {
      data: { proxy_auto: true }
    });
    expect([401, 403, 503]).toContain(response.status());
  });
});
