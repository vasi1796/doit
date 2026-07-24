import { test, expect } from '@playwright/test'
import { mockApi } from './helpers/mock-api'

test.describe('Default landing route', () => {
  test('root redirects to Today', async ({ page }) => {
    await mockApi(page)
    await page.goto('/')
    await page.waitForURL('**/today')
    await expect(page.locator('main h1').first()).toHaveText('Today')
  })

  test('unknown route redirects to Today', async ({ page }) => {
    await mockApi(page)
    await page.goto('/no-such-page')
    await page.waitForURL('**/today')
    await expect(page.locator('main h1').first()).toHaveText('Today')
  })
})
