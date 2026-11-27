import { request, type FullConfig } from "@playwright/test";

import { SETUP_TOKEN } from "../playwright.config";
import { ADMIN, PASSWORD } from "./fixtures";

// Creates the admin on the run's fresh database; tests get their accounts from the admin's invites.
export default async function globalSetup(config: FullConfig) {
  const api = await request.newContext({ baseURL: config.projects[0].use.baseURL });
  const res = await api.post("/api/auth/setup", {
    data: { setupToken: SETUP_TOKEN, email: ADMIN, name: "Admin", password: PASSWORD },
  });
  if (!res.ok()) throw new Error(`setup: ${res.status()} ${await res.text()}`);
  await api.dispose();
}
