import { defineConfig, devices } from '@playwright/test'

/**
 * End-to-end tests against a running stack (PostgreSQL + Go API + Vite). They do
 * not start servers or create users themselves; see docs/DEVELOPMENT_SETUP.md
 * ("End-to-end browser tests"). Uses the locally installed Google Chrome, so no
 * browser download is needed.
 */
export default defineConfig({
  testDir: './e2e',
  testMatch: '**/*.e2e.ts',
  // Tests share one API (sessions, login throttling), so run them one at a time.
  workers: 1,
  fullyParallel: false,
  retries: 0,
  reporter: [['list']],
  outputDir: './test-results',
  use: {
    baseURL: process.env.E2E_BASE_URL ?? 'http://localhost:5173',
    channel: 'chrome',
    headless: true,
    screenshot: 'only-on-failure',
    trace: 'retain-on-failure',
  },
  projects: [
    { name: 'desktop', use: { viewport: { width: 1366, height: 900 } } },
    { name: 'mobile', use: { ...devices['Pixel 7'], channel: 'chrome' } },
  ],
})
