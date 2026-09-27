import { defineConfig, devices } from '@playwright/test';

const apiBase = process.env.API_BASE_URL ?? 'http://127.0.0.1:8080';
const webBase = process.env.WEB_BASE_URL ?? 'http://127.0.0.1:3000';

export default defineConfig({
  testDir: './e2e',
  timeout: 60_000,
  fullyParallel: false,
  retries: 0,
  reporter: 'list',
  use: {
    baseURL: webBase,
    extraHTTPHeaders: {
      Accept: 'application/json',
    },
  },
  projects: [
    {
      name: 'api-flows',
      testMatch: /api-flows\.spec\.ts/,
      use: {
        baseURL: apiBase,
      },
    },
    {
      name: 'ui-flows',
      testMatch: /ui-flows\.spec\.ts/,
      use: {
        ...devices['Desktop Chrome'],
        baseURL: webBase,
      },
    },
  ],
});
