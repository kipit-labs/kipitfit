import { test as base, expect, request, type APIRequestContext, type Page } from "@playwright/test";

export { expect };

export const PASSWORD = "correct horse battery";
export const ADMIN = "admin@e2e.test";

// Every uncaught error in a page fails the test that caused it. Except one: WebKit reports each request still in
// flight when a test navigates away (page.goto) as "… due to access control checks", same-origin ones included.
// shortcut: matched by message; revisit if a Playwright or WebKit update changes the wording.
const cancelledByNavigation = /due to access control checks\.$/;

/** A signed-in API session for the admin, to create invites. */
export async function adminAPI(baseURL: string | undefined): Promise<APIRequestContext> {
  const api = await request.newContext({ baseURL });
  const res = await api.post("/api/auth/login", { data: { email: ADMIN, password: PASSWORD } });
  expect(res.ok(), "admin sign-in").toBeTruthy();
  return api;
}

/** An invite link, the way the admin shares it. */
export async function inviteLink(baseURL: string | undefined, email: string): Promise<string> {
  const api = await adminAPI(baseURL);
  const res = await api.post("/api/admin/invites", { data: { email } });
  expect(res.ok(), "invite").toBeTruthy();
  const { link } = await res.json();
  await api.dispose();
  return new URL(link).pathname + new URL(link).search;
}

type Account = { email: string; name: string };

type Fixtures = {
  /** A fresh account, joined through an invite link in the UI and signed in on page. */
  account: Account;
  /** Whether the project runs at phone size. */
  phone: boolean;
};

export const test = base.extend<Fixtures & { errors: string[] }>({
  errors: async ({}, use) => {
    const errors: string[] = [];
    await use(errors);
    expect(errors, "uncaught errors in the page").toEqual([]);
  },
  context: async ({ context }, use) => {
    // Tests stay offline: kipitfit calls no outside service, and nothing may start to.
    await context.route(
      (url) => url.hostname !== "localhost",
      (route) => route.fulfill({ status: 404, body: "" }),
    );
    await use(context);
  },
  page: async ({ page, errors }, use) => {
    page.on("pageerror", (e) => {
      if (!cancelledByNavigation.test(e.message)) errors.push(e.message);
    });
    await use(page);
  },
  phone: async ({ isMobile }, use) => use(isMobile),
  account: async ({ page, baseURL }, use) => {
    const email = `e2e-${crypto.randomUUID()}@e2e.test`;
    const name = "Ana";
    await page.goto(await inviteLink(baseURL, email));
    await page.getByLabel("Name").fill(name);
    await page.getByLabel("Password").fill(PASSWORD);
    await page.getByRole("button", { name: "Create account" }).click();
    await expect(tab(page, "Workout")).toBeVisible();
    await use({ email, name });
  },
});

/** The bottom tab bar. */
export function tab(page: Page, name: "Feed" | "Workout" | "Progress" | "Me") {
  return page.getByRole("navigation").getByRole("link", { name, exact: true });
}
