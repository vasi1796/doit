import { defineConfig, devices } from '@playwright/test'

export default defineConfig({
  testDir: './e2e',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: 'html',

  expect: {
    toHaveScreenshot: {
      maxDiffPixelRatio: 0.03,
    },
  },

  use: {
    // The app's service worker (sw.js) passes /api/ fetches through its own
    // fetch handler; Playwright cannot intercept SW-mediated requests in
    // WebKit, so whether page.route mocks apply would depend on an
    // activation race. Block SWs so mocked APIs are deterministic.
    serviceWorkers: 'block',
  },

  projects: [
    {
      name: 'webkit-desktop',
      use: {
        ...devices['Desktop Safari'],
        viewport: { width: 1280, height: 800 },
        // Pin to light so visual baselines are deterministic across
        // dev machines and CI regardless of the host OS dark-mode setting.
        colorScheme: 'light',
      },
    },
    {
      name: 'webkit-mobile',
      use: {
        ...devices['iPhone 14'],
        colorScheme: 'light',
      },
    },
  ],

  webServer: {
    command: 'npm run build && npm run preview -- --port 4173',
    port: 4173,
    reuseExistingServer: !process.env.CI,
  },
})
