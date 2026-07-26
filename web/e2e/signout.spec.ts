import { test, expect } from '@playwright/test'
import { mockApi } from './helpers/mock-api'

test.describe('Sign out', () => {
  test('wipes local data and lands on the login page', async ({ page, isMobile }) => {
    test.skip(isMobile, 'sidebar sign-out is the desktop entry point')

    await mockApi(page)
    await page.route('**/auth/logout', (route) => route.fulfill({ json: { status: 'logged out' } }))
    await page.goto('/today')
    await page.getByRole('link', { name: 'Project Alpha' }).waitFor({ state: 'visible' })

    await page.getByRole('button', { name: 'Sign out' }).click()

    await page.waitForURL('**/login')
    const hasDoitDb = await page.evaluate(async () => {
      const dbs = await indexedDB.databases()
      return dbs.some((d) => d.name === 'doit')
    })
    expect(hasDoitDb).toBe(false)
  })
})
