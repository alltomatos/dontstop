import { test, expect, BASE_URL } from "./fixtures";

test.describe("@security endurecimento do login", () => {
  test("força bruta retorna 429 após repetidas falhas", async ({ request }) => {
    const email = process.env.E2E_RATE_LIMIT_EMAIL ?? `security-suite-${Date.now()}@example.com`;
    let blocked = false;

    for (let attempt = 0; attempt < 15; attempt++) {
      const response = await request.post(`${BASE_URL}/api/auth/login`, {
        data: { email, password: `wrong-password-${attempt}` },
        failOnStatusCode: false
      });
      expect([401, 429]).toContain(response.status());
      if (response.status() === 429) {
        blocked = true;
        expect(response.headers()["retry-after"]).toBeTruthy();
        break;
      }
    }

    expect(blocked, "o login deve bloquear após 10 falhas na mesma janela").toBe(true);
  });

  test("next externo é descartado antes de renderizar o formulário", async ({ request }) => {
    const response = await request.get(`${BASE_URL}/login?next=//evil.example`, {
      failOnStatusCode: false
    });
    const html = await response.text();
    expect(html).not.toContain("//evil.example");
  });

  test("login não redireciona para domínio externo via next", async ({ page }) => {
    const email = process.env.E2E_EMAIL;
    const password = process.env.E2E_PASSWORD;
    test.skip(!email || !password, "defina E2E_EMAIL/E2E_PASSWORD para validar o redirect autenticado");

    await page.goto("/login?next=//evil.example");
    await page.fill('input[type="email"]', email!);
    await page.fill('input[type="password"]', password!);
    await Promise.all([
      page.waitForURL(url => !url.pathname.startsWith("/login")),
      page.click('button[type="submit"]')
    ]);

    expect(new URL(page.url()).host).toBe(new URL(BASE_URL).host);
  });

  test("setup não pode ser refeito quando já concluído", async ({ request }) => {
    test.skip(process.env.E2E_SETUP_COMPLETE !== "true", "defina E2E_SETUP_COMPLETE=true no alvo já configurado");
    const response = await request.post(`${BASE_URL}/api/auth/setup`, {
      data: { name: "Attacker", email: "attacker@example.com", password: "attacker-password" },
      failOnStatusCode: false
    });
    expect(response.status()).toBe(409);
  });
});
