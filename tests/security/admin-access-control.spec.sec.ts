import { test, expect, BASE_URL } from "./fixtures";

const forgedCookies = [
  "dontstop_session=forged-session-token",
  "dontstop_admin_key=forged-admin-key"
];

test.describe("@security controle de acesso do console", () => {
  test("rotas do console redirecionam para o login sem sessão", async ({ page }) => {
    test.skip(process.env.DONTSTOP_CONSOLE_AUTH === "off", "gate de auth desligado no alvo");
    await page.context().clearCookies();
    await page.goto("/dashboard", { waitUntil: "domcontentloaded" });
    expect(new URL(page.url()).pathname).toBe("/login");
  });

  test("cookie forjado não cria API key administrativa", async ({ request }) => {
    for (const cookie of forgedCookies) {
      const response = await request.post(`${BASE_URL}/api/admin/api-keys`, {
        headers: { Cookie: cookie, "Content-Type": "application/json" },
        data: { name: "security-suite", scopes: ["manage"] },
        failOnStatusCode: false
      });
      expect([401, 403], `cookie ${cookie} não pode virar manage`).toContain(response.status());
      expect(await response.text()).not.toContain("sk-ds-");
    }
  });

  test("cookie forjado não adiciona conta a provedor", async ({ request }) => {
    for (const cookie of forgedCookies) {
      const response = await request.post(
        `${BASE_URL}/api/admin/providers/security-suite/accounts`,
        {
          headers: { Cookie: cookie, "Content-Type": "application/json" },
          data: { name: "attacker", api_key: "sk-attacker" },
          failOnStatusCode: false
        }
      );
      expect([401, 403], `cookie ${cookie} não pode escrever em providers`).toContain(response.status());
    }
  });

  test("cookie forjado não altera configuração de MCP", async ({ request }) => {
    for (const cookie of forgedCookies) {
      const response = await request.patch(`${BASE_URL}/api/admin/mcp/settings`, {
        headers: { Cookie: cookie, "Content-Type": "application/json" },
        data: { enabled: true },
        failOnStatusCode: false
      });
      expect([401, 403], `cookie ${cookie} não pode ligar MCP`).toContain(response.status());
    }
  });

  test("erros de autenticação não expõem stack trace nem segredos", async ({ request }) => {
    const response = await request.post(`${BASE_URL}/api/admin/api-keys`, {
      headers: { Cookie: forgedCookies[0], "Content-Type": "application/json" },
      data: {},
      failOnStatusCode: false
    });
    const body = await response.text();
    expect(body).not.toMatch(/goroutine|panic|node_modules|\.go:\d+/i);
    expect(body).not.toMatch(/DONTSTOP_ADMIN_API_KEY|MASTER_KEY/);
  });
});
