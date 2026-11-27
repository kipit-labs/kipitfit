import { expect, test } from "./fixtures";

test("browse the library and create an exercise of your own", async ({ page, account: _ }) => {
  await page.getByRole("link", { name: /^Exercise library/ }).click();
  await page.getByRole("searchbox", { name: "Search exercises" }).fill("Barbell Squat");
  await page.getByRole("link", { name: /^Barbell Squat Quadriceps/ }).click();
  await expect(page.getByRole("heading", { name: "Barbell Squat" })).toBeVisible();
  await expect(page.getByRole("tabpanel", { name: "Summary" })).toContainText("Quadriceps");
  await page.getByRole("button", { name: "Exercises" }).click();

  await page.getByRole("button", { name: "New exercise" }).click();
  const form = page.getByRole("dialog", { name: "New exercise" });
  await form.getByLabel("Name").fill("Landmine Press");
  await form.getByRole("button", { name: "Create" }).click();
  await expect(form).toBeHidden();
  await expect(page.getByRole("heading", { name: "Landmine Press" })).toBeVisible();
  await expect(page.getByRole("main")).toContainText("Custom");

  await page.getByRole("button", { name: "Exercises" }).click();
  await page.getByRole("searchbox", { name: "Search exercises" }).fill("Landmine");
  await expect(page.getByRole("link", { name: /^Landmine Press/ })).toBeVisible();
});
