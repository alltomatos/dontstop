import { test, expect, type Page } from "@playwright/test";

const FRESH = process.env.E2E_FRESH_INSTANCE === "true";
const EMAIL = process.env.E2E_SETUP_EMAIL ?? `qa-owner-${Date.now()}@example.com`;
const PASSWORD = process.env.E2E_SETUP_PASSWORD ?? "qa-password-123";

test.describe.configure({ mode: "serial" });

async function fillAccountStep(page: Page) {
  await page.getByRole("radio", { name: "English" }).check();
  await page.getByRole("button", { name: "Continue" }).click();
  await page.locator('input[autocomplete="name"]').fill("QA Owner");
  await page.locator('input[type="email"]').fill(EMAIL);
  await page.locator('input[autocomplete="new-password"]').fill(PASSWORD);
  await page.getByRole("button", { name: "Create account" }).click();
}

test("TC-ONB-01 — onboarding leva do idioma ao dashboard pulando provedor", async ({ page }) => {
  test.skip(!FRESH, "requer instância sem owner (E2E_FRESH_INSTANCE=true)");

  await page.goto("/onboarding");
  await expect(page.getByRole("heading", { name: "Choose your language" })).toBeVisible();
  await fillAccountStep(page);

  await expect(page.getByRole("heading", { name: "Connect your first provider" })).toBeVisible();
  await page.getByRole("button", { name: "Skip" }).first().click();

  await expect(page).toHaveURL(/\/dashboard/);
  await expect(page.getByText("Dashboard").first()).toBeVisible();
});

test("TC-ONB-02 — onboarding conecta o primeiro provedor", async ({ page }) => {
  test.skip(!FRESH, "requer instância sem owner (E2E_FRESH_INSTANCE=true)");

  await page.goto("/onboarding");
  await fillAccountStep(page);
  await expect(page.getByRole("heading", { name: "Connect your first provider" })).toBeVisible();

  await page.getByRole("button", { name: "OA OpenAI API key", exact: true }).click();
  await page.getByLabel("Provider name").fill("openai-qa");
  await page.getByLabel("API key").fill("sk-test-fake-key");
  await page.getByRole("button", { name: "Connect provider" }).click();

  await expect(page).toHaveURL(/\/dashboard/);
  await page.goto("/providers");
  await expect(page.getByText("OpenAI").first()).toBeVisible();
});

test("TC-ONB-03 — senha curta não avança do passo 2", async ({ page }) => {
  test.skip(!FRESH, "requer instância sem owner (E2E_FRESH_INSTANCE=true)");

  await page.goto("/onboarding");
  await page.getByRole("radio", { name: "English" }).check();
  await page.getByRole("button", { name: "Continue" }).click();
  await page.locator('input[autocomplete="name"]').fill("QA Owner");
  await page.locator('input[type="email"]').fill(EMAIL);
  await page.locator('input[autocomplete="new-password"]').fill("short");
  await page.getByRole("button", { name: "Create account" }).click();

  await expect(page.getByRole("heading", { name: "Create your account" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Connect your first provider" })).toHaveCount(0);
});

test("TC-ONB-04 — voltar do passo 3 e reenviar a conta retoma o passo 3", async ({ page }) => {
  test.skip(!FRESH, "requer instância sem owner (E2E_FRESH_INSTANCE=true)");

  await page.goto("/onboarding");
  await fillAccountStep(page);
  await expect(page.getByRole("heading", { name: "Connect your first provider" })).toBeVisible();

  await page.getByRole("button", { name: "Back" }).click();
  await expect(page.getByRole("heading", { name: "Create your account" })).toBeVisible();
  await page.getByRole("button", { name: "Create account" }).click();

  await expect(page.getByRole("heading", { name: "Connect your first provider" })).toBeVisible();
});

test("TC-ONB-05 — recarregar o wizard depois do setup redireciona para o login", async ({ page }) => {
  test.skip(FRESH, "só faz sentido com o setup já concluído");

  await page.goto("/onboarding");
  await expect(page).toHaveURL(/\/login/);
});

test("TC-UI-01 — topbar sem tenant/produção e rodapé neutro", async ({ page }) => {
  test.skip(FRESH, "só faz sentido com o setup concluído");
  const email = process.env.E2E_EMAIL ?? EMAIL;
  const password = process.env.E2E_PASSWORD ?? PASSWORD;
  test.skip(!email || !password, "defina E2E_EMAIL/E2E_PASSWORD");

  await page.goto("/login");
  await page.locator('input[type="email"]').fill(email);
  await page.locator('input[type="password"]').fill(password);
  await page.getByRole("button", { name: "Sign in" }).click();
  await expect(page).toHaveURL(/\/dashboard/);
  await expect(page.locator(".who")).not.toContainText(/tenant|production|produção/i);
  await expect(page.locator(".sidebar-foot")).toHaveText("dontstop · v0.1");
});

test("TC-ONB-06 — login com credenciais válidas abre o console", async ({ page }) => {
  test.skip(FRESH, "só faz sentido com o owner já criado");

  const email = process.env.E2E_EMAIL ?? EMAIL;
  const password = process.env.E2E_PASSWORD ?? PASSWORD;
  test.skip(!email || !password, "defina E2E_EMAIL/E2E_PASSWORD");

  await page.goto("/login");
  await page.locator('input[type="email"]').fill(email);
  await page.locator('input[type="password"]').fill(password);
  await page.getByRole("button", { name: "Sign in" }).click();
  await expect(page).toHaveURL(/\/dashboard/);
});
