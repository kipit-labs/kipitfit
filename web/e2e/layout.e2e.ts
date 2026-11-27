import { expect, test } from "./fixtures";

// Nothing pushes a page wider than the phone: no sideways scrolling.
test("pages fit the screen", async ({ page, account: _ }) => {
  for (const path of ["/workout", "/routines/new", "/exercises", "/history", "/progress", "/feed", "/me"]) {
    await page.goto(path);
    await expect(page.getByRole("heading", { level: 1 })).toBeVisible();
    const [scroll, inner] = await page.evaluate(() => [document.documentElement.scrollWidth, innerWidth]);
    expect(scroll, `${path} is ${scroll}px wide on a ${inner}px screen`).toBeLessThanOrEqual(inner);
  }
});
