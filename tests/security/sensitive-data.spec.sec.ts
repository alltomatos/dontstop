import { test, expect, BASE_URL } from "./fixtures";

test.describe("@security exposição de dados sensíveis", () => {
  test("login não devolve o token no corpo e usa cookie HttpOnly/SameSite", async ({ request }) => {
    const email = process.env.E2E_EMAIL;
    const password = process.env.E2E_PASSWORD;
    test.skip(!email || !password, "defina E2E_EMAIL/E2E_PASSWORD");

    const response = await request.post(`${BASE_URL}/api/auth/login`, {
      data: { email, password },
      failOnStatusCode: false
    });
    expect(response.status()).toBe(204);
    expect(await response.text()).toBe("");

    const setCookie = response
      .headersArray()
      .filter(header => header.name.toLowerCase() === "set-cookie")
      .map(header => header.value)
      .join("; ");
    expect(setCookie).toContain("dontstop_session=");
    expect(setCookie.toLowerCase()).toContain("httponly");
    expect(setCookie.toLowerCase()).toContain("samesite=strict");
  });

  test("respostas administrativas não vazam credenciais de provedor", async ({ request }) => {
    const email = process.env.E2E_EMAIL;
    const password = process.env.E2E_PASSWORD;
    test.skip(!email || !password, "defina E2E_EMAIL/E2E_PASSWORD");

    const login = await request.post(`${BASE_URL}/api/auth/login`, {
      data: { email, password },
      failOnStatusCode: false
    });
    expect(login.status()).toBe(204);

    const providers = await request.get(`${BASE_URL}/api/admin/providers`, {
      failOnStatusCode: false
    });
    if (providers.status() !== 200) return;

    const body = await providers.text();
    expect(body).not.toMatch(/sk-(ds|ant|proj)-[A-Za-z0-9_-]{8,}/);
    expect(body).not.toMatch(/"api_key"\s*:\s*"[^"]+"/i);
  });
});
