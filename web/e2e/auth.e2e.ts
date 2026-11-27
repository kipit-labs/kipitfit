import { expect, inviteLink, PASSWORD, tab, test } from "./fixtures";

test("a member joins through an invite, signs out and back in", async ({ page, account }) => {
  await tab(page, "Me").click();
  await expect(page.getByLabel("Name")).toHaveValue(account.name);
  await page.getByRole("button", { name: "Sign out" }).click();

  await expect(page.getByRole("heading", { name: "Sign in" })).toBeVisible();
  await page.getByLabel("Email").fill(account.email);
  await page.getByLabel("Password").fill("not the password");
  await page.getByRole("button", { name: "Sign in" }).click();
  await expect(page.getByRole("alert")).toHaveText("Wrong email or password.");

  await page.getByLabel("Password").fill(PASSWORD);
  await page.getByRole("button", { name: "Sign in" }).click();
  // Back where the member signed out.
  await expect(page.getByRole("heading", { name: "Me" })).toBeVisible();
});

test("an invite link works once", async ({ page, baseURL }) => {
  const link = await inviteLink(baseURL, `e2e-${crypto.randomUUID()}@e2e.test`);
  await page.goto(link);
  await page.getByLabel("Name").fill("Bo");
  await page.getByLabel("Password").fill(PASSWORD);
  await page.getByRole("button", { name: "Create account" }).click();
  await expect(tab(page, "Workout")).toBeVisible();

  await page.context().clearCookies();
  await page.goto(link);
  await page.getByLabel("Name").fill("Eve");
  await page.getByLabel("Password").fill(PASSWORD);
  await page.getByRole("button", { name: "Create account" }).click();
  await expect(page.getByRole("alert")).toBeVisible();
  await expect(tab(page, "Workout")).toHaveCount(0);
});

test("the profile keeps its weight unit", async ({ page, account: _ }) => {
  await tab(page, "Me").click();
  await page.getByRole("radio", { name: "lb" }).click();
  await expect(page.getByRole("radio", { name: "lb" })).toBeChecked();
  await page.reload();
  await expect(page.getByRole("radio", { name: "lb" })).toBeChecked();
});
