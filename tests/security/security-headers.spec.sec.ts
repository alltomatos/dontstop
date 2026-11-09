import { test, expect, BASE_URL } from "./fixtures";

test.describe("@security cabeçalhos do edge", () => {
  test("edge aplica HSTS, nosniff, frame-deny e referrer policy", async ({ request }) => {
    test.skip(process.env.E2E_EDGE !== "true", "defina E2E_EDGE=true para validar o Caddy");

    const response = await request.get(`${BASE_URL}/login`, { failOnStatusCode: false });
    const headers = response.headers();

    expect(headers["strict-transport-security"]).toContain("max-age=31536000");
    expect(headers["x-content-type-options"]).toBe("nosniff");
    expect(headers["x-frame-options"]).toBe("DENY");
    expect(headers["referrer-policy"]).toBe("strict-origin-when-cross-origin");
    expect(headers["server"]).toBeUndefined();
  });
});
