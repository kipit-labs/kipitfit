import type { Page } from "@playwright/test";

import { expect, tab, test } from "./fixtures";

/** Adds one exercise through the picker, the routine editor's or the live workout's. */
async function addExercise(page: Page, name: string, muscle: string) {
  await page.getByRole("button", { name: "Add exercises" }).click();
  const picker = page.getByRole("dialog", { name: "Add exercises" });
  await picker.getByRole("searchbox", { name: "Search exercises" }).fill(name);
  await picker.getByRole("button", { name: new RegExp(`^${name} ${muscle} ·`) }).click();
  await picker.getByRole("button", { name: "Add 1 exercise" }).click();
  await expect(picker).toBeHidden();
}

test("plan a routine, then train it live from start to finish", async ({ page, account: _ }) => {
  await page.getByRole("link", { name: "New routine" }).click();
  await page.getByLabel("Routine name").fill("Leg day");
  await addExercise(page, "Barbell Squat", "Quadriceps");
  await page.getByLabel("Weight, set 1").fill("100");
  await page.getByLabel("Reps, set 1").fill("5");
  await page.getByRole("button", { name: "Add set" }).click();
  await page.getByRole("button", { name: "Save" }).click();

  const routine = page.getByRole("link", { name: /^Leg day/ });
  await expect(routine).toContainText("Barbell Squat");
  await page.getByRole("button", { name: "Start routine" }).click();

  // At the gym: log the sets the routine planned, rest between them.
  await expect(page.getByRole("heading", { name: "Leg day" })).toBeVisible();
  const sets = page.getByRole("table", { name: "Barbell Squat" });
  await sets.getByLabel("Weight, set 1").fill("100");
  await sets.getByLabel("Reps, set 1").fill("5");
  await sets.getByRole("button", { name: "Set 1 done" }).click();
  await expect(page.getByRole("timer", { name: "Rest" })).toBeVisible();
  await page.getByRole("button", { name: "Skip" }).click();
  await expect(page.getByRole("timer", { name: "Rest" })).toBeHidden();
  await sets.getByLabel("Weight, set 2").fill("100");
  await sets.getByLabel("Reps, set 2").fill("6");
  await sets.getByRole("button", { name: "Set 2 done" }).click();
  await expect(page.getByRole("definition").filter({ hasText: "1,100 kg" })).toBeVisible();

  // Leaving the screen keeps the workout going.
  await page.getByRole("button", { name: "Back to the tabs" }).click();
  await page.getByRole("link", { name: /^Leg day .* Resume$/ }).click();
  await expect(sets.getByLabel("Reps, set 2")).toHaveValue("6");

  await page.getByRole("button", { name: "Finish" }).click();
  const finish = page.getByRole("dialog", { name: "Finish workout" });
  await expect(finish.getByLabel("Title")).toHaveValue("Leg day");
  await finish.getByRole("button", { name: "Save workout" }).click();
  await expect(page.getByText("Workout saved")).toBeVisible();
  await expect(routine).toContainText("Last done today");

  await page.getByRole("link", { name: /^History/ }).click();
  await page.getByRole("link", { name: /^Leg day/ }).click();
  await expect(page.getByRole("heading", { name: "Leg day" })).toBeVisible();
  await expect(page.getByRole("listitem").filter({ hasText: "100 kg × 6" })).toBeVisible();

  await tab(page, "Progress").click();
  await expect(page.getByRole("listitem").filter({ hasText: "Quadriceps" })).toContainText("2");
});

test("a heavier set beats the record", async ({ page, account: _ }) => {
  for (const kg of ["100", "110"]) {
    await tab(page, "Workout").click();
    await page.getByRole("button", { name: "Start empty workout" }).click();
    await addExercise(page, "Barbell Squat", "Quadriceps");
    await page.getByLabel("Weight, set 1").fill(kg);
    await page.getByLabel("Reps, set 1").fill("5");
    await page.getByRole("button", { name: "Set 1 done" }).click();
    await page.getByRole("button", { name: "Finish" }).click();
    await page.getByRole("dialog", { name: "Finish workout" }).getByRole("button", { name: "Save workout" }).click();
    await expect(page.getByText("Workout saved").first()).toBeVisible();
  }

  await page.getByRole("link", { name: /^Exercise library/ }).click();
  await page.getByRole("searchbox", { name: "Search exercises" }).fill("Barbell Squat");
  await page.getByRole("link", { name: /^Barbell Squat Quadriceps/ }).click();
  await page.getByRole("tab", { name: "Records" }).click();
  await expect(page.getByRole("tabpanel", { name: "Records" })).toContainText("110 kg");
});

test("a discarded workout leaves nothing behind", async ({ page, account: _ }) => {
  await page.getByRole("button", { name: "Start empty workout" }).click();
  await addExercise(page, "Barbell Squat", "Quadriceps");
  await page.getByRole("button", { name: "Discard workout" }).click();
  const confirm = page.getByRole("alertdialog");
  await confirm.getByRole("button", { name: /Discard/ }).click();
  await expect(page.getByRole("heading", { name: "Workout" })).toBeVisible();
  await expect(page.getByRole("link", { name: /Resume$/ })).toHaveCount(0);
  await page.getByRole("link", { name: /^History/ }).click();
  await expect(page.getByText("No workouts yet")).toBeVisible();
});
