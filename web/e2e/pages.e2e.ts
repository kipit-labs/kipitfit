import type { Locator, Page } from "@playwright/test";

import { addExercise, ADMIN, expect, inviteLink, PASSWORD, tab, test } from "./fixtures";

// Every page and every tab, opened once: a render error shows the router's error page instead of throwing, so the
// page fixture's uncaught-error check alone wouldn't see it. The journeys test what pages do; this, that they open.

/** Waits for where to show and the page's loading to settle, then checks the page didn't crash rendering it. */
async function renders(page: Page, where: Locator) {
  await expect(where).toBeVisible();
  await expect(page.getByRole("status", { name: "Loading" })).toHaveCount(0);
  await expect(page.getByRole("heading", { name: "Unexpected Application Error!" })).toHaveCount(0);
}

/** Opens every tab of the page's tab list, in order. */
async function everyTab(page: Page) {
  const tabs = page.getByRole("main").getByRole("tablist").getByRole("tab");
  await expect(tabs.first()).toBeVisible();
  for (const name of await tabs.allInnerTexts()) {
    await tabs.filter({ hasText: name }).click();
    await renders(page, page.getByRole("tabpanel", { name, exact: true }));
  }
}

const title = (page: Page) => page.getByRole("main").getByRole("heading", { level: 1 });

test("every page and tab opens, with a routine trained", async ({ page, account: _ }) => {
  // The data the pages show: a routine, trained live with a set logged, saved.
  await page.goto("/routines/new");
  await renders(page, title(page));
  await page.getByLabel("Routine name").fill("Leg day");
  await addExercise(page, "Barbell Squat", "Quadriceps");
  await page.getByLabel("Weight, set 1").fill("100");
  await page.getByLabel("Reps, set 1").fill("5");
  await page.getByRole("button", { name: "Save" }).click();
  await page.getByRole("button", { name: "Start routine" }).click();
  await renders(page, page.getByRole("heading", { name: "Leg day", level: 1 }));
  await page.getByRole("button", { name: "Set 1 done" }).click();
  await renders(page, page.getByRole("timer", { name: "Rest" }));
  await page.getByRole("button", { name: "Finish" }).click();
  await page.getByRole("dialog", { name: "Finish workout" }).getByRole("button", { name: "Save workout" }).click();
  await expect(page.getByText("Workout saved")).toBeVisible();

  for (const path of ["/feed", "/workout", "/exercises", "/history", "/progress", "/me"]) {
    await page.goto(path);
    await renders(page, title(page));
  }
  await page.goto("/workout");
  await page.getByRole("link", { name: /^Leg day/ }).click();
  await renders(page, page.getByLabel("Routine name"));
  await expect(page.getByLabel("Routine name")).toHaveValue("Leg day");

  // From the history to the workout, from it to its exercise.
  await page.goto("/history");
  await page.getByRole("link", { name: /^Leg day/ }).click();
  await renders(page, page.getByRole("heading", { name: "Leg day", level: 1 }));
  await page.getByRole("link", { name: /^Barbell Squat/ }).click();
  await renders(page, page.getByRole("heading", { name: "Barbell Squat", level: 1 }));
  await everyTab(page);
});

test("the signed-out pages and the admin's profile open", async ({ page, baseURL }) => {
  const email = `e2e-${crypto.randomUUID()}@e2e.test`;
  await page.goto(await inviteLink(baseURL, email));
  await renders(page, page.getByRole("heading", { name: "Join kipitfit" }));
  await page.goto("/");
  await renders(page, page.getByRole("heading", { name: "Sign in" }));

  await page.getByLabel("Email").fill(ADMIN);
  await page.getByLabel("Password").fill(PASSWORD);
  await page.getByRole("button", { name: "Sign in" }).click();
  await tab(page, "Me").click();
  // The invites the admin made.
  await renders(page, page.getByRole("main").getByText(email));
});
