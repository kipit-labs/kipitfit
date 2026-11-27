import { defineConfig, devices } from "@playwright/test";

const port = 8092;
export const SETUP_TOKEN = "e2e-setup-token";

// UI tests drive the real app (Go binary, embedded build, SQLite) in a real browser: WebKit at phone size, as on the
// iPhone, and Chromium at desktop size. Run them with `make e2e`, which builds bin/kipitfit first. Each run starts the
// server on a fresh database, which is the cleanup: kipitfit has no account deletion.
export default defineConfig({
  testDir: "e2e",
  testMatch: "*.e2e.ts",
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: 0, // a flaky test is a bug to fix, not to retry
  reporter: process.env.CI ? [["list"], ["html", { open: "never" }]] : "list",
  globalSetup: "./e2e/global-setup.ts",
  use: {
    baseURL: `http://localhost:${port}`,
    trace: "retain-on-failure",
  },
  projects: [
    { name: "iphone", use: { ...devices["iPhone 13"] } },
    { name: "desktop", use: { ...devices["Desktop Chrome"] } },
  ],
  webServer: {
    command: "rm -rf e2e-data && exec ../bin/kipitfit",
    url: `http://localhost:${port}/api/health`,
    reuseExistingServer: false,
    timeout: 120_000,
    env: {
      KIPITFIT_ADDR: `:${port}`,
      KIPITFIT_DATA_DIR: "e2e-data",
      KIPITFIT_APP_URL: `http://localhost:${port}`,
      KIPITFIT_SETUP_TOKEN: SETUP_TOKEN,
      KIPITFIT_S3_ENDPOINT: "",
    },
  },
});
